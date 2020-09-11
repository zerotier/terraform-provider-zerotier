package zerotier

import (
	"fmt"
	//	"strconv"
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	zt "github.com/someara/terraform-provider-zerotier/zerotier-client"
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
	c := m.(*zt.Client)
	var diags diag.Diagnostics

	zerotier_network_id, zerotier_node_id := resourceNetworkAndNodeIdentifiers(d)

	member, err := c.GetMember(zerotier_network_id, zerotier_node_id)
	if err != nil {
		return diag.FromErr(err)
	}

	if member == nil {
		d.SetId("")
		return nil
	}

	d.SetId(member.Id)
	d.Set("name", member.Name)
	d.Set("description", member.Description)
	d.Set("node_id", zerotier_node_id)
	d.Set("network_id", zerotier_network_id)
	d.Set("hidden", member.Hidden)
	d.Set("offline_notify_delay", member.OfflineNotifyDelay)
	d.Set("authorized", member.Config.Authorized)
	d.Set("allow_ethernet_bridging", member.Config.ActiveBridge)
	d.Set("no_auto_assign_ips", member.Config.NoAutoAssignIps)
	d.Set("ip_assignments", member.Config.IpAssignments)
	d.Set("capabilities", member.Config.Capabilities)
	setTags(d, member)

	return diags
}

func resourceMemberCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*zt.Client)
	var diags diag.Diagnostics

	member, err := memberInit(d)
	if err != nil {
		return diag.FromErr(err)
	}

	cm, err := c.CreateMember(member)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(cm.Id)
	setTags(d, cm)
	return diags
}

func resourceMemberUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*zt.Client)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	stored, err := memberInit(d)
	if err != nil {
		return diag.FromErr(err)
	}

	updated, err := c.UpdateMember(stored)
	if err != nil {
		return diag.FromErr(err)
	}

	setTags(d, updated)
	return diags
}

func resourceMemberDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*zt.Client)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	member, err := memberInit(d)
	if err != nil {
		return diag.FromErr(err)
	}

	err = c.DeleteMember(member)
	return diags
}

//
// helpers
//

func memberInit(d *schema.ResourceData) (*zt.Member, error) {
	m := &zt.Member{
		Id:                 d.Id(),
		NetworkId:          toString(d, "network_id"),
		NodeId:             toString(d, "node_id"),
		Hidden:             toBool(d, "hidden"),
		OfflineNotifyDelay: toInt(d, "offline_notify_delay"),
		Name:               toString(d, "name"),
		Description:        toString(d, "description"),
		Config: zt.MemberConfig{
			Authorized:      toBool(d, "authorized"),
			ActiveBridge:    toBool(d, "allow_ethernet_bridging"),
			NoAutoAssignIps: toBool(d, "no_auto_assign_ips"),
			Capabilities:    toIntList(d, "capabilities"),
			IpAssignments:   toStringList(d, "ip_assignments"),
		},
	}
	return m, nil
}

func setTags(d *schema.ResourceData, member *zt.Member) {
	rawTags := map[string]int{}
	for _, tuple := range member.Config.Tags {
		key := fmt.Sprintf("%d", tuple[0])
		val := tuple[1]
		rawTags[key] = val
	}
}

func resourceNetworkAndNodeIdentifiers(d *schema.ResourceData) (string, string) {
	zerotier_network_id := d.Get("network_id").(string)
	nodeID := d.Get("node_id").(string)

	if zerotier_network_id == "" && nodeID == "" {
		parts := strings.Split(d.Id(), "-")
		zerotier_network_id, nodeID = parts[0], parts[1]
	}
	return zerotier_network_id, nodeID
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

