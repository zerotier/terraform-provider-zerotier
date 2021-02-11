package zerotier

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztcentral"
)

func ztMemberYield(vs ValidatedSchema) interface{} {
	return &ztcentral.Member{
		NetworkID: vs.Get("network_id").(string),
		MemberID:  vs.Get("member_id").(string),
		Hidden:    vs.Get("hidden").(bool),
		//OfflineNotifyDelay: toInt(d, "offline_notify_delay"),
		Name:        vs.Get("name").(string),
		Description: vs.Get("description").(string),
		Config: ztcentral.MemberConfig{
			Authorized:      vs.Get("authorized").(bool),
			ActiveBridge:    vs.Get("allow_ethernet_bridging").(bool),
			NoAutoAssignIPs: vs.Get("no_auto_assign_ips").(bool),
			//Capabilities:    toIntList(d, "capabilities"),
			IPAssignments: fetchStringList(vs, "ip_assignments"),
		},
	}
}

func ztMemberCollect(vs ValidatedSchema, d *schema.ResourceData, i interface{}) diag.Diagnostics {
	ztMember := i.(*ztcentral.Member)

	var diags diag.Diagnostics

	diags = append(diags, vs.Set(d, "name", ztMember.Name)...)
	diags = append(diags, vs.Set(d, "description", ztMember.Description)...)
	diags = append(diags, vs.Set(d, "member_id", ztMember.MemberID)...)
	diags = append(diags, vs.Set(d, "network_id", ztMember.NetworkID)...)
	diags = append(diags, vs.Set(d, "hidden", ztMember.Hidden)...)
	diags = append(diags, vs.Set(d, "authorized", ztMember.Config.Authorized)...)
	diags = append(diags, vs.Set(d, "allow_ethernet_bridging", ztMember.Config.ActiveBridge)...)
	diags = append(diags, vs.Set(d, "no_auto_assign_ips", ztMember.Config.NoAutoAssignIPs)...)
	diags = append(diags, vs.Set(d, "ip_assignments", ztMember.Config.IPAssignments)...)
	diags = append(diags, vs.Set(d, "capabilities", ztMember.Config.Capabilities)...)

	return diags
}

// ZTMember is our internal validated schema. See schemawrap.go.
var ZTMember = ValidatedSchema{
	YieldFunc:   ztMemberYield,
	CollectFunc: ztMemberCollect,
	Schema: map[string]*SchemaWrap{
		"network_id": {
			Schema: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
		"member_id": {
			Schema: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
		"name": {
			Schema: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
		"description": {
			Schema: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Managed by Terraform",
			},
		},
		"hidden": {
			Schema: &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
		"offline_notify_delay": {
			Schema: &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
		},
		"authorized": {
			Schema: &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
		},
		"allow_ethernet_bridging": {
			Schema: &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
		"no_auto_assign_ips": {
			Schema: &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
		"ip_assignments": {
			Schema: &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
		"capabilities": {
			Schema: &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
			},
		},
	},
}
