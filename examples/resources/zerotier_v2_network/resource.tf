resource "zerotier_v2_network" "lab" {
  network_group_id = zerotier_v2_network_group.lab.id
  name             = "lab"

  v4_subnet           = "10.99.0.0/24"
  v4_assignment_pools = [{ start = "10.99.0.10", end = "10.99.0.250" }]
  routes              = [{ target = "192.168.50.0/24", via = "10.99.0.1" }]

  dns_domain  = "lab.example.com"
  dns_servers = ["10.99.0.53"]

  mtu             = 2800
  multicast_limit = 32
}
