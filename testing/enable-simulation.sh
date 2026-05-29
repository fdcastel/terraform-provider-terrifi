#!/usr/bin/env bash
# testing/enable-simulation.sh
#
# Enable UniFi Network's built-in simulation mode on a running uosserver,
# synthesizing fake devices in mongo so tests that need an adopted gateway
# (firewall zones, firewall policies, AP groups, port profiles) become
# runnable. Same `is_simulation=true` trick the docker-based acceptance
# target uses against `linuxserver/unifi-network-application` — applied here
# to the first-party UOS Server.
#
# Idempotent: re-running strips prior simulation flags from system.properties
# and re-appends them with the current NUM_* values.
#
# Requirements:
#   - Run as root on the host where uosserver is installed.
#   - bootstrap.sh has already provisioned the admin account.
#
# Env overrides:
#   NUM_UGW      (default 1)
#   NUM_USW      (default 2)
#   NUM_UAP      (default 3)
#   UOS_URL      (default https://127.0.0.1:11443)
#   ADMIN_USER   (default admin)
#   ADMIN_PASS   (default TerrifiTest!2026)
#
# Stdout: a single line `UNIFI_FAKE_DEVICES=<count>` for CI assertion.
# Stderr: progress logs.

set -euo pipefail

NUM_UGW="${NUM_UGW:-1}"
NUM_USW="${NUM_USW:-2}"
NUM_UAP="${NUM_UAP:-3}"
UOS_URL="${UOS_URL:-https://127.0.0.1:11443}"
ADMIN_USER="${ADMIN_USER:-admin}"
ADMIN_PASS="${ADMIN_PASS:-TerrifiTest!2026}"

SP_HOST=/home/uosserver/.local/share/containers/storage/volumes/uosserver_var_lib_unifi/_data/system.properties

log() { printf >&2 '[enable-simulation] %s\n' "$*"; }

[ -f "$SP_HOST" ] || { log "ERROR: $SP_HOST not found — has uosserver been installed?"; exit 1; }

log "stopping uosserver"
systemctl stop uosserver
# Wait until the container is fully exited; podman state transitions through "stopping".
for i in $(seq 1 30); do
  state=$(sudo -iu uosserver podman ps -a --format '{{.State}}' --filter name=uosserver 2>/dev/null || echo "")
  case "$state" in
    exited|created|"") break ;;
  esac
  sleep 1
done

log "stripping any prior simulation flags from $SP_HOST"
sed -i '/^is_simulation=/d; /^demo\.num_/d' "$SP_HOST"

log "appending is_simulation=true with $NUM_UGW UGW / $NUM_USW USW / $NUM_UAP UAP"
cat >> "$SP_HOST" <<EOF
is_simulation=true
demo.num_ugw=$NUM_UGW
demo.num_usw=$NUM_USW
demo.num_uap=$NUM_UAP
EOF

log "starting uosserver"
systemctl start uosserver

log "waiting for controller HTTPS up"
deadline=$(( $(date +%s) + 180 ))
until curl -fsSk -o /dev/null "$UOS_URL/api/ping"; do
  [ "$(date +%s)" -lt "$deadline" ] || { log "ERROR: controller did not come up in 3 min"; exit 1; }
  sleep 1
done

# Network app behind /proxy/network warms up a few seconds after /api/ping returns 204.
log "waiting for legacy API"
deadline=$(( $(date +%s) + 90 ))
while :; do
  http=$(curl -sk -o /dev/null -w '%{http_code}' "$UOS_URL/proxy/network/api/s/default/stat/device" || echo 0)
  case "$http" in
    200|401) break ;;
  esac
  [ "$(date +%s)" -lt "$deadline" ] || { log "ERROR: legacy API didn't come up in 90 s (last: $http)"; exit 1; }
  sleep 2
done

