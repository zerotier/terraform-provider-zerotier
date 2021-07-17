provider "zerotier" {}

resource "zerotier_network" "bobs_garage" {
  name        = "bobs_garage"
  description = "so say we bob"
}

data "zerotier_network" "bob" {
  id = zerotier_network.bobs_garage.id
}

resource "zerotier_network" "bob2" {
  name        = zerotier_network.bobs_garage.name
  description = zerotier_network.bobs_garage.description
}
