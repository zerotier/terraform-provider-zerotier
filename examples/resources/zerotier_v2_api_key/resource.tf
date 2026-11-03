resource "zerotier_v2_api_key" "ci" {
  service_account_id = zerotier_v2_service_account.ci.id
  description        = "GitHub Actions"
  expires            = "2027-06-30T00:00:00Z"
}

# The key is only returned when it is created, and is stored in state.
output "ci_api_key" {
  value     = zerotier_v2_api_key.ci.key
  sensitive = true
}
