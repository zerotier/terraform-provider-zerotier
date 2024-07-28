terraform {
  required_providers {
    docker = {
      source = "kreuzwerker/docker"
    }
  }
}

provider "docker" {}

resource "docker_network" "private_network" {
  name = "my_network"
}

resource "docker_image" "zerotier" {
  name         = "zerotier/terraform-test"
  keep_locally = true
}

resource "docker_container" "alice" {
  name    = "zerotier_alice"
  image   = docker_image.zerotier.image_id
  command = [zerotier_network.docker_network.id]

  devices { host_path = "/dev/net/tun" }

  capabilities {
    add = ["CAP_NET_ADMIN", "CAP_SYS_ADMIN"]
  }

  upload {
    file    = "/var/lib/zerotier-one/identity.public"
    content = zerotier_identity.alice.public_key
  }
  upload {
    file    = "/var/lib/zerotier-one/identity.secret"
    content = zerotier_identity.alice.private_key
  }
}

resource "docker_container" "bob" {
  name    = "zerotier_bob"
  image   = docker_image.zerotier.image_id
  command = [zerotier_network.docker_network.id]

  capabilities {
    add = ["CAP_NET_ADMIN", "CAP_SYS_ADMIN"]
  }

  devices { host_path = "/dev/net/tun" }

  upload {
    file    = "/var/lib/zerotier-one/identity.public"
    content = zerotier_identity.bob.public_key
  }
  upload {
    file    = "/var/lib/zerotier-one/identity.secret"
    content = zerotier_identity.bob.private_key
  }
}

#
# Provider
#

provider "zerotier" {}

#
# Alice
#

variable "ipv4_cidr" {
  default = "10.0.1.0/24"
}

resource "zerotier_network" "docker_network" {
  name = "docker"

  assignment_pool {
    start = "10.0.1.2"
    end   = "10.0.1.253"
  }

  route {
    target = var.ipv4_cidr
    via    = "10.0.0.1"
  }
}

resource "zerotier_identity" "alice" {}

resource "zerotier_member" "alice" {
  name       = "docker-alice"
  member_id  = zerotier_identity.alice.id
  network_id = zerotier_network.docker_network.id
}

resource "zerotier_identity" "bob" {}

resource "zerotier_member" "bob" {
  name       = "docker-bob"
  member_id  = zerotier_identity.bob.id
  network_id = zerotier_network.docker_network.id
}
