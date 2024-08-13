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
  depends_on = [zerotier_member.bobs_car]
  network_id = zerotier_network.bobs_garage.id
}

output "member_name" {
  value = one(data.zerotier_members.bobs_garage.members)["name"]
}

output "member_description" {
  value = one(data.zerotier_members.bobs_garage.members)["description"]
}
