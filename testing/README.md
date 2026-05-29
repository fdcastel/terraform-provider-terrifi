# testing/

Bootstrap scripts for the `uos` acceptance-test target — a third option
alongside the existing `docker` (default) and `hardware` targets. UOS Server
is Ubiquiti's first-party packaging of the Network application; using it
exercises code paths the docker simulation (`linuxserver/unifi-network-application`)
does not, particularly around firewall zones/policies, WLAN, and client groups.

The scripts are designed to run **as root on a Linux host where uosserver is
already installed** (LXC, VM, or bare metal). They do not provision the host
itself or install UOS Server; that step is out of scope here (typically a
GitHub Actions step or a one-off `pveam` template + `pct create`).

## Scripts

- `bootstrap.sh` — Walks a fresh UOS Server through first-run setup, creates
  an integration API key on the admin account, and prints
  `UNIFI_API=…`, `UNIFI_API_KEY=…`, `UNIFI_INSECURE=true` on stdout. Idempotent
  — re-running on an already-set-up controller skips the setup step. Pass
  `ENABLE_FAKE_DEVICES=1` to chain into `enable-simulation.sh`.

- `enable-simulation.sh` — Stops uosserver, edits `system.properties` to set
  `is_simulation=true` + `demo.num_ugw`/`usw`/`uap`, restarts uosserver, and
  polls + adopts the synthesized devices via the Network API. Required for
  any test that depends on an adopted gateway or AP (firewall zones, AP
  groups, port profiles, traffic routes). Idempotent.

- `verify-devices.sh` — Post-condition: asserts the controller has the
  expected number of adopted devices. Useful in CI to fail fast if
  simulation didn't fully synthesize.

- `reset-controller.sh` — Wipes the Network app's data volume (mongo,
  `system.properties`, keystore) and restarts uosserver. Preserves UOS-level
  state (admin account, API keys). Use between acceptance-test runs to undo
  whatever state previous tests left behind. After it returns, run
  `bootstrap.sh` again to obtain a fresh integration API key (the previous
  key references the now-wiped Network site and may not work).

## Typical use

```sh
# Once per host, after UOS Server is installed:
eval "$(ENABLE_FAKE_DEVICES=1 sudo testing/bootstrap.sh)"

# Then on the test runner (which can be the same host or a different one
# that can reach UOS over the network):
export UNIFI_SITE=default
task test:acc:uos
```

The three env vars (`UNIFI_API`, `UNIFI_API_KEY`, `UNIFI_INSECURE`) are the
same ones the existing `hardware` target consumes, so the same provider code
runs against either.

## Fast local dev loop: Proxmox LXC snapshots

If your UOS Server is hosted in a Proxmox LXC and the storage backend is ZFS
or any other snapshot-capable driver, snapshot/rollback is dramatically
faster than running `reset-controller.sh + bootstrap.sh + enable-simulation.sh`
between test runs. The scripts here remain the canonical CI path (no
Proxmox dependency); snapshots are a local-dev convenience.

```sh
# One-time, after the initial bootstrap.sh + ENABLE_FAKE_DEVICES=1:
pct snapshot <VMID> clean --description "uosserver bootstrapped + 6 simulated devices adopted"

# Between test runs:
pct rollback <VMID> clean   # ~2 s; controller comes back already-bootstrapped

# To update the baseline (e.g. after upgrading uosserver):
pct snapshot <VMID> clean-$(date +%Y%m%d) --description "..."  # keep history
pct delsnapshot <VMID> clean     # only after the new one is proven
pct snapshot <VMID> clean
```

The CI workflow (§8 L05) will not use snapshots — it installs UOS fresh on
each run via `bootstrap.sh`. The local dev loop is the only place this
matters.

## Defaults

- Admin credentials: `admin` / `TerrifiTest!2026` — override via `ADMIN_USER`
  and `ADMIN_PASS` if your environment requires it.
- API key name: `terrifi-test` — repeated bootstrap runs append a timestamp
  suffix rather than clobbering prior keys.
- Simulation device counts: 1 UGW, 2 USW, 3 UAP — override via `NUM_UGW`,
  `NUM_USW`, `NUM_UAP`.
