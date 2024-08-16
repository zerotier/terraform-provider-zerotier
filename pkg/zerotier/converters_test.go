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
		"network_id":              "11122334455aabbccdd",
		"ip_assignments":          []interface{}{"10.10.10.10", "1.2.3.4"},
		"capabilities":            []interface{}{1, 2, 3},
		"tags":                    []interface{}{[]interface{}{1, 2}, []interface{}{3, 4}},
		"member_id":               "2468012345",
		"authorized":              true,
		"sso_exempt":              true,
		"hidden":                  false,
		"name":                    "baub",
		"description":             "praise baub",
		"allow_ethernet_bridging": true,
		"no_auto_assign_ips":      false,
	})

	expectedNetworkId := "11122334455aabbccdd"
	expectedName := "baub"
	expectedDescription := "praise baub"
	expectedTags := [][]interface{}{{1, 2}, {3, 4}}
	expectedCaps := []int{1, 2, 3}
	expectedIps := []string{"10.10.10.10", "1.2.3.4"}
	expectedAuthorized := true
	expectedSsoExempt := true
	expectedHidden := false
	expectedAllowEthernetBridging := true
	expectedNoAutoAssignIps := false

	expected := &spec.Member{
		NetworkId:   &expectedNetworkId,
		Hidden:      &expectedHidden,
		Name:        &expectedName,
		Description: &expectedDescription,
		Config: &spec.MemberConfig{
			Authorized:      &expectedAuthorized,
			Tags:            &expectedTags,
			Capabilities:    &expectedCaps,
			IpAssignments:   &expectedIps,
			SsoExempt:       &expectedSsoExempt,
			ActiveBridge:    &expectedAllowEthernetBridging,
			NoAutoAssignIps: &expectedNoAutoAssignIps,
		},
	}
	out := toMember(d)

	assert.Equal(t, *expected.NetworkId, *out.NetworkId)
	assert.Equal(t, *expected.Hidden, *out.Hidden)
	assert.Equal(t, *expected.Name, *out.Name)
	assert.Equal(t, *expected.Description, *out.Description)
	assert.Equal(t, "2468012345", *out.NodeId)
	assert.ElementsMatch(t, *expected.Config.IpAssignments, *out.Config.IpAssignments)
	assert.ElementsMatch(t, *expected.Config.Tags, *out.Config.Tags)
	assert.ElementsMatch(t, *expected.Config.Capabilities, *out.Config.Capabilities)
	assert.Equal(t, *expected.Config.Authorized, *out.Config.Authorized)
	assert.Equal(t, *expected.Config.SsoExempt, *out.Config.SsoExempt)
	assert.Equal(t, *expected.Config.ActiveBridge, *out.Config.ActiveBridge)
	assert.Equal(t, *expected.Config.NoAutoAssignIps, *out.Config.NoAutoAssignIps)
}
