# terraform-provider-zerotier CHANGELOG

## v1.0.1
- Updating deps

## v1.0.0

Initial Stable Release!
This project will follow semantic versioning as described at https://semver.org/

- Changes from v0.1.62...
- CIDR has been removed in favor of start/end ranges, as that is what
  the API actually uses.
- If you would like to describe subnets with CIDR, please see the
  module at https://registry.terraform.io/modules/zerotier/network/zerotier/latest
- Removed MTU, as you cannot actually set it.
- Both assign_ipv4 and assign_ipv6 have been changed to sets instead
  of maps, This allows them to be presented as blocks (useful for
  dynamic configurations).  
- Adding zerotier_token resource