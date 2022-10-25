# Terraform Provider ZeroTier

ZeroTier is a smart Ethernet switch for planet Earth.

It's a distributed network hypervisor built atop a cryptographically
secure global peer to peer network. It provides advanced network
virtualization and management capabilities on par with an enterprise
SDN switch, but across both local and wide area networks, connecting
almost any kind of app or device.

It does not distinguish between on-premise, cloud, desktop, or mobile
devices. You can even ZeroTier-enable individual programs with the SDK
socket interface.

This Terraform provider allows you to manipulate objects in the [ZeroTier Central API](https://my.zerotier.com/help/api):

# Networks

ZeroTier Networks can be thought of as encrypted virtual Ethernet
switches.

```hcl
resource "zerotier_network" "occams_router" {
  name        = "occams_router"
  description = "The prefix with largest number of bits is usually correct"
  assignment_pool {
    start = "10.1.0.1/24"
    end   = "10.1.0.254/24"
  }
  route {
    target = "10.1.0.0/24"
  }
  flow_rules = "accept;"
}
```

ZeroTier networks can also push routes to network members, if they
want to receive them.

```hcl
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
    via = "10.2.0.1"
  }
  flow_rules = "accept;"
}
```

ZeroTier networks have a robust ```flow_rules``` language, allowing you do
to things like dropping select traffic, or even as advanced as
Ethernet tapping. Please refer to the [ZeroTier Reference Manual](https://www.zerotier.com/manual/) for
details.


```hcl
resource "zerotier_network" "silence_of_the_lan" {
  name        = "silence_of_the_lan"
  description = "It puts the bits in the bucket. It does this whenever it is told."
  assignment_pool {
    start = "10.3.0.1/24"
    end   = "10.3.0.254/24"
  }
  route {
    target = "10.3.0.0/24"
  }
  flow_rules = "drop;"
}
```

Pre 1.0.0 versions of this provider allowed you to specify an IPv4 CIDR
instead of a start/end range. This was removed to reflect the reality
of the current Central API. If you would like to use a CIDR, please
see the [ZeroTier Network Terraform Module](https://registry.terraform.io/modules/zerotier/network/zerotier/latest)
on the Registry, or [Github](https://github.com/zerotier/terraform-zerotier-network)

# Members

Members are associations between Nodes and Networks. These are created
when a node is authorized in the WebUI.


```hcl
resource "zerotier_member" "alice" {
  name                    = "alice"
  member_id               = "ABCDEF1234"
  network_id              = zerotier_network.occams_router.id
  description             = "Curiouser and curiouser"
  hidden                  = true
  allow_ethernet_bridging = true
  no_auto_assign_ips      = true
  ip_assignments          = ["10.1.0.42"]
}
```


```hcl
resource "zerotier_member" "bob" {
  name                    = "in Bob we trust"
  member_id               = "1234ABCDEF"
  network_id              = zerotier_network.schrödingers_nat.id
}
```

# Identities

The ```zerotier_identity``` resource is the odd-ball of the bunch. You
cannot create an Identity in the API. A ZeroTier identity is the
cryptographic identity of a ZeroTier node. It is more akin to a
[Terraform TLS Private Key](https://registry.terraform.io/providers/hashicorp/tls/latest/docs/resources/private_key).

In "normal" ZeroTier usage, the ZeroTier Identity is created by the ZeroTier
client on first launch. When a client tries to join a ZeroTier
Network, the public half shows up in the WebUI, waiting for
Authorization from an administrator. (A "member" association in the
API).

Without a third party to certify the validity of the identity
(Certificate Authority model), the node's identity needs to be verified
out-of-band of ZeroTier. This usually means it is Trusted on First
Use. This is the same pattern as SSH keys.

With dymamic and ephemeral infrastructure, we have the usual
chicken-and-egg problem. We cannot associate a member with a network
until we know the identity. 

The ```zerotier_identity``` resource lets us pre-generate an identity
for use with a ```zerotier_member``` resource, but a freshly provisioned
instance or container will not know the secret.

Therefore, the secret part of the identity will need to somehow be installed on the node by one of:

- Injection via userdata / environment
- Pre-baking of the secret into the booted instance or container
- Mounting of the secret as a volume

In any event, usage of the ```zerotier_identity``` resource means the
secret will be stored in the Terraform State, creating a potential
security risk, and should be documented as such.


```hcl
resource "zerotier_identity" "alice" {}
resource "zerotier_identity" "bob" {}
```

# Tokens

The ```zerotier_token``` resource will return a token from the Central
API. You can use this to inject a token into containers or instance
cloud-init scripts for things that also need to talk to the Central
API.

```hcl
zerotier_token "foo" {
  name = "foo"
}

zerotier_token "bar" {
  name = "bar"
}
```

# Putting it all together.

This example connects two docker containers with zerotier. You can then `docker exec` into one and ping the other over the ZeroTier network.

```hcl
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

provider "zerotier" {}

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
ZEROTIER_CENTRAL_TOKEN=<apitoken> terraform apply -auto-approve
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
ZEROTIER_CENTRAL_TOKEN=<apitoken> terraform destroy -auto-approve
```

And everything should be cleaned up! Note that you can do more with the allocation system to better scope your IP addresses; this is just a way to show off more that you can do.

# Development

Included here is a description of the Make tasks and environment variables you need to run the tests and perform builds.

- `make checks`: Everything you need to do to get through CI
- `make test`
  - `FORCE_TESTS=1`: do not use test cache
  - `QUIET_TESTS=1`: do not show test log (just results)
  - `TEST=<pattern>`: run a specific test or tests that match the pattern
- `ZEROTIER_CENTRAL_TOKEN`
  - set in env or write to `test-token.txt` at the root.
    - env is preferred but the token from file is just propagated to env and gitignored. No different, just easier to use.

## Cleanup commands you may find useful
Sometimes tests fail and resources get left behind

- `docker ps -a -f name=zerotier --format {{.ID}} | xargs docker rm -f` removed orphaned containers with "zerotier" in the name.
- `docker network prune` removed orphaned networks 
- `rm /tmp/tftest` if problems downloading kreuzwerker/docker.
- Delete the "hello-word" auth token from your central account.
- Don't use your main central account for tests. Use an alternate. Delete it and recreate when too many networks to clean up.

# License

- BSD 3-Clause

