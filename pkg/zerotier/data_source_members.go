package zerotier

import (
	"context"
	"crypto/md5"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztcentral"
	"sort"
	"strings"
)

func dataSourceMembers() *schema.Resource {
	return &schema.Resource{
		Description: "Data source for ZeroTier members. This data source can be used to retrieve information about members of a ZeroTier network.",
		ReadContext: datasourceMemberRead,
		Schema: map[string]*schema.Schema{
			"network_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "ID of the network to retrieve members from.",
			},
			"members": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: buildMemberSchema(false),
				},
			},
		},
	}
}

func datasourceMemberRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ztcentral.Client)

	nwid, _, err := resourceNetworkAndNodeIdentifiers(d)
	if err != nil {
		return diag.FromErr(err)
	}

	networkMembers, err := c.GetMembers(ctx, nwid)
	if err != nil {
		return diag.FromErr(err)
	}
	members := make([]map[string]interface{}, 0, len(networkMembers))
	memberIDs := make([]string, 0, len(networkMembers))
	for _, member := range networkMembers {
		ipv4Assignments, ipv6Assignments := assignedIpsGrouping(*member.Config.IpAssignments)
		_, nodeID, err := resourceNetworkAndNodeIdentifiers(d)
		if err != nil {
			return diag.FromErr(err)
		}
		members = append(members, map[string]interface{}{
			"name":                    *member.Name,
			"description":             *member.Description,
			"member_id":               *member.NodeId,
			"network_id":              *member.NetworkId,
			"hidden":                  *member.Hidden,
			"authorized":              *member.Config.Authorized,
			"allow_ethernet_bridging": *member.Config.ActiveBridge,
			"no_auto_assign_ips":      *member.Config.NoAutoAssignIps,
			"ip_assignments":          *member.Config.IpAssignments,
			"capabilities":            *member.Config.Capabilities,
			"tags":                    *member.Config.Tags,
			"ipv4_assignments":        ipv4Assignments,
			"ipv6_assignments":        ipv6Assignments,
			"rfc4193":                 rfc4193Address(nwid, nodeID),
			"sixplane":                sixPlaneAddress(nwid, nodeID),
		})
		memberIDs = append(memberIDs, *member.NodeId)
	}
	d.Set("members", members)
	d.SetId(stringChecksum(strings.Join(memberIDs, "")))
	return nil
}

// stringChecksum takes a string and returns the checksum of the string.
func stringChecksum(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	bs := h.Sum(nil)

	return fmt.Sprintf("%x", bs)
}

func stringListChecksum(s []string) string {
	sort.Strings(s)
	return stringChecksum(strings.Join(s, ""))
}
