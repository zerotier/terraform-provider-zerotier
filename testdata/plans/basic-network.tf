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
  #  assignment_pool { cidr = each.value.ipv4_cidr }
  #  route { target = each.value.ipv4_cidr }
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

resource "zerotier_network" "bobs_garage" {
  name        = "bobs_garage"
  description = "so say we bob"
  //  rules_source = "accept;"
  //  ip_assignment_pools {
  //      ip_range_start = "192.168.1.1"
  //      ip_range_end = "192.168.1.254"
  //  }
  //  routes { target = "192.168.1.0/24" }
}

resource "zerotier_identity" "bob" {}

resource "zerotier_member" "bob" {
  name       = "bob"
  node_id    = zerotier_identity.bob.id
  network_id = zerotier_network.bobs_garage.id
}

#resource "zerotier_member" "sean" {
#  name           = "sean"
#  node_id        = "eff05def90"
#  network_id     = zerotier_network.bob.id
#  ip_assignments = ["192.168.1.42", "192.168.1.69"]
#}
