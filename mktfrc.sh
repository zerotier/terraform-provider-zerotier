#!/bin/sh

cat >test.tfrc <<EOF
provider_installation {
  filesystem_mirror {
    path = "${PWD}/.tfdata"
  }

  direct {}
}
EOF
