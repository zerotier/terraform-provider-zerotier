# output "alice" {
#   value = zerotier_identity.alice
# }

# output "bob" {
#   value = zerotier_identity.bob
# }

output "networks" {
  value = {
    for n in zerotier_network.this:
    n.name => n.id
  }
}
