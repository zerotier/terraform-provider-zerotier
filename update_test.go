package main

import (
	"context"
	"testing"
	"time"

	"github.com/zerotier/go-ztcentral"
)

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
				// not updateable
			case "assign_off":
				err := modifyNetwork(ctx, attrs["id"].(string), func(net *ztcentral.Network) {
					net.Config.IPV4AssignMode = ztcentral.IPV4AssignMode{ZeroTier: true}
					net.Config.IPV6AssignMode = ztcentral.IPV6AssignMode{ZeroTier: true, ZT6Plane: false, RFC4193: false}
				})

				if err != nil {
					t.Fatal(err)
				}
			case "private":
				err := modifyNetwork(ctx, attrs["id"].(string), func(net *ztcentral.Network) {
					net.Config.Private = false
				})

				if err != nil {
					t.Fatal(err)
				}
			case "no_broadcast":
				err := modifyNetwork(ctx, attrs["id"].(string), func(net *ztcentral.Network) {
					net.Config.EnableBroadcast = true
				})

				if err != nil {
					t.Fatal(err)
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
				// not updateable
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
