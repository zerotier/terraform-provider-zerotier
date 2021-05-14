package main

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/zerotier/go-ztcentral"
)

func boolPtr(b bool) *bool {
	return &b
}

// this is slightly different than the converters.go version; the result coming
// back from json parsing will always be `float64`, instead of `int`, which we
// pass normally through the type system in converters. See
// pkg/zerotier/converters.go for more information.
func toUintList(i interface{}) []uint {
	ray := []uint{}
	for _, x := range i.([]interface{}) {
		ray = append(ray, uint(x.(float64))) // here
	}
	return ray
}

func modifyMember(ctx context.Context, networkID string, memberID string, updateFunc func(*ztcentral.Member)) error {
	c := ztcentral.NewClient(controllerToken)
	member, err := c.GetMember(ctx, networkID, memberID)
	if err != nil {
		return err
	}

	updateFunc(member)

	if _, err := c.UpdateMember(ctx, member); err != nil {
		return err
	}

	return nil
}

func TestMemberUpdate(t *testing.T) {
	// see TestNetworkUpdate for the flow of this test.
	tf := getTFTest(t)
	tf.Apply("testdata/plans/basic-member.tf")

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)

	// for each network, perform a set of transformations with the client.
	for _, resource := range a(tf.State()["resources"]) {
		m := h(resource)
		attrs := h(h(a(m["instances"])[0])["attributes"])

		switch m["type"] {
		case "zerotier_member":
			switch m["name"] {
			case "alice":
				err := modifyMember(ctx, attrs["network_id"].(string), attrs["member_id"].(string), func(member *ztcentral.Member) {
					member.Description = "This is a new description"
					member.Hidden = false
					member.Config.ActiveBridge = false
					member.Config.NoAutoAssignIPs = false
					member.Config.IPAssignments = []string{"10.0.0.2"}
					member.Config.Capabilities = []uint{0, 1, 2}
				})
				if err != nil {
					t.Fatal(err)
				}
			default:
				t.Fatalf("invalid member %q encountered", m["name"])
			}
		}
	}

	tf.Refresh()

	// for each network, validate the transformations were applied.
	for _, resource := range a(tf.State()["resources"]) {
		m := h(resource)
		attrs := h(h(a(m["instances"])[0])["attributes"])

		switch m["type"] {
		case "zerotier_member":
			switch m["name"] {
			case "alice":
				if attrs["description"].(string) != "This is a new description" {
					t.Fatal("description was not set")
				}

				isBool(t, attrs["hidden"], false, "hidden")
				isBool(t, attrs["allow_ethernet_bridging"], false, "allow_ethernet_bridging")
				isBool(t, attrs["no_auto_assign_ips"], false, "no_auto_assign_ips")

				if a(attrs["ip_assignments"])[0].(string) != "10.0.0.2" {
					t.Fatal("ip_assignments was improperly set")
				}

				if !reflect.DeepEqual(toUintList(a(attrs["capabilities"])), []uint{0, 1, 2}) {
					t.Fatalf("capabilities was improperly set: is %#v", a(attrs["capabilities"]))
				}
			default:
				t.Fatalf("invalid member %q encountered", m["name"])
			}
		}
	}

	// reset the settings with terraform. Did it work?
	tf.Apply("testdata/plans/basic-member.tf")

	// for each network, validate the transformations were applied.
	for _, resource := range a(tf.State()["resources"]) {
		m := h(resource)
		attrs := h(h(a(m["instances"])[0])["attributes"])

		switch m["type"] {
		case "zerotier_member":
			switch m["name"] {
			case "alice":
				if attrs["description"].(string) != "Hello, world" {
					t.Fatalf("description was not set; was %q", attrs["description"])
				}

				isBool(t, attrs["hidden"], true, "hidden")
				isBool(t, attrs["allow_ethernet_bridging"], true, "allow_ethernet_bridging")
				isBool(t, attrs["no_auto_assign_ips"], true, "no_auto_assign_ips")

				if !reflect.DeepEqual(toUintList(a(attrs["capabilities"])), []uint{1, 2, 3}) {
					t.Fatalf("capabilities was improperly set: is %#v", a(attrs["capabilities"]))
				}

				if a(attrs["ip_assignments"])[0].(string) != "10.0.0.1" {
					t.Fatal("ip_assignments was improperly set")
				}
			default:
				t.Fatalf("invalid member %q encountered", m["name"])
			}
		}
	}
}

