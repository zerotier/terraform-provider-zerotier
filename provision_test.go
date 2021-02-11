package main

import (
	"testing"

	"github.com/erikh/tftest"
)

// these are deliberately named to keep the code small. they do not add
// anything else.
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

func extractIdentities(tf *tftest.Harness) map[string]identity {
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
	tf := getTFTest(t)
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
	tf := getTFTest(t)
	tf.Apply("testdata/plans/basic-network.tf")

	for _, resource := range a(tf.State()["resources"]) {
		m := h(resource)
		attrs := h(h(a(m["instances"])[0])["attributes"])

		switch m["type"] {
		case "zerotier_network":
			switch m["name"] {
			// FIXME: missing/failing support for these test cases in the Client API
			//
			// What should happen when the API is updated so that these items can
			// be modified, is that these tests can be uncommented and that they
			// will automagically pass, because all the plumbing is already done
			// for you, presuming nothing moves, etc.
			//
			// PLEASE NOTE that the case statements /themselves/ must be left
			// available so they are exhausted in before the default statement, which
			// will fail the test for unknown networks. This is a safeguard to keep
			// extraneous stuff from landing in the test plan.
			case "mtu":
				// i, ok := attrs["mtu"].(float64)
				// if !ok {
				// 	t.Fatal("mtu was not set")
				// }
				//
				// if i != 1500 {
				// 	t.Fatalf("mtu was improperly set: %f", i)
				// }
			case "multicast_limit":
				// i, ok := attrs["multicast_limit"].(float64)
				// if !ok {
				// 	t.Fatal("multicast_limit was not set")
				// }
				//
				// if i != 50 {
				// 	t.Fatalf("multicast_limit was improperly set: %f", i)
				// }
			case "description":
				// s, ok := attrs["description"].(string)
				// if !ok {
				// 	t.Fatal("description was not set")
				// }
				//
				// if s != "My description is changed!" {
				// 	t.Fatalf("description was improperly set")
				// }
			case "assign_off":
				m, ok := attrs["assign_ipv4"]
				if !ok {
					t.Fatal("assign_ipv4 key was missing")
				}

				if b, ok := h(m)["zerotier"].(bool); !ok || b {
					t.Fatal("assign_ipv4/zerotier was not set to false")
				}

				m, ok = attrs["assign_ipv6"]
				if !ok {
					t.Fatal("assign_ipv6 key was missing")
				}

				if b, ok := h(m)["zerotier"].(bool); !ok || b {
					t.Fatal("assign_ipv6/zerotier was not set to false")
				}

				if b, ok := h(m)["sixplane"].(bool); !ok || !b {
					t.Fatal("assign_ipv6/sixplane was not set to true")
				}

				if b, ok := h(m)["rfc4193"].(bool); !ok || !b {
					t.Fatal("assign_ipv6/rfc4193 was not set to true")
				}
			case "private":
				b, ok := attrs["private"].(bool)
				if !ok {
					t.Fatal("private was not set")
				}

				if !b {
					t.Fatalf("private was improperly set")
				}
			case "no_broadcast":
				b, ok := attrs["enable_broadcast"].(bool)
				if !ok {
					t.Fatal("enable_broadcast was not set")
				}

				if b {
					t.Fatal("enable_broadcast was improperly set")
				}
			case "alice", "bobs_garage":
				if f, ok := attrs["creation_time"].(float64); !ok || f == 0 {
					t.Fatal("creation time for alice network was 0")
				}

				if f, ok := attrs["tf_last_updated"].(float64); !ok || f == 0 {
					t.Fatal("tf_last_updated (in terraform) for alice network was 0")
				}

				if b, ok := attrs["enable_broadcast"].(bool); !ok || !b {
					t.Fatal("enable_broadcast should be defaulted to true")
				}

				if b, ok := attrs["private"].(bool); !ok || b {
					t.Fatal("private should be defaulted to false")
				}

				m, ok := attrs["assign_ipv4"]
				if !ok {
					t.Fatal("assign_ipv4 key was missing")
				}

				if b, ok := h(m)["zerotier"].(bool); !ok || !b {
					t.Fatal("assign_ipv4/zerotier was not set to true")
				}

				m, ok = attrs["assign_ipv6"]
				if !ok {
					t.Fatal("assign_ipv6 key was missing")
				}

				if b, ok := h(m)["zerotier"].(bool); !ok || !b {
					t.Fatal("assign_ipv6/zerotier was not set to true")
				}

				if b, ok := h(m)["sixplane"].(bool); !ok || b {
					t.Fatal("assign_ipv6/sixplane was not set to false")
				}

				if b, ok := h(m)["rfc4193"].(bool); !ok || b {
					t.Fatal("assign_ipv6/rfc4193 was not set to false")
				}

				// FIXME needs patch to ztcentral
				// if f, ok := attrs["last_modified"].(float64); !ok || f == 0 {
				// 	t.Fatal("last modified (on zerotier) for alice network was 0")
				// }
			default:
				t.Fatalf("Unexpected network %q in plan", m["name"])
			}
		}
	}
}
