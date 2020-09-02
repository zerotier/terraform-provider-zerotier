package zerotier

import (
	"fmt"
	"strconv"
	"strings"
	"context"

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
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"capabilities": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
			},
			"tags": {
				Type:     schema.TypeMap,
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

func resourceMemberCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*zt.Client)

	// Warning or errors can be collected in a slice type
        var diags diag.Diagnostics

	stored, err := memberFromResourceData(d)
	if err != nil {
		return diag.FromErr(err)
	}
	
	created, err := c.CreateMember(stored)
	if err != nil {
		return diag.FromErr(err)
	}
	
	d.SetId(created.Id)
	setTags(d, created)
	return diags
}

func resourceMemberRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*zt.Client)

        // Warning or errors can be collected in a slice type
        var diags diag.Diagnostics
	
	// Attempt to read from an upstream API
	nwid, nodeId := resourceNetworkAndNodeIdentifiers(d)
	member, err := c.GetMember(nwid, nodeId)

	// If the resource does not exist, inform Terraform. We want to immediately
	// return here to prevent further processing.
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
	d.Set("node_id", nodeId)
	d.Set("network_id", nwid)
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

func resourceMemberUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*zt.Client)

        // Warning or errors can be collected in a slice type
        var diags diag.Diagnostics
	
	stored, err := memberFromResourceData(d)
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
	
	member, err := memberFromResourceData(d)
	if err != nil {
		return diag.FromErr(err)
	}
	
	err = c.DeleteMember(member)
	return diags
}

func setTags(d *schema.ResourceData, member *zt.Member) {
	rawTags := map[string]int{}
	for _, tuple := range member.Config.Tags {
		key := fmt.Sprintf("%d", tuple[0])
		val := tuple[1]
		rawTags[key] = val
	}
}

func memberFromResourceData(d *schema.ResourceData) (*zt.Member, error) {
	tags := d.Get("tags").(map[string]interface{})
	tagTuples := [][]int{}
	for key, val := range tags {
		i, err := strconv.Atoi(key)
		if err != nil {
			break
		}
		tagTuples = append(tagTuples, []int{i, val.(int)})
	}
	capsRaw := d.Get("capabilities").([]interface{})
	caps := make([]int, len(capsRaw))
	for i := range capsRaw {
		caps[i] = capsRaw[i].(int)
	}
	ipsRaw := d.Get("ip_assignments").([]interface{})
	ips := make([]string, len(ipsRaw))
	for i := range ipsRaw {
		ips[i] = ipsRaw[i].(string)
	}
	n := &zt.Member{
		Id:                 d.Id(),
		NetworkId:          d.Get("network_id").(string),
		NodeId:             d.Get("node_id").(string),
		Hidden:             d.Get("hidden").(bool),
		OfflineNotifyDelay: d.Get("offline_notify_delay").(int),
		Name:               d.Get("name").(string),
		Description:        d.Get("description").(string),
		Config: &zt.MemberConfig{
			Authorized:      d.Get("authorized").(bool),
			ActiveBridge:    d.Get("allow_ethernet_bridging").(bool),
			NoAutoAssignIps: d.Get("no_auto_assign_ips").(bool),
			Capabilities:    caps,
			Tags:            tagTuples,
			IpAssignments:   ips,
		},
	}
	return n, nil
}

// Extracts the Network ID and Node ID from the resource definition, or from the id during import
//
// When importing a resource, both the network id and node id writen on the definition will be ignored
// and we could retrieve the network id and node id from parts of the id
// which is formated as <network-id>-<node-id> on zerotier
func resourceNetworkAndNodeIdentifiers(d *schema.ResourceData) (string, string) {
	nwid := d.Get("network_id").(string)
	nodeID := d.Get("node_id").(string)

	if nwid == "" && nodeID == "" {
		parts := strings.Split(d.Id(), "-")
		nwid, nodeID = parts[0], parts[1]
	}
	return nwid, nodeID
}
