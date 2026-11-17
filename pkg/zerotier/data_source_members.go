package zerotier

import (
	"context"
	"crypto/md5"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztcentral"
	"github.com/zerotier/go-ztcentral/pkg/spec"
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

	nwid := d.Get("network_id").(string)

	networkMembers, err := c.GetMembers(ctx, nwid)
	if err != nil {
		return diag.FromErr(err)
	}
	members := make([]map[string]interface{}, 0, len(networkMembers))
	memberIDs := make([]string, 0, len(networkMembers))
	for _, member := range networkMembers {
		cfg := member.Config
		if cfg == nil {
			cfg = &spec.MemberConfig{}
		}
		nodeID := deref(member.NodeId)

		ipv4Assignments, ipv6Assignments := assignedIpsGrouping(deref(cfg.IpAssignments))
		members = append(members, map[string]interface{}{
			"name":                    deref(member.Name),
			"description":             deref(member.Description),
			"member_id":               nodeID,
			"network_id":              deref(member.NetworkId),
			"hidden":                  deref(member.Hidden),
			"authorized":              deref(cfg.Authorized),
			"sso_exempt":              deref(cfg.SsoExempt),
			"allow_ethernet_bridging": deref(cfg.ActiveBridge),
			"no_auto_assign_ips":      deref(cfg.NoAutoAssignIps),
			"ip_assignments":          deref(cfg.IpAssignments),
			"capabilities":            deref(cfg.Capabilities),
			"tags":                    deref(cfg.Tags),
			"ipv4_assignments":        ipv4Assignments,
			"ipv6_assignments":        ipv6Assignments,
			"rfc4193":                 rfc4193Address(nwid, nodeID),
			"sixplane":                sixPlaneAddress(nwid, nodeID),
		})
		memberIDs = append(memberIDs, nodeID)
	}
	err = d.Set("members", members)
	if err != nil {
		return diag.FromErr(err)
	}
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
