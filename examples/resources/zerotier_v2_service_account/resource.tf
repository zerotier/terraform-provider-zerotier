resource "zerotier_v2_service_account" "ci" {
  org_id      = "<your org id>"
  slug        = "ci-bot"
  name        = "CI bot"
  description = "Used by the deploy pipeline"
}
