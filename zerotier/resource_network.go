package zerotier

import (
	"context"
	//	"net"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	zt "github.com/someara/terraform-provider-zerotier/zerotier-client"
)

func resourceNetwork() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNetworkCreate,
		ReadContext:   resourceNetworkRead,
		DeleteContext: resourceNetworkDelete,
		UpdateContext: resourceNetworkUpdate,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Managed by Terraform",
			},
			"rules_source": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"private": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"auto_assign_v4": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"routes": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"target": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"via": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"tags": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{},
			},
			"ip_assignment_pools": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ip_range_start": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"ip_range_end": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			"v4_assign_mode": &schema.Schema{
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"zt": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			"v6_assign_mode": &schema.Schema{
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"zt": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
						},
						"six_plane": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
						},
						"rfc_4193": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			"permissions": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"owner_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"online_member_count": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			"authorized_member_count": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			"total_member_count": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			"capabilities_by_name": &schema.Schema{
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"ui": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceNetworkRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*zt.Client)
	var diags diag.Diagnostics

	zerotier_network_id := d.Id()
	zerotier_network, err := c.GetNetwork(zerotier_network_id)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("authorized_member_count", zerotier_network.AuthorizedMemberCount)
	d.Set("auto_assign_v4", zerotier_network.Config.V4AssignMode.ZT)
	d.Set("capabilities_by_name", zerotier_network.CapabilitiesByName)
	d.Set("description", zerotier_network.Description)
	d.Set("ip_assignment_pools", zerotier_network.Config.IpAssignmentPools)
	d.Set("name", zerotier_network.Config.Name)
	d.Set("online_member_count", zerotier_network.OnlineMemberCount)
	d.Set("owner_id", zerotier_network.OwnerId)
	d.Set("permissions", zerotier_network.Permissions)
	d.Set("private", zerotier_network.Config.Private)
	d.Set("routes", zerotier_network.Config.Routes)
	d.Set("rules_source", zerotier_network.RulesSource)
	d.Set("tags", zerotier_network.Config.Tags)
	d.Set("ui", zerotier_network.Ui)
	d.Set("v4_assign_mode", zerotier_network.Config.V4AssignMode)
	d.Set("v6_assign_mode", zerotier_network.Config.V6AssignMode)

	return diags
}

func resourceNetworkCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*zt.Client)
	var diags diag.Diagnostics

	network, err := networkInit(d)
	if err != nil {
		return diag.FromErr(err)
	}

	cn, err := c.CreateNetwork(network)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(cn.Id)

	resourceNetworkRead(ctx, d, m)
	return diags
}

func resourceNetworkUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*zt.Client)

	n, err := networkInit(d)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = c.UpdateNetwork(d.Id(), n)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceNetworkRead(ctx, d, m)
}

func resourceNetworkDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*zt.Client)
	var diags diag.Diagnostics

	networkID := d.Id()

	err := c.DeleteNetwork(networkID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return diags
}

// func fromResourceData(d *schema.ResourceData) (*zt.Network, error) {
// 	routesRaw := d.Get("route").([]interface{})
// 	var routes []zt.Route
// 	for _, raw := range routesRaw {
// 		r := raw.(map[string]interface{})
// 		via := r["via"].(string)
// 		routes = append(routes, zt.Route{
// 			Target: r["target"].(string),
// 			Via:    &via,
// 		})
// 	}

// }

//
// helpers
//

func networkInit(d *schema.ResourceData) (*zt.Network, error) {
	rules_source := d.Get("rules_source").(string)
	description := d.Get("description").(string)
	name := d.Get("name").(string)
	private := d.Get("private").(bool)

	v4_assign_mode := d.Get("v4_assign_mode")
	v6_assign_mode := d.Get("v6_assign_mode")

	routes := d.Get("routes").([]interface{})
	ip_assignment_pools := d.Get("ip_assignment_pools").([]interface{})

	n := &zt.Network{
		Id:          d.Id(),
		RulesSource: rules_source,
		Description: description,
		Config: &zt.NetworkConfig{
			Name:    name,
			Private: private,
			V4AssignMode: zt.V4AssignMode{
				ZT: v4_assign_mode.zt,
			},
			V6AssignMode: zt.V6AssignMode{
				ZT:       v6_assign_mode.zt,
				SixPlane: v6_assign_mode.six_plane,
				Rfc4193:  v6_assign_mode.rfc_4193,
			},
			Routes:            routes,
			IpAssignmentPools: ip_assignment_pools,
		},
	}
	return n, nil
}

//
// coerce things
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
