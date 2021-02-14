provider "zerotier" {}

resource "zerotier_network" "bobs_garage" {
  name        = "bobs_garage"
  description = "so say we bob"
}

data "zerotier_network" "bob" {
  id = zerotier_network.bobs_garage.id
}
