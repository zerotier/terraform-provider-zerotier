package zerotier

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
)

func Test_ResourceNetworkAndNodeIdentifiers_PresetValues(t *testing.T) {
	tests := []struct {
		desc           string
		inputNetworkID string
		inputMemberID  string
	}{
		{
			desc:           "Preset values",
			inputNetworkID: "0xcafe", inputMemberID: "0xbaby",
		},
		{
			desc:           "Missing network ID",
			inputNetworkID: "", inputMemberID: "0xbaby",
		},
		{
			desc:           "Missing member ID",
			inputNetworkID: "0xcafe", inputMemberID: "",
		},
	}

	res := schema.Resource{Schema: buildMemberSchema(true)}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			d := res.TestResourceData()
			d.Set("network_id", test.inputNetworkID)
			d.Set("member_id", test.inputMemberID)

			nwid, nodeID, err := resourceNetworkAndNodeIdentifiers(d)
			assert.NoError(t, err)
			assert.Equal(t, test.inputNetworkID, nwid)
			assert.Equal(t, test.inputMemberID, nodeID)
		})
	}
}

func Test_ResourceNetworkAndNodeIdentifiers_ParsingMemberID(t *testing.T) {
	tests := []struct {
		desc    string
		inputId string

		expectedErrPattern string
		expectedNetworkID  string
		expectedNodeId     string
	}{
		{
			desc:               "Wrong syntax",
			inputId:            "0xcafe",
			expectedErrPattern: "invalid format.*(wrong syntax)",
		},
		{
			desc:               "Missing network id",
			inputId:            "0xcafe-",
			expectedErrPattern: "invalid format.*(all components are required)",
		},
		{
			desc:               "Missing node id",
			inputId:            "-0xbaby",
			expectedErrPattern: "invalid format.*(all components are required)",
		},
		{
			desc:              "Wellformed",
			inputId:           "0xcafe-0xbaby",
			expectedNetworkID: "0xcafe", expectedNodeId: "0xbaby",
		},
	}

	res := schema.Resource{Schema: buildMemberSchema(true)}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			d := res.TestResourceData()
			d.SetId(test.inputId)

			nwid, nodeID, err := resourceNetworkAndNodeIdentifiers(d)

			if test.expectedErrPattern != "" {
				assert.Error(t, err)
				assert.Regexp(t, test.expectedErrPattern, err.Error())
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, test.expectedNetworkID, nwid)
			assert.Equal(t, test.expectedNodeId, nodeID)
		})
	}
}
