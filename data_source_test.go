package main

import (
	"testing"
)

func TestDataSourceNetwork(t *testing.T) {
	tf := getTFTest(t)
	tf.Apply("testdata/plans/data-source-network.tf")
	tf.Refresh()

	for _, data := range a(tf.State()["resources"]) {
		m := h(data)
		attrs := h(h(a(m["instances"])[0])["attributes"])

		if m["type"] == "zerotier_network" {
			if attrs["name"].(string) != "bobs_garage" {
				t.Fatal("name was not set on object, was:", attrs["name"])
			}

			if attrs["description"].(string) != "so say we bob" {
				t.Fatal("description was incorrect, was:", attrs["description"])
			}

			if attrs["id"].(string) == "" {
				t.Fatal("id was not set on object, was", attrs["id"])
			}
		}

	}
}

func TestDataSourceMembers(t *testing.T) {
	tf := getTFTest(t)

	tf.Apply("testdata/plans/data-source-members.tf")
	tf.Refresh()

	outputs := h(tf.State()["outputs"])

	memberName := s(h(outputs["member_name"])["value"])
	memberNameExp := "bobs_car"
	if memberName != memberNameExp {
		t.Fatalf("member name was not set correctly: expected: %q, found: %q", memberNameExp, memberName)
	}

	memberDesc := s(h(outputs["member_description"])["value"])
	memberDescExp := "bobs shiny car"
	if memberDesc != memberDescExp {
		t.Fatalf("member description was not set correctly: expected: %q, found: %q", memberDescExp, memberDesc)
	}
}
