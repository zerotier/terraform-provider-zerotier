#!/bin/sh

rm -rf ${PWD}/.tfdata

cat >test.tfrc <<EOF
provider_installation {
  filesystem_mirror {
    path = "${PWD}/.tfdata"
  }

  direct {}
}
EOF
