# terraform-provider-zerotier CHANGELOG

## v1.5.1
- Fixing up tests

## v1.5.0
- Network members datasource

## v1.4.2
- Signing with correct GPG key

## v1.4.1
- fixing 6plane / rfc4193 address flipping #38
- Update terraform-plugin-docs + generated files + dependencies #39

## v1.4.0
- Add rfc4193 and sixplane to member.go
- Return human-readable error on malformed member id #35
- Thanks Andrew Sichevoi!

## v1.3.1
- Fix for managing DNS setting (#23)
- Fix for avoid unnecessary diffs of members (#33)
- Documentation fixes (#31) (#26)

## v1.3.0
- Allow importing zerotier_member resources

## v1.2.0
- Support importing network resources

## v1.1.0
- Support tags in member definition
- Support zerotier_central_url
- Fix version tagging in user_agent

## v1.0.2
- BUGFIX: Networks are now private by default

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
