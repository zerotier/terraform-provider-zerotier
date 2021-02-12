package zerotier

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztcentral"
)

func ztNetworkYield(vs ValidatedSchema) interface{} {
	return &ztcentral.NetworkConfig{
		Name:             vs.Get("name").(string),
		IPAssignmentPool: vs.Get("assignment_pool").([]ztcentral.IPRange),
		Routes:           vs.Get("route").([]ztcentral.Route),
		IPV4AssignMode:   vs.Get("assign_ipv4").(ztcentral.IPV4AssignMode),
		IPV6AssignMode:   vs.Get("assign_ipv6").(ztcentral.IPV6AssignMode),
		EnableBroadcast:  vs.Get("enable_broadcast").(bool),
		MTU:              vs.Get("mtu").(int),
		MulticastLimit:   vs.Get("multicast_limit").(int),
		Private:          vs.Get("private").(bool),
	}
}

func ztNetworkCollect(vs ValidatedSchema, d *schema.ResourceData, i interface{}) diag.Diagnostics {
	ztNetwork := i.(*ztcentral.NetworkConfig)

	var diags diag.Diagnostics

	diags = append(diags, vs.Set(d, "name", ztNetwork.Name)...)
	diags = append(diags, vs.Set(d, "mtu", ztNetwork.MTU)...)
	diags = append(diags, vs.Set(d, "creation_time", ztNetwork.CreationTime)...)
	diags = append(diags, vs.Set(d, "route", ztNetwork.Routes)...)
	diags = append(diags, vs.Set(d, "assignment_pool", ztNetwork.IPAssignmentPool)...)
	diags = append(diags, vs.Set(d, "enable_broadcast", ztNetwork.EnableBroadcast)...)
	diags = append(diags, vs.Set(d, "multicast_limit", ztNetwork.MulticastLimit)...)
	diags = append(diags, vs.Set(d, "private", ztNetwork.Private)...)
	diags = append(diags, vs.Set(d, "assign_ipv4", ztNetwork.IPV4AssignMode)...)
	diags = append(diags, vs.Set(d, "assign_ipv6", ztNetwork.IPV6AssignMode)...)

	return diags
}

// ZTNetwork is our internal validated schema. See schemawrap.go.
var ZTNetwork = ValidatedSchema{
	YieldFunc:   ztNetworkYield,
	CollectFunc: ztNetworkCollect,
	Schema: map[string]*SchemaWrap{
		"creation_time": {
			Schema: &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
		"tf_last_updated": {
			Schema: &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
		"name": {
			Schema: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			ValidatorFunc: strNonEmpty,
		},
		"description": { // FIXME this is currently not working
			Schema: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Managed by Terraform",
			},
		},
		"enable_broadcast": {
			Schema: &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
		},
		"mtu": {
			Schema: &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  2800,
			},
		},
		"multicast_limit": {
			Schema: &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  32,
			},
		},
		"private": {
			Schema: &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
		"route": {
			FromTerraformFunc: mkRoutes,
			ToTerraformFunc:   mktfRoutes,
			Schema: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"via": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"target": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
		"assign_ipv4": {
			FromTerraformFunc: mkipv4assign,
			ToTerraformFunc:   mktfipv4assign,
			Schema: &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
					ForceNew: true,
					Default:  true,
				},
			},
		},
		"assign_ipv6": {
			FromTerraformFunc: mkipv6assign,
			ToTerraformFunc:   mktfipv6assign,
			Schema: &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
					ForceNew: true,
				},
			},
		},
		"assignment_pool": {
			FromTerraformFunc: mkIPRange,
			ToTerraformFunc:   mktfRanges,
			Schema: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"start": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"end": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"cidr": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	},
}
