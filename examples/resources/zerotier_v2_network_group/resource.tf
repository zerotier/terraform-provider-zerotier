resource "zerotier_v2_network_group" "lab" {
  org_id      = "<your org id>"
  name        = "lab"
  description = "Managed by Terraform"
}
