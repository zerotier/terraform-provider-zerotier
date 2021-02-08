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

resource "docker_container" "uno" {
  name  = "zerotier_uno"
  image = docker_image.zerotier.latest
}

resource "docker_container" "dos" {
  name  = "zerotier_dos"
  image = docker_image.zerotier.latest
}
