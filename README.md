######

https://learn.hashicorp.com/collections/terraform/providers
https://my.zerotier.com/help/api

## tests

- make test
  - `FORCE_TESTS=1`: do not use test cache
  - `QUIET_TESTS=1`: do not show test log (just results)
- `ZEROTIER_CONTROLLER_TOKEN`
  - set in env or write to `test-token.txt` at the root.
    - env is preferred but the token from file is just propagated to env and gitignored. No different, just easier to use.
