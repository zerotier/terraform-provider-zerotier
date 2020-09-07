terraform {
  required_providers {
    zerotier = {
      versions = ["0.2"]
      source = "zerotier.com/dev/zerotier"
    }
  }
}

#
# Provider
#

provider "zerotier" {}

#
# vars
#

variable "networks" {
  type = map
  default  = {
    occams_router = {
      ipv4_cidr = "10.0.1.0/24"
      ipv6_cidr = "fc19:c8b9:01/80"
    }
    schrödingers_nat = {
      ipv4_cidr = "10.0.2.0/24"
      ipv6_cidr = "fc19:c8b9:02/80"
    }
    silence_of_the_lan  = {
      ipv4_cidr = "10.0.3.0/24"
      ipv6_cidr = "fc19:c8b9:03/80"
    }
  }
}

#
# Alice
#

resource "zerotier_network" "alice" {
  for_each = var.networks
  name = each.key
  assignment_pool { cidr = each.value.ipv4_cidr }
  route { target = each.value.ipv4_cidr }
}

resource "zerotier_identity" "alice" {}

resource "zerotier_member" "alice" {
  for_each   = zerotier_network.alice
  name       = "${each.key}-alice"
  node_id    = zerotier_identity.alice.id
  network_id = each.value.id
}

#
# Bob
#

resource "zerotier_network" "bob" {
  name = "bobs_garage"
  assignment_pool { cidr = "192.168.1.0/24" }
  route { target = "192.168.1.0/24" }
}

resource "zerotier_identity" "bob" {}

resource "zerotier_member" "bob" {
  name       = "bob"
  node_id    = zerotier_identity.bob.id
  network_id = zerotier_network.bob.id
}
