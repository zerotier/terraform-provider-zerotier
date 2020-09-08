package zerotier

import (
	"context"
	"net"

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
			"route": &schema.Schema{
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
			"assignment_pool": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cidr": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
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
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"v6_assign_mode": &schema.Schema{
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
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

func resourceNetworkCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*zt.Client)
	var diags diag.Diagnostics

	n, err := fromResourceData(d)
	if err != nil {
		return diag.FromErr(err)
	}

	created, err := c.CreateNetwork(n)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(created.Id)
	return diags
}

func resourceNetworkRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*zt.Client)

	var diags diag.Diagnostics

	zerotier_network, err := c.GetNetwork(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if zerotier_network == nil {
		d.SetId("")
		return nil
	}

	d.Set("name", zerotier_network.Config.Name)
	d.Set("description", zerotier_network.Description)
	d.Set("private", zerotier_network.Config.Private)
	d.Set("auto_assign_v4", zerotier_network.Config.V4AssignMode.ZT)
	d.Set("rules_source", zerotier_network.RulesSource)

	// setRoutes(d, zerotier_network)
	// setAssignmentPools(d, zerotier_network)
	return diags
}

func resourceNetworkUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*zt.Client)

	n, err := fromResourceData(d)
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

func fromResourceData(d *schema.ResourceData) (*zt.Network, error) {

	routesRaw := d.Get("route").([]interface{})
	var routes []zt.Route
	for _, raw := range routesRaw {
		r := raw.(map[string]interface{})
		via := r["via"].(string)
		routes = append(routes, zt.Route{
			Target: r["target"].(string),
			Via:    &via,
		})
	}

	var pools []zt.IpRange
	for _, raw := range d.Get("assignment_pool").(*schema.Set).List() {
		r := raw.(map[string]interface{})
		cidr := r["cidr"].(string)
		first, last, err := zt.CIDRToRange(cidr)
		if err != nil {
			first = net.ParseIP(r["ip_range_start"].(string))
			last = net.ParseIP(r["ip_range_end"].(string))
		}
		pools = append(pools, zt.IpRange{
			Start: first.String(),
			End:   last.String(),
		})
	}

	n := &zt.Network{
		Id:          d.Id(),
		RulesSource: d.Get("rules_source").(string),
		Description: d.Get("description").(string),
		Config: &zt.NetworkConfig{
			Name:              d.Get("name").(string),
			Private:           d.Get("private").(bool),
			V4AssignMode:      zt.V4AssignModeConfig{ZT: true},
			Routes:            routes,
			IpAssignmentPools: pools,
		},
	}
	return n, nil
}
