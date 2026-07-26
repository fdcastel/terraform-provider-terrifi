---
page_title: "terrifi_clients Data Source - Terrifi"
subcategory: ""
description: |-
  Lists every client the UniFi controller knows about on a site.
---

# terrifi_clients (Data Source)

Returns the list of clients the UniFi controller knows about on a site — every record at
`/api/s/{site}/rest/user`, which is the union of currently-connected stations and
configured/remembered clients.

Two things this is good for: a read-only liveness check on the controller, and a source for
`for_each` over clients that already exist without declaring each one.

~> This is the *known-client* table, not the live-station table. A record here does not mean the
client is connected right now — use `last_seen` for that. Conversely, clients that have connected
but were never configured still appear.

## Example Usage

### List every client

```terraform
data "terrifi_clients" "all" {}

output "client_count" {
  value = length(data.terrifi_clients.all.clients)
}
```

### Read a specific site

```terraform
data "terrifi_clients" "branch" {
  site = "branch-office"
}
```

### Find the wired clients that hold a DHCP reservation

```terraform
data "terrifi_clients" "all" {}

locals {
  reserved = [
    for c in data.terrifi_clients.all.clients : c
    if c.is_wired && c.fixed_ip != null
  ]
}

output "reservations" {
  value = { for c in local.reserved : c.mac => c.fixed_ip }
}
```

### Use as a liveness assertion

```terraform
data "terrifi_clients" "all" {}

check "controller_reachable" {
  assert {
    condition     = length(data.terrifi_clients.all.clients) > 0
    error_message = "Controller returned no clients — the API is up but the site looks empty."
  }
}
```

## Schema

### Optional

- `site` (String) — The site to read clients from. Defaults to the provider site.

### Read-Only

- `clients` (List of Object) — Every client record on the site. Attributes below.

### Nested Schema for `clients`

- `id` (String) — The controller-assigned ID of the client record.
- `mac` (String) — The MAC address of the client.
- `name` (String) — The user-assigned alias for the client (the `name` field on
  `terrifi_client_device`). Often null for clients that have never had an alias set.
- `hostname` (String) — The hostname the client reported via DHCP option 12, if any.
- `ip` (String) — The IP the client most recently held. May be null if the controller has never
  observed the client connecting.
- `fixed_ip` (String) — The fixed (DHCP-reserved) IP assigned to the client, if any.
- `network_id` (String) — The configured network ID for the client (set when `fixed_ip` is in use).
- `network_name` (String) — The name of the network the client was last seen on.
- `is_wired` (Boolean) — Whether the client was last seen on a wired connection.
- `is_guest` (Boolean) — Whether the client is on a guest network.
- `oui` (String) — The OUI (vendor) of the client's MAC address.
- `blocked` (Boolean) — Whether the client is blocked from network access.
- `note` (String) — Free-text note attached to the client (the `note` field on
  `terrifi_client_device`).
- `local_dns_record` (String) — The local DNS record hostname for the client, if one is configured.
- `fixed_ap_mac` (String) — The MAC of the access point the client is locked to, if any.
- `first_seen` (Number) — Unix timestamp when the controller first saw the client.
- `last_seen` (Number) — Unix timestamp when the controller last saw the client.

## Where the values come from

A few attributes are not stored under the name you might expect, which matters when comparing
against the raw API or the UI:

| Attribute | Controller field | Note |
|---|---|---|
| `ip` | `last_ip` | The last observed address, not a current lease. Null for clients never seen connecting. |
| `network_name` | `last_connection_network_name` | The network the client was last seen on. |
| `network_id` | `network_id`, falling back to `last_connection_network_id` | Reservations created through the UI carry no explicit `network_id` — the controller resolves the DHCP scope from the address — so the fallback keeps this consistent with `network_name` instead of reporting a populated name next to a null ID. |

Optional strings decode as **null** rather than `""` when absent, so `c.name != null` distinguishes
"no alias set" from "alias set to empty". Guard on null before string operations in a `for`
expression.
