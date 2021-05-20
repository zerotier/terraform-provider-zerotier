package zerotier

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztcentral/pkg/spec"
)

func ztMemberYield(vs ValidatedSchema) interface{} {
	return &spec.Member{
		NetworkId: stringPtr(vs.Get("network_id").(string)),
		NodeId:    stringPtr(vs.Get("member_id").(string)),
		Hidden:    boolPtr(vs.Get("hidden").(bool)),
		//OfflineNotifyDelay: toInt(d, "offline_notify_delay"),
		Name:        stringPtr(vs.Get("name").(string)),
		Description: stringPtr(vs.Get("description").(string)),
		Config: &spec.MemberConfig{
			Authorized:      boolPtr(vs.Get("authorized").(bool)),
			ActiveBridge:    boolPtr(vs.Get("allow_ethernet_bridging").(bool)),
			NoAutoAssignIps: boolPtr(vs.Get("no_auto_assign_ips").(bool)),
			Capabilities:    fetchIntList(vs, "capabilities"),
			IpAssignments:   fetchStringList(vs, "ip_assignments"),
		},
	}
}

func ztMemberCollect(vs ValidatedSchema, d *schema.ResourceData, i interface{}, force bool) diag.Diagnostics {
	ztMember := i.(*spec.Member)

	var diags diag.Diagnostics

	stuff := map[string]interface{}{
		"name":                    ztMember.Name,
		"description":             ztMember.Description,
		"member_id":               ztMember.NodeId,
		"network_id":              ztMember.NetworkId,
		"hidden":                  ztMember.Hidden,
		"authorized":              ztMember.Config.Authorized,
		"allow_ethernet_bridging": ztMember.Config.ActiveBridge,
		"no_auto_assign_ips":      ztMember.Config.NoAutoAssignIps,
		"ip_assignments":          ztMember.Config.IpAssignments,
		"capabilities":            ztMember.Config.Capabilities,
	}

	for key, value := range stuff {
		if d.HasChange(key) || force {
			if err := vs.Set(d, key, value); err != nil {
				diags = append(diags, err...)
			}
		}
	}

	return diags
}

// NewMember creates a new member schema
func NewMember() ValidatedSchema {
	// ZTMember is our internal validated schema. See schemawrap.go.
	return ValidatedSchema{
		YieldFunc:   ztMemberYield,
		CollectFunc: ztMemberCollect,
		Schema: map[string]*SchemaWrap{
			"network_id": {
				Schema: &schema.Schema{
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "ID of the network this member belongs to",
				},
			},
			"member_id": {
				Schema: &schema.Schema{
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "ID of this member.",
				},
			},
			"name": {
				Schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Computed:    true,
					Description: "Descriptive name of this member.",
				},
			},
			"description": {
				Schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "Managed by Terraform",
					Description: "Text description of this member.",
				},
			},
			"hidden": {
				Schema: &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
					Description: "Is this member visible?",
				},
			},
			"authorized": {
				Schema: &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     true,
					Description: "Is the member authorized on the network?",
				},
			},
			"allow_ethernet_bridging": {
				Schema: &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
					Description: "Is this member allowed to activate ethernet bridging over the ZeroTier network?",
				},
			},
			"no_auto_assign_ips": {
				Schema: &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
					Description: "Exempt this member from the IP auto assignment pool on a Network",
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
					Description: "List of IP address assignments",
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
					Description: "List of network capabilities",
				},
			},
		},
	}
}
