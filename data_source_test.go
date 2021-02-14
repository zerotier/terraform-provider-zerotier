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

		if m["type"] == "zerotier_network" && m["mode"] == "data" && m["name"] == "bob" {
			if attrs["name"] != "bobs_garage" {
				t.Fatal("name was not set on object")
			}

			if attrs["id"] != "" {
				t.Fatal("name was not set on object")
			}
		}

	}
}
