package zerotier

import (
	// "log"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztcentral/pkg/spec"
)

func TestZeroTier_ToMember(t *testing.T) {

	d := schema.TestResourceDataRaw(t, resourceMember().Schema, map[string]interface{}{
		"network_id":     "11122334455aabbccdd",
		"ip_assignments": []interface{}{"10.10.10.10", "1.2.3.4"},
		"capabilities":   []interface{}{1, 2, 3},
		"tags":           []interface{}{[]interface{}{1, 2}, []interface{}{3, 4}},
		"member_id":      "2468012345",
		// "hidden": false,
		// "name": "a name",
		// "description": "a description",
		// "authorized": true,
		// "allow_ethernet_bridging": false,
		// "no_auto_assign_ips": false,
	})

	expectedNetworkId := "11122334455aabbccdd"
	expectedTags := [][]int{{1, 2}, {3, 4}}
	expectedCaps := []int{1, 2, 3}
	expectedIps := []string{"10.10.10.10", "1.2.3.4"}
	expected := &spec.Member{
		NetworkId: &expectedNetworkId,
		Config: &spec.MemberConfig{
			Tags:          &expectedTags,
			Capabilities:  &expectedCaps,
			IpAssignments: &expectedIps,
		},
	}
	out := toMember(d)

	assert.Equal(t, *expected.NetworkId, *out.NetworkId)
	assert.Equal(t, "2468012345", *out.NodeId)
	assert.ElementsMatch(t, *expected.Config.IpAssignments, *out.Config.IpAssignments)
	assert.ElementsMatch(t, *expected.Config.Tags, *out.Config.Tags)
	assert.ElementsMatch(t, *expected.Config.Capabilities, *out.Config.Capabilities)
}
