provider "zerotier" {}

resource "zerotier_network" "bobs_garage" {
  name        = "bobs_garage"
  description = "so say we bob"
}

resource "zerotier_identity" "bob" {}

resource "zerotier_member" "bobs_car" {
  network_id  = zerotier_network.bobs_garage.id
  member_id   = zerotier_identity.bob.id
  name        = "bobs_car"
  description = "bobs shiny car"
  authorized  = true
}

data "zerotier_members" "bobs_garage" {
  network_id = zerotier_network.bobs_garage.id
}