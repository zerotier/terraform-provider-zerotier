provider "zerotier" {}

variable "networks" {
  type = map(any)
  default = {
    occams_router = {
      ipv4_cidr = "10.0.1.0/24"
      ipv6_cidr = "fc19:c8b9:01/80"
    }
    schrödingers_nat = {
      ipv4_cidr = "10.0.2.0/24"
      ipv6_cidr = "fc19:c8b9:02/80"
    }
    silence_of_the_lan = {
      ipv4_cidr = "10.0.3.0/24"
      ipv6_cidr = "fc19:c8b9:03/80"
    }
  }
}

resource "zerotier_network" "alice" {
  for_each = var.networks
  name     = each.key
}

resource "zerotier_identity" "alice" {}

resource "zerotier_member" "alice" {
  for_each   = zerotier_network.alice
  name       = "${each.key}-alice"
  member_id  = zerotier_identity.alice.id
  network_id = each.value.id
}


resource "zerotier_network" "bobs_garage" {
  name        = "bobs_garage"
  description = "so say we bob"
}

resource "zerotier_network" "assign_off" {
  name = "assign_off"
  assign_ipv4 {
    zerotier = false
  }

  assign_ipv6 {
    zerotier = false
    sixplane = true
    rfc4193  = true
  }
}

resource "zerotier_network" "description" {
  name        = "description"
  description = "My description is changed!"
}

resource "zerotier_network" "no_broadcast" {
  name             = "no_broadcast"
  enable_broadcast = false
}

resource "zerotier_network" "mtu" {
  name = "mtu"
  mtu  = 1500
}

resource "zerotier_network" "multicast_limit" {
  name            = "multicast_limit"
  multicast_limit = 50
}

resource "zerotier_network" "private" {
  name    = "private"
  private = true
}

resource "zerotier_network" "flow_rules" {
  name       = "flow_rules"
  flow_rules = "drop;"
}

resource "zerotier_identity" "bob" {}

resource "zerotier_member" "bob" {
  name       = "bob"
  member_id  = zerotier_identity.bob.id
  network_id = zerotier_network.bobs_garage.id
}
