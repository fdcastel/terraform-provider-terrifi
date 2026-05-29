# examples/complete — coverage stubs.
#
# The EXAMPLE_NETWORK reference (tmp/EXAMPLE_NETWORK.md) describes resources
# terrifi does not implement yet. They are recorded here as commented blocks
# so the gap is visible alongside the working config and reviewers can see
# exactly what the reference network needs next. Each block names the upstream
# UniFi concept and the reference section it comes from. When terrifi grows a
# resource, lift its block out of here into main.tf (or firewall-zbf.tf).
#
# None of these are valid HCL yet — they are intentionally commented out.

# --- WAN (dual ISP) -------------------------------------------------------
# Reference §WAN. No terrifi_wan resource. The reference's tfacc-wan-isp1
# (primary) / tfacc-wan-isp2 (failover) cannot be expressed today.
#
# resource "terrifi_wan" "isp1" { ... }   # primary, DHCP dual-stack
# resource "terrifi_wan" "isp2" { ... }   # failover

# --- purpose=guest network ------------------------------------------------
# Reference §Networks. terrifi_network.purpose only accepts corporate|vlan-only.
# tfacc-net-guest is modeled as corporate in main.tf; the controller's native
# guest-policy branch (captive portal, voucher hooks) is unmanaged.
#
# resource "terrifi_network" "guest_native" { purpose = "guest" ... }

# --- Port profiles --------------------------------------------------------
# Reference §Port profiles (6 profiles). No terrifi_port_profile resource.
#
# resource "terrifi_port_profile" "trunk_all"      { ... }
# resource "terrifi_port_profile" "ap_uplink"      { ... }
# resource "terrifi_port_profile" "access_trusted" { ... }
# resource "terrifi_port_profile" "access_iot"     { ... }
# resource "terrifi_port_profile" "access_guest"   { ... }
# resource "terrifi_port_profile" "dot1x"          { ... }   # 802.1X / RADIUS

# --- Port forwards / DNAT capture -----------------------------------------
# Reference §Firewall (DNAT capture) + §Inbound. No terrifi_port_forward
# resource. Blocks the 4 tfacc-pf-dnat-* DNS/NTP captures and tfacc-pf-https.
# Note the reference also flags wan.interface="lan" as a likely provider gap.
#
# resource "terrifi_port_forward" "dnat_dns_iot"  { ... }
# resource "terrifi_port_forward" "https"         { ... }

# --- Static / traffic routes ----------------------------------------------
# Reference §Routing. No terrifi_static_route / terrifi_traffic_route.
#
# resource "terrifi_static_route"  "test_net"          { ... }
# resource "terrifi_traffic_route" "trusted_via_wan1"  { ... }

# --- RADIUS ---------------------------------------------------------------
# Reference §RADIUS. No terrifi_radius_profile / terrifi_radius_user. The
# 802.1X port profile above would bind to this.
#
# resource "terrifi_radius_profile" "radius"      { ... }
# resource "terrifi_radius_user"    "radius_user" { ... }

# --- Dynamic DNS ----------------------------------------------------------
# Reference §Dynamic DNS. No terrifi_dynamic_dns resource.
#
# resource "terrifi_dynamic_dns" "cloudflare" { ... }
