resource "zerotier_member" "alice" {
  name                    = "alice"
  member_id               = zerotier_identity.alice.id
  network_id              = zerotier_network.alicenet.id
  description             = "Hello, world"
  hidden                  = true
  allow_ethernet_bridging = true
  no_auto_assign_ips      = true
  ip_assignments          = ["10.0.0.1"]
}
