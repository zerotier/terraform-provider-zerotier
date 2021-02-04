package main

import (
	"testing"

	"github.com/erikh/tftest"
)

// these are deliberately named to keep the code small. they do not add
// anything else. Hopefully we won't need this soon.
func h(m interface{}) map[string]interface{} {
	return m.(map[string]interface{})
}

func a(m interface{}) []interface{} {
	return m.([]interface{})
}

func s(m interface{}) string {
	return m.(string)
}

type identity struct {
	name    string
	pubkey  string
	privkey string
}

func extractIdentities(tf tftest.Harness) map[string]identity {
	resources := a(tf.State()["resources"])

	ids := map[string]identity{}

	for _, resource := range resources {
		res := h(resource)
		name := s(res["name"])

		ids[name] = identity{
			name:    name,
			pubkey:  s(h(h(a(res["instances"])[0])["attributes"])["public_key"]),
			privkey: s(h(h(a(res["instances"])[0])["attributes"])["private_key"]),
		}
	}

	return ids
}

func TestIdentity(t *testing.T) {
	tf := tftest.New(t)
	tf.Apply("testdata/plans/basic-identity.tf")

	ids := extractIdentities(tf)

	if len(ids) != 2 {
		t.Fatalf("invalid count of identities created")
	}

	for _, name := range []string{"alice", "bob"} {
		if _, ok := ids[name]; !ok {
			t.Fatalf("%q was not present in state", name)
		}

		if ids[name].pubkey == "" {
			t.Fatalf("%q: pubkey empty", name)
		}

		if ids[name].privkey == "" {
			t.Fatalf("%q: pubkey empty", name)
		}
	}
}

func TestBasicNetworkSetup(t *testing.T) {
	t.Skip("test fails because we need to upgrade the client")
	tf := tftest.New(t)
	tf.Apply("testdata/plans/basic-network.tf")
}
