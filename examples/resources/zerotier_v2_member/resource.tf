resource "zerotier_identity" "server" {}

resource "zerotier_v2_member" "server" {
  network_id = zerotier_v2_network.lab.id
  device_id  = zerotier_identity.server.id
  name       = "server"
  status     = "authorized"
}
