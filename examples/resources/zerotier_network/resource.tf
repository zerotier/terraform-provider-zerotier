resource "zerotier_network" "my_network" {
  name        = "<required>"
  description = "Managed by Terraform"

  assign_ipv4 = {
    zerotier = true
  }

  assign_ipv6 = {
    zerotier = true
    sixplane = false
    rfc4193  = false
  }

  enable_broadcast = true
  private          = false
  flow_rules       = "accept;"
}
