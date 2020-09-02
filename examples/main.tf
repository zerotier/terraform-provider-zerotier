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
# Identity
#

# resource "zerotier_identity" "alice" {}
# resource "zerotier_identity" "bob" {}

#
# Networks
#

resource "zerotier_network" "this" {
  for_each = var.networks

  name = each.key
  assignment_pool {
    cidr = each.value.ipv4_cidr

  }
  route {
    target = each.value.ipv4_cidr
  }
}