func modifyNetwork(ctx context.Context, id string, updateFunc func(*ztcentral.Network)) error {
	c := ztcentral.NewClient(controllerToken)
	net, err := c.GetNetwork(ctx, id)
	if err != nil {
		return err
	}

	updateFunc(net)

	if _, err := c.UpdateNetwork(ctx, net); err != nil {
		return err
	}

	return nil
}

func TestNetworkUpdate(t *testing.T) {
	// this test uses the same plan as BasicNetwork, but then inverts some values
	// back to defaults in each one to ensure that terraform picks them up on
	// refresh.
	tf := getTFTest(t)
	tf.Apply("testdata/plans/basic-network.tf")

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)

	// for each network, perform a set of transformations with the client.
	for _, resource := range a(tf.State()["resources"]) {
		m := h(resource)
		attrs := h(h(a(m["instances"])[0])["attributes"])

		switch m["type"] {
		case "zerotier_network":
			switch m["name"] {
			// XXX please, before you modify any of this, read the comments in provision_test.go.
			case "mtu":
				// not updateable
			case "multicast_limit":
				// not updateable
			case "description":
				err := modifyNetwork(ctx, attrs["id"].(string), func(net *spec.Network) {
					net.Description = nil
				})

				if err != nil {
					t.Fatal(err)
				}
			case "assign_off":
				err := modifyNetwork(ctx, attrs["id"].(string), func(net *ztcentral.Network) {
					net.Config.IPV4AssignMode = &ztcentral.IPV4AssignMode{ZeroTier: boolPtr(true)}
					net.Config.IPV6AssignMode = &ztcentral.IPV6AssignMode{ZeroTier: boolPtr(true), ZT6Plane: boolPtr(false), RFC4193: boolPtr(false)}
				})

				if err != nil {
					t.Fatal(err)
				}
			case "private":
				err := modifyNetwork(ctx, attrs["id"].(string), func(net *ztcentral.Network) {
					net.Config.Private = boolPtr(false)
				})

				if err != nil {
					t.Fatal(err)
				}
			case "no_broadcast":
				err := modifyNetwork(ctx, attrs["id"].(string), func(net *ztcentral.Network) {
					net.Config.EnableBroadcast = boolPtr(true)
				})

				if err != nil {
					t.Fatal(err)
				}
			case "flow_rules":
				c := ztcentral.NewClient(controllerToken)
				rules, err := c.UpdateNetworkRules(ctx, attrs["id"].(string), "accept;")
				if err != nil {
					t.Fatal(err)
				}

				if rules != "accept;" {
					t.Fatal("rules were not alterered on set")
				}
			case "alice", "bobs_garage":
				// this is a collection of defaults; not sure testing this is really worth the effort.
			default:
				t.Fatalf("Unexpected network %q in plan", m["name"])
			}
		}
	}

	tf.Refresh()

	// for each network, validate the transformations were applied.
	for _, resource := range a(tf.State()["resources"]) {
		m := h(resource)
		attrs := h(h(a(m["instances"])[0])["attributes"])

		switch m["type"] {
		case "zerotier_network":
			switch m["name"] {
			// XXX please, before you modify any of this, read the comments in provision_test.go.
			case "mtu":
				// not updateable
			case "multicast_limit":
				// not updateable
			case "description":
				if attrs["description"].(string) != "My description is changed!" {
					t.Fatal("description was not updated")
				}
			case "flow_rules":
				if attrs["flow_rules"].(string) != "accept;" {
					t.Fatal("flow_rules were not updated")
				}
			case "assign_off":
				isBool(t, h(attrs["assign_ipv4"])["zerotier"], true, "assign_ipv4/zerotier")

				table := map[string]bool{
					"zerotier": true,
					"sixplane": false,
					"rfc4193":  false,
				}

				for name, val := range table {
					isBool(t, h(attrs["assign_ipv6"])[name], val, "assign_ipv6/"+name)
				}
			case "private":
				isBool(t, attrs["private"], false, "private")
			case "no_broadcast":
				isBool(t, attrs["enable_broadcast"], true, "private")
			case "alice", "bobs_garage":
				// this is a collection of defaults; not sure testing this is really worth the effort.
			default:
				t.Fatalf("Unexpected network %q in plan", m["name"])
			}
		}
	}
}
