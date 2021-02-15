# Terraform Provider for the ZeroTier Central API

Connect team members from anywhere in the world on any device. ZeroTier creates secure networks between on-premise, cloud, desktop, and mobile devices.

This terraform provider allows you to stitch the following objects from the [ZeroTier Central API](https://my.zerotier.com/help/api):

- Networks
- Members
- Identities

# Example

This example connects two docker containers with zerotier. You can then `docker exec` into one and ping the other over the ZeroTier network.

```terraform
terraform {
  required_providers {
    docker = {
      source = "kreuzwerker/docker"
    }
    # FIXME update for final zerotier source location
  }
}

provider "docker" {}

resource "docker_network" "private_network" {
  name = "my_network"
}

resource "docker_image" "zerotier" {
  name         = "erikh/zerotier"
  keep_locally = true
}

resource "docker_container" "alice" {
  name    = "zerotier_alice"
  image   = docker_image.zerotier.latest
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
  image   = docker_image.zerotier.latest
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
```

From here, we can:

```shell
terraform init
ZEROTIER_CONTROLLER_TOKEN=<apitoken> terraform apply -auto-approve
```

And things should "just work". Once everything has applied, let's try a ping between the two container images:

```
[520] erikh@zerotier terraform/docker-test% docker exec -it zerotier_alice zerotier-cli listnetworks
200 listnetworks <nwid> <name> <mac> <status> <type> <dev> <ZT assigned ips>
200 listnetworks 8056c2e21c1930be docker be:ec:37:36:f9:7a OK PUBLIC ztmjfnxuvc 10.0.1.93/24

# Ok. Alice is 10.0.1.93.

[522] erikh@zerotier terraform/docker-test% docker exec -it zerotier_bob zerotier-cli listnetworks
200 listnetworks <nwid> <name> <mac> <status> <type> <dev> <ZT assigned ips>
200 listnetworks 8056c2e21c1930be docker be:fd:18:cb:02:55 OK PUBLIC ztmjfnxuvc 10.0.1.247/24

# Ok. Bob is 10.0.1.247. Let's ping from alice to bob over zerotier:

[523] erikh@zerotier terraform/docker-test% docker exec -it zerotier_alice ping 10.0.1.247
PING 10.0.1.247 (10.0.1.247) 56(84) bytes of data.
64 bytes from 10.0.1.247: icmp_seq=1 ttl=64 time=205 ms
64 bytes from 10.0.1.247: icmp_seq=2 ttl=64 time=0.288 ms
64 bytes from 10.0.1.247: icmp_seq=3 ttl=64 time=0.302 ms
64 bytes from 10.0.1.247: icmp_seq=4 ttl=64 time=0.273 ms
64 bytes from 10.0.1.247: icmp_seq=5 ttl=64 time=0.332 ms
```

Once we're done:

```
ZEROTIER_CONTROLLER_TOKEN=<apitoken> terraform destroy -auto-approve
```

And everything should be cleaned up! Note that you can do more with the allocation system to better scope your IP addresses; this is just a way to show off more that you can do.

# Development

Included here is a description of the Make tasks and environment variables you need to run the tests and perform builds.

- `make checks`: Everything you need to do to get through CI
- `make test`
  - `FORCE_TESTS=1`: do not use test cache
  - `QUIET_TESTS=1`: do not show test log (just results)
  - `TEST=<pattern>`: run a specific test or tests that match the pattern
- `ZEROTIER_CONTROLLER_TOKEN`
  - set in env or write to `test-token.txt` at the root.
    - env is preferred but the token from file is just propagated to env and gitignored. No different, just easier to use.

# License

- BSD 3-Clause
