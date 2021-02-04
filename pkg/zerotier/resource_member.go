package zerotier

import (
	"fmt"
	//	"strconv"
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztcentral"
)

func resourceMember() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMemberCreate,
		ReadContext:   resourceMemberRead,
		UpdateContext: resourceMemberUpdate,
		DeleteContext: resourceMemberDelete,
		Schema: map[string]*schema.Schema{
			"network_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"node_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Managed by Terraform",
			},
			"hidden": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"offline_notify_delay": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
			"authorized": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"allow_ethernet_bridging": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"no_auto_assign_ips": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"ip_assignments": {
				Type:     schema.TypeList,
				Computed: true,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"capabilities": {
				Type:     schema.TypeList,
				Computed: true,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
			},
		},
	}
}

//
// CRUD
//

func resourceMemberRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ztcentral.Client)
	var diags diag.Diagnostics

	ztNetworkID, ztNodeID := resourceNetworkAndNodeIdentifiers(d)

	member, err := c.GetMember(ctx, ztNetworkID, ztNodeID)
	if err != nil {
		return diag.FromErr(err)
	}

	if member == nil {
		d.SetId("")
		return nil
	}

	d.SetId(member.ID)
	d.Set("name", member.Name)
	d.Set("description", member.Description)
	d.Set("node_id", ztNodeID)
	d.Set("network_id", ztNetworkID)
	d.Set("hidden", member.Hidden)
	//d.Set("offline_notify_delay", member.OfflineNotifyDelay)
	d.Set("authorized", member.Config.Authorized)
	d.Set("allow_ethernet_bridging", member.Config.ActiveBridge)
	d.Set("no_auto_assign_ips", member.Config.NoAutoAssignIPs)
	d.Set("ip_assignments", member.Config.IPAssignments)
	d.Set("capabilities", member.Config.Capabilities)
	setTags(d, member)

	return diags
}

func resourceMemberCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ztcentral.Client)
	var diags diag.Diagnostics

	member, err := memberInit(d)
	if err != nil {
		return diag.FromErr(err)
	}

	cm, err := c.CreateAuthorizedMember(ctx, member.NetworkID, member.NodeID, member.Name)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(cm.ID)
	setTags(d, cm)
	return diags
}

func resourceMemberUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ztcentral.Client)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	stored, err := memberInit(d)
	if err != nil {
		return diag.FromErr(err)
	}

	updated, err := c.UpdateMember(ctx, stored)
	if err != nil {
		return diag.FromErr(err)
	}

	setTags(d, updated)
	return diags
}

func resourceMemberDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ztcentral.Client)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	member, err := memberInit(d)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := c.DeleteMember(ctx, member); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

//
// helpers
//

func memberInit(d *schema.ResourceData) (*ztcentral.Member, error) {
	m := &ztcentral.Member{
		ID:        d.Id(),
		NetworkID: toString(d, "network_id"),
		NodeID:    toString(d, "node_id"),
		Hidden:    toBool(d, "hidden"),
		//OfflineNotifyDelay: toInt(d, "offline_notify_delay"),
		Name:        toString(d, "name"),
		Description: toString(d, "description"),
		Config: ztcentral.MemberConfig{
			Authorized:      toBool(d, "authorized"),
			ActiveBridge:    toBool(d, "allow_ethernet_bridging"),
			NoAutoAssignIPs: toBool(d, "no_auto_assign_ips"),
			//Capabilities:    toIntList(d, "capabilities"),
			IPAssignments: toStringList(d, "ip_assignments"),
		},
	}
	return m, nil
}

func setTags(d *schema.ResourceData, member *ztcentral.Member) {
	rawTags := map[string]uint{}
	for _, tuple := range member.Config.Tags {
		key := fmt.Sprintf("%d", tuple[0])
		val := tuple[1]
		rawTags[key] = val
	}
}

func resourceNetworkAndNodeIdentifiers(d *schema.ResourceData) (string, string) {
	ztNetworkID := d.Get("network_id").(string)
	nodeID := d.Get("node_id").(string)

	if ztNetworkID == "" && nodeID == "" {
		parts := strings.Split(d.Id(), "-")
		ztNetworkID, nodeID = parts[0], parts[1]
	}
	return ztNetworkID, nodeID
}

//
// helpers
//

func toStringList(d *schema.ResourceData, attr string) []string {
	raw := d.Get(attr).([]interface{})
	ray := make([]string, len(raw))
	for i := range raw {
		ray[i] = raw[i].(string)
	}
	return ray
}

func toIntList(d *schema.ResourceData, attr string) []int {
	raw := d.Get(attr).([]interface{})
	ray := make([]int, len(raw))
	for i := range raw {
		ray[i] = raw[i].(int)
	}
	return ray
}

func toString(d *schema.ResourceData, attr string) string {
	return d.Get(attr).(string)
}

func toInt(d *schema.ResourceData, attr string) int {
	return d.Get(attr).(int)
}

func toBool(d *schema.ResourceData, attr string) bool {
	return d.Get(attr).(bool)
}
