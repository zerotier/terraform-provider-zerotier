terraform {
  required_providers {
    zerotier = {
      source  = "zerotier/zerotier"
      version = "0.2.0"
    }
  }
}

provider "zerotier" {}

#
# Occam's Router
#

resource "zerotier_network" "occams_router" {
  name        = "occams_router"
  description = "The route prefix with largest number of bits is usually correct"
  assignment_pool {
    start = "10.1.0.1/24"
    end   = "10.1.0.254/24"
  }
  route {
    target = "10.1.0.0/24"
  }
  flow_rules = "accept;"
}

#
# Schrödinger's Nat
#

resource "zerotier_network" "schrödingers_nat" {
  name        = "schrödingers_nat"
  description = "A packet's destination is simultaneiously Alice and Bob until observed by a NAT table."
  assignment_pool {
    start = "10.2.0.1/24"
    end   = "10.2.0.254/24"
  }
  route {
    target = "10.2.0.0/24"
  }
  route {
    target = "0.0.0.0/0"
    via    = "10.2.0.1"
  }
  flow_rules = "accept;"
}

#
# Silence of the Lan
#

resource "zerotier_network" "silence_of_the_lan" {
  name        = "silence_of_the_lan"
  description = "It puts the packet in the bit bucket. It does this whenever it is told."
  assignment_pool {
    start = "10.3.0.1/24"
    end   = "10.3.0.254/24"
  }
  route {
    target = "10.3.0.0/24"
  }
  flow_rules = "drop;"
}
