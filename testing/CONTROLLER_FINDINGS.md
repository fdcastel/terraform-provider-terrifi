# Controller findings catalog (F-001 … F-017)

The mock-controller investigation behind the `uos` test target catalogued 14
controller / provider findings (F-001 … F-014). That investigation was run
against the upstream `terraform-provider-unifi` provider, so each finding has
to be re-mapped to terrifi, which has a different (and smaller) resource set
and owns its own HTTP layer. F-015 was added later, surfaced while landing
E02 (network: dhcp_boot_* / domain_name / multicast_dns). F-016 was
surfaced on 2026-06-02 while importing an existing UDM-Pro site's DHCP
reservations into `terrifi_client_device`.

**Bottom line: none of the 14 findings map to a currently-failing terrifi
test.** Each one is either (a) already handled by terrifi and proven by a
passing acceptance test, (b) not applicable because terrifi doesn't model the
field, or (c) dormant because terrifi hasn't ported the affected resource
yet. So there are no `t.Skip("F-NNN…")` markers to add — adding skips to green
tests would be wrong. This file is the catalog the improvement-plan §8 L08
calls for; revisit it when a dormant resource lands.

Legend: **handled** = terrifi does the right thing, covered by a passing test ·
**n/a** = terrifi doesn't expose the affected field · **dormant** = resource
not ported; re-evaluate when it lands.

