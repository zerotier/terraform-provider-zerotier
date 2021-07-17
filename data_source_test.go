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
