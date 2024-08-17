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
  private  = false
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
  private     = false
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

resource "zerotier_network" "dns_settings" {
  name = "dns_settings"
  dns {
    domain = "leisure.town"
    servers = [
      "1.2.3.4",
      "5.6.7.8"
    ]
  }
}

# resource "zerotier_network" "sso_config" {
#   name = "sso_config"
#   sso_config {
#     allow_list             = ["hi.com", "bye.com"]
#     authorization_endpoint = "https://computers.biz"
#     client_id              = "H4H4H4H0H0H0H0H3H3H3"
#     issuer                 = "https://computers.biz"
#     mode                   = "default"
#     enabled                = false
#   }
# }

resource "zerotier_identity" "bob" {}

resource "zerotier_member" "bob" {
  name       = "bob"
  member_id  = zerotier_identity.bob.id
  network_id = zerotier_network.bobs_garage.id
}