| Finding | Upstream subject | terrifi status | Notes |
|---|---|---|---|
| F-001 | `unifi_network` doesn't expose `purpose` | **handled (partial)** | terrifi_network exposes `purpose` = `corporate` \| `vlan-only`. `purpose = "guest"` is **not** supported — dormant gap, recorded in `examples/complete/stubs.tf`. Covered by `TestAccNetwork_*` (6/6). |
| F-002 | `subnet` required even for vlan-only | **handled** | terrifi_network applies a vlan-only network with no subnet. Covered by `TestAccNetwork_vlanOnly`. |
| F-003 | `data.ap_group` requires `name` | dormant | No terrifi AP-group data source. Re-evaluate if one is added. |
| F-004 | `data.client_qos_rate` requires `name` | dormant | No terrifi client-QoS-rate resource/data source. |
| F-005 | default AP group is `"All APs"` not `"Default"` | dormant | Same as F-003 — no AP-group lookup in terrifi. |
| F-006 | `network.domain_name` round-trips empty | **handled** | `unifi.Network` exposes `DomainName *string` (E02); empty-string responses decode as null in `apiToModel` to match the schema's optional shape. Covered by the `domain_name round-trips` unit tests and `TestAccNetwork_dhcpBootDomainMdns`. |
| F-007 | `unifi_firewall_rule` bool `null` vs `false` | dormant | terrifi has no `firewall_rule` (it uses zone-based firewall). The analogous bool-omitempty concern in `firewall_policy` is already handled (see `firewall_policy_api.go`). |
| F-008 | `port_forward.wan.interface` rejects LAN IDs | dormant | No terrifi port-forward resource. |
| F-009 | `traffic_route` fails on zero-device controller | dormant | No terrifi traffic-route resource. |
| F-010 | `port_profile` round-trips 3 fields wrong (2 parts) | dormant | No terrifi port-profile resource. |
| F-011 | `unifi_wlan` create → `InvalidPayload` from `schedule_with_duration: null` | **handled** | terrifi sends `schedule_with_duration` as `[]`, never `null` (`normalizeWLAN` in `wlan_api.go`). Covered by `TestAccWLAN_*` (23/23 on UOS 5.1.15). |
| F-012 | `dynamic_dns` → `UnsupportedDynamicDns` | dormant | No terrifi dynamic-DNS resource. |
| F-013 | `firewall_rule` WAN_IN index 20000+ rejected with devices | dormant | No terrifi `firewall_rule`. |
| F-014 | `traffic_route` POST 400 (post-F-009) | dormant | No terrifi traffic-route resource. |
| F-015 | `network.mdns_enabled` is coerced to `true` on write | **handled** | UOS 5.1.15 silently stores `mdns_enabled: true` regardless of what we POST/PUT (confirmed via direct curl: create with `false` → GET reads `true`; PUT with `false` → GET still reads `true`). terrifi's `multicast_dns` schema default is `true` to match, and `ModifyPlan` forces `false` only on vlan-only networks where the controller does not emit the field. There is no way to disable mdns per-network from this provider; users who need it off must use the controller UI / site-level settings. Covered by `TestAccNetwork_dhcpBootDomainMdns`. |
| F-016 | DNS A record name collision blocks every write to a `/rest/user` reservation that carries that hostname as `local_dns_record` | **n/a (controller behavior; user-side fix required)** | The controller's reservation-writer validates that the reservation's `local_dns_record` value is not already owned by an explicit DNS A record. If a row exists at Settings → Policies → DNS records with the same hostname, every `POST`/`PUT` on `/rest/user/{id}` for that MAC fails with the opaque `api.err.Invalid` — independent of which `_id`, what payload, or what other fields change. Surfaced 2026-06-02 against UDM-Pro Network 10.x while bulk-importing an existing site's reservations: 22 of 23 applied cleanly and one — the only one whose hostname also had a hand-created DNS A record — refused every write. Symptom is easy to misread as a corrupt `_id` or a payload problem, because the error is identical regardless of what you change. Resolution: delete the colliding DNS A row from Settings → Policies → DNS records via the UI, then the API accepts the reservation write. Provider-side defense would have to inspect unrelated DNS-record data before each `/rest/user` write — not worth the complexity; a docstring on `terrifi_client_device.local_dns_record` pointing at this finding is sufficient. |
| F-017 | Clearing an optional string on `networkconf` is accepted for `domain_name` only; `ip_subnet` / `dhcpd_boot_server` / `dhcpd_boot_filename` reject `""` | **n/a (controller behavior; provider cannot fix)** | Probed field-by-field on UOS 5.1.15 (Network app 10.4.57) with full-object `PUT`s to `/rest/networkconf/{id}`, one field emptied per request, re-reading after each: `domain_name: ""` → **200, field cleared** (reads back `""`). `dhcpd_boot_server: ""` → **400 `api.err.BootServerInvalid`**. `dhcpd_boot_filename: ""` → **400 `api.err.InvalidPayload`**. `ip_subnet: ""` → **400 `api.err.InvalidPayload`**. Also tried the realistic "user removes the PXE block" path — `dhcpd_boot_enabled:false` plus both boot fields emptied in a single PUT — which is **also 400 `api.err.InvalidPayload`**, so disabling the feature first does not unlock clearing its fields. Consequence for the provider: there is no single merge-semantics change that makes "remove the attribute from config" clear these on the controller, because for three of the four the controller refuses the only wire representation available. `domain_name` is genuinely clearable; the rest are set-or-change-only and should be documented as such (the `Optional + Computed` shape used for `client_device.note`). Relevant to the "optional string clearing" report upstream. |

## Separate from the F-findings: the zone-based-firewall simulation gap

terrifi's `firewall_zone` / `firewall_policy` / `firewall_policy_order` tests
are gated `requireHardware(t)` because the controller only seeds the default
zones (incl. the `Hotspot` parent every user zone needs) once a real gateway
is adopted and Connected — neither docker nor UOS simulation does this. That
is **not** one of F-001 … F-014; it is documented in
`testing/README.md` → "firewall zones require real adopted hardware" and in
`examples/complete/firewall-zbf.tf`.

## When a dormant resource lands

When terrifi gains one of the unported resources (port profile, port forward,
static/traffic route, RADIUS profile/user, dynamic DNS, AP-group data
source), re-read the matching finding above and the full write-up in the
investigation doc, add the resource's acceptance tests, and — only if a test
genuinely fails for the documented reason — add a
`t.Skip("F-NNN: <one-line>")` until the provider catches up. Update this
catalog's status column at the same time.