log "logging in"
COOKIE=$(mktemp)
trap 'rm -f "$COOKIE"' EXIT
curl -fsSk -c "$COOKIE" -o /dev/null \
  -X POST -H 'Content-Type: application/json' \
  -d "$(jq -nc --arg u "$ADMIN_USER" --arg p "$ADMIN_PASS" '{username:$u,password:$p,rememberMe:false}')" \
  "$UOS_URL/api/auth/login"
TOKEN=$(awk '/^#HttpOnly_.*TOKEN/ {print $NF}' "$COOKIE")
PAYLOAD=$(printf '%s' "$TOKEN" | cut -d. -f2 | tr '_-' '/+')
while [ $((${#PAYLOAD} % 4)) -ne 0 ]; do PAYLOAD="${PAYLOAD}="; done
CSRF=$(printf '%s' "$PAYLOAD" | base64 -d 2>/dev/null | jq -r .csrfToken)

expected=$(( NUM_UGW + NUM_USW + NUM_UAP ))

log "polling for $expected synthetic devices"
deadline=$(( $(date +%s) + 60 ))
count=0
while :; do
  count=$(curl -fsSk -b "$COOKIE" "$UOS_URL/proxy/network/api/s/default/stat/device" \
          | jq '.data | length' 2>/dev/null || echo 0)
  [ "$count" -ge "$expected" ] && break
  [ "$(date +%s)" -lt "$deadline" ] || break
  sleep 2
done

if [ "$count" -lt "$expected" ]; then
  log "ERROR: expected $expected devices, got $count after 60 s"
  printf 'UNIFI_FAKE_DEVICES=%d\n' "$count"
  exit 1
fi

log "adopting all $count devices"
curl -fsSk -b "$COOKIE" "$UOS_URL/proxy/network/api/s/default/stat/device" > /tmp/.dev.$$
trap 'rm -f /tmp/.dev.$$ "$COOKIE"' EXIT
for mac in $(jq -r '.data[] | select(.adopted | not) | .mac' /tmp/.dev.$$); do
  http=$(curl -sk -b "$COOKIE" -H "X-CSRF-Token: $CSRF" -H 'Content-Type: application/json' \
    -X POST -o /dev/null -w '%{http_code}' \
    -d "$(jq -nc --arg m "$mac" '{cmd:"adopt",mac:$m}')" \
    "$UOS_URL/proxy/network/api/s/default/cmd/devmgr")
  log "adopt $mac -> HTTP $http"
done

# Poll until every device is reported as adopted. Simulation devices step
# through state 2 (pending) → state 1 (connected) at varying rates; some can
# take 30+ s to reach state 1 even after the adopt command returns 200.
log "waiting for all $count devices to settle into adopted=true"
deadline=$(( $(date +%s) + 90 ))
while :; do
  all_adopted=$(curl -fsSk -b "$COOKIE" "$UOS_URL/proxy/network/api/s/default/stat/device" \
                | jq '[.data[] | .adopted] | all')
  pending=$(curl -fsSk -b "$COOKIE" "$UOS_URL/proxy/network/api/s/default/stat/device" \
            | jq '[.data[] | select(.adopted | not)] | length')
  [ "$all_adopted" = "true" ] && break
  if [ "$(date +%s)" -ge "$deadline" ]; then
    log "WARN: $pending device(s) still unadopted after 90 s — re-issuing adopt"
    curl -fsSk -b "$COOKIE" "$UOS_URL/proxy/network/api/s/default/stat/device" \
      | jq -r '.data[] | select(.adopted | not) | .mac' \
      | while read mac; do
          curl -sk -b "$COOKIE" -H "X-CSRF-Token: $CSRF" -H 'Content-Type: application/json' \
            -X POST -o /dev/null -w "re-adopt $mac %{http_code}\n" \
            -d "$(jq -nc --arg m "$mac" '{cmd:"adopt",mac:$m}')" \
            "$UOS_URL/proxy/network/api/s/default/cmd/devmgr"
        done >&2
    deadline=$(( $(date +%s) + 60 ))
    continue
  fi
  sleep 3
done
log "all $count devices adopted"

printf 'UNIFI_FAKE_DEVICES=%d\n' "$count"
