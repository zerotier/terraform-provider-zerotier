package zerotier

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztcentral/pkg/spec"
)

// DataSourceMemberSchema is the zerotier_member data source schema
var DataSourceMemberSchema = map[string]*schema.Schema{
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
}

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
		Type:     schema.TypeList,
		Computed: true,
		Optional: true,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
		Description: "List of IP address assignments",
	},
	"capabilities": {
		Type:     schema.TypeList,
		Computed: true,
		Optional: true,
		Elem: &schema.Schema{
			Type: schema.TypeInt,
		},
		Description: "List of network capabilities",
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
			Capabilities:    fetchIntList(d, "capabilities"),
			IpAssignments:   fetchStringList(d, "ip_assignments"),
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

	return nil
}
