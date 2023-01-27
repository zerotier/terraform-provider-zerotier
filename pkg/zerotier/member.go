package zerotier

import (
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztcentral/pkg/spec"
)

// MemberSchema is the zerotier_member resource schema.
var MemberSchema = map[string]*schema.Schema{
	"network_id": {
		Type:        schema.TypeString,
		Required:    true,
		ForceNew:    true,
		Description: "ID of the network this member belongs to",
	},
	"member_id": {
		Type:        schema.TypeString,
		Required:    true,
		ForceNew:    true,
		Description: "ID of this member.",
	},
	"name": {
		Type:        schema.TypeString,
		Optional:    true,
		Computed:    true,
		Description: "Descriptive name of this member.",
	},
	"description": {
		Type:        schema.TypeString,
		Optional:    true,
		Default:     "Managed by Terraform",
		Description: "Text description of this member.",
	},
	"hidden": {
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     false,
		Description: "Is this member visible?",
	},
	"authorized": {
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     true,
		Description: "Is the member authorized on the network?",
	},
	"allow_ethernet_bridging": {
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     false,
		Description: "Is this member allowed to activate ethernet bridging over the ZeroTier network?",
	},
	"no_auto_assign_ips": {
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     false,
		Description: "Exempt this member from the IP auto assignment pool on a Network",
	},
	"ip_assignments": {
		Type:     schema.TypeSet,
		Computed: true,
		Optional: true,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
		Description: "List of IP address assignments",
	},
	"capabilities": {
		Type:     schema.TypeSet,
		Computed: true,
		Optional: true,
		Elem: &schema.Schema{
			Type: schema.TypeInt,
		},
		Description: "List of network capabilities",
	},
	"tags": {
		Type:     schema.TypeSet,
		Computed: true,
		Optional: true,
		Elem: &schema.Schema{
			Type: schema.TypeList,
			Elem: &schema.Schema{
				Type: schema.TypeInt,
			},
		},
		Description: "List of network tags",
	},
	"ipv4_assignments": {
		Type:     schema.TypeSet,
		Computed: true,
		Optional: true,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
		Description: "ZeroTier managed IPv4 addresses.",
	},
	"ipv6_assignments": {
		Type:     schema.TypeSet,
		Computed: true,
		Optional: true,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
		Description: "ZeroTier managed IPv6 addresses.",
	},
	"sixplane": {
		Type:        schema.TypeString,
		Optional:    true,
		Computed:    true,
		Description: "Computed 6PLANE address. assign_ipv6.sixplane must be enabled on the network resource.",
	},
	"rfc4193": {
		Type:        schema.TypeString,
		Optional:    true,
		Computed:    true,
		Description: "Computed RFC4193 address. assign_ipv6.rfc4193 must be enabled on the network resource.",
	},
}

func toMember(d *schema.ResourceData) *spec.Member {
	return &spec.Member{
		NetworkId: stringPtr(d.Get("network_id").(string)),
		NodeId:    stringPtr(d.Get("member_id").(string)),
		Hidden:    boolPtr(d.Get("hidden").(bool)),
		//OfflineNotifyDelay: toInt(d, "offline_notify_delay"),
		Name:        stringPtr(d.Get("name").(string)),
		Description: stringPtr(d.Get("description").(string)),
		Config: &spec.MemberConfig{
			Authorized:      boolPtr(d.Get("authorized").(bool)),
			ActiveBridge:    boolPtr(d.Get("allow_ethernet_bridging").(bool)),
			NoAutoAssignIps: boolPtr(d.Get("no_auto_assign_ips").(bool)),
			Capabilities:    fetchIntSet(d, "capabilities"),
			IpAssignments:   fetchStringSet(d, "ip_assignments"),
			Tags:            fetchTags(d.Get("tags").(*schema.Set).List()),
		},
	}
}

func memberToTerraform(d *schema.ResourceData, m *spec.Member) diag.Diagnostics {
	d.SetId(strings.Join([]string{*m.NetworkId, *m.NodeId}, "/"))

	d.Set("name", *m.Name)
	d.Set("description", *m.Description)
	d.Set("member_id", *m.NodeId)
	d.Set("network_id", *m.NetworkId)
	d.Set("hidden", *m.Hidden)
	d.Set("authorized", *m.Config.Authorized)
	d.Set("allow_ethernet_bridging", *m.Config.ActiveBridge)
	d.Set("no_auto_assign_ips", *m.Config.NoAutoAssignIps)
	d.Set("ip_assignments", *m.Config.IpAssignments)
	d.Set("capabilities", *m.Config.Capabilities)
	d.Set("tags", *m.Config.Tags)

	ipv4Assignments, ipv6Assignments := assignedIpsGrouping(*m.Config.IpAssignments)
	d.Set("ipv4_assignments", ipv4Assignments)
	d.Set("ipv6_assignments", ipv6Assignments)

	nwid, nodeID, err := resourceNetworkAndNodeIdentifiers(d)
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("rfc4193", rfc4193Address(nwid, nodeID))
	d.Set("sixplane", sixPlaneAddress(nwid, nodeID))

	return nil
}

func sixPlaneAddress(nwid, nodeID string) string {
	return buildIPV6("fd" + nwid + "9993" + nodeID)
}

func rfc4193Address(nwid, nodeID string) string {
	nwidInt, _ := strconv.ParseUint(nwid, 16, 64)
	networkMask := uint32((nwidInt >> 32) ^ nwidInt)
	networkPrefix := strconv.FormatUint(uint64(networkMask), 16)
	return buildIPV6("fc" + networkPrefix + nodeID + "000000000001")
}

// Receive a string and format every 4th element with a ":"
func buildIPV6(data string) (result string) {
	s := strings.SplitAfter(data, "")
	end := len(s) - 1
	result = ""
	for i, s := range s {
		result += s
		if (i+1)%4 == 0 && i != end {
			result += ":"
		}
	}
	return
}

func assignedIpsGrouping(ipAssignments []string) (ipv4s []string, ipv6s []string) {
	for _, element := range ipAssignments {
		if strings.Contains(element, ":") {
			ipv6s = append(ipv6s, element)
		} else {
			ipv4s = append(ipv4s, element)
		}
	}
	return
}
