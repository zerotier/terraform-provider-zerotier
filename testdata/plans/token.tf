provider "zerotier" {}

resource "zerotier_token" "terraform-test-hello-world" {
  name = "hello-world"
}

resource "zerotier_token" "terraform-test-random-string" {}
