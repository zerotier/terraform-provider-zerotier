package main

import (
	"strings"
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

func isBool(t *testing.T, i interface{}, val bool, name string) {
	b, ok := i.(bool)
	if ok && b != val {
		t.Fatalf("%q was not set to %v", name, val)
	} else if !ok {
		b2, ok := i.(*bool)
		if ok && (b2 == nil || *b2 != val) {
			t.Fatalf("%q was not set to %v", name, val)
		} else if !ok {
			t.Fatalf("%q was not set properly", name)
		}
	}
}

func isNum(t *testing.T, i interface{}, val float64, name string) {
	if f, ok := i.(float64); !ok || f != val {
		t.Fatalf("%q was not set to %v", name, val)
	}
}

func isNonZeroNum(t *testing.T, i interface{}, name string) {
	if f, ok := i.(float64); !ok || f == 0 {
		t.Fatalf("%q was not set", name)
	}
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

func TestBasicMembers(t *testing.T) {
	tf := getTFTest(t)
	tf.Apply("testdata/plans/basic-member.tf")
	for _, resource := range a(tf.State()["resources"]) {
		m := h(resource)
		attrs := h(h(a(m["instances"])[0])["attributes"])

		switch m["type"] {
		case "zerotier_member":
			switch m["name"] {
			// see TestBasicNetworkSetup for examples on how this loop works.
			case "alice":
				if attrs["description"].(string) != "Hello, world" {
					t.Fatal("description was not set")
				}

				isBool(t, attrs["hidden"], true, "hidden")
				isBool(t, attrs["allow_ethernet_bridging"], true, "allow_ethernet_bridging")
				isBool(t, attrs["no_auto_assign_ips"], true, "no_auto_assign_ips")

				if a(attrs["ip_assignments"])[0].(string) != "10.0.0.1" {
					t.Fatal("ip_assignments was improperly set")
				}
			default:
				t.Fatalf("Unexpected network member %q in plan", m["name"])
			}
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
				s, ok := attrs["description"].(string)
				if !ok {
					t.Fatal("description was not set")
				}

				if s != "My description is changed!" {
					t.Fatalf("description was improperly set")
				}
			case "flow_rules":
				if attrs["flow_rules"].(string) != "drop;" {
					t.Fatal("flow_rules were not altered", attrs["flow_rules"])
				}
			case "assign_off":
				isBool(t, h(a(attrs["assign_ipv4"])[0])["zerotier"], false, "assign_ipv4/zerotier")

				table := map[string]bool{
					"zerotier": false,
					"sixplane": true,
					"rfc4193":  true,
				}

				for name, val := range table {
					isBool(t, h(a(attrs["assign_ipv6"])[0])[name], val, "assign_ipv6/"+name)
				}
			case "private":
				isBool(t, attrs["private"], true, "private")
			case "no_broadcast":
				isBool(t, attrs["enable_broadcast"], false, "enable_broadcast")
			case "alice", "bobs_garage":
				for _, name := range []string{"creation_time"} {
					isNonZeroNum(t, attrs[name], name)
				}

				isBool(t, attrs["enable_broadcast"], true, "enable_broadcast")
				isBool(t, attrs["private"], false, "private")

				m, ok := attrs["assign_ipv4"]
				if !ok {
					t.Fatal("assign_ipv4 key was missing")
				}

				isBool(t, h(a(m)[0])["zerotier"], true, "assign_ipv4/zerotier")

				m, ok = attrs["assign_ipv6"]
				if !ok {
					t.Fatal("assign_ipv6 key was missing")
				}

				table := map[string]bool{
					"zerotier": false,
					"sixplane": false,
					"rfc4193":  false,
				}

				for name, val := range table {
					isBool(t, h(a(m)[0])[name], val, "assign_ipv6/"+name)
				}

				if !strings.HasSuffix(strings.TrimSpace(attrs["flow_rules"].(string)), "accept;") {
					t.Fatal("flow_rules were not accept by default:", attrs["flow_rules"])
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
