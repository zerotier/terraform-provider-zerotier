package zerotier

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztcentral"
)

func ztNetworkYield(vs ValidatedSchema) interface{} {
	return &ztcentral.Network{
		ID:          vs.Get("id").(string),
		RulesSource: vs.Get("flow_rules").(string),
		Config: ztcentral.NetworkConfig{
			Name:             vs.Get("name").(string),
			IPAssignmentPool: vs.Get("assignment_pool").([]ztcentral.IPRange),
			Routes:           vs.Get("route").([]ztcentral.Route),
			IPV4AssignMode:   vs.Get("assign_ipv4").(*ztcentral.IPV4AssignMode),
			IPV6AssignMode:   vs.Get("assign_ipv6").(*ztcentral.IPV6AssignMode),
			EnableBroadcast:  boolPtr(vs.Get("enable_broadcast").(bool)),
			MTU:              vs.Get("mtu").(int),
			MulticastLimit:   vs.Get("multicast_limit").(int),
			Private:          boolPtr(vs.Get("private").(bool)),
		},
	}
}

func ztNetworkCollect(vs ValidatedSchema, d *schema.ResourceData, i interface{}) diag.Diagnostics {
	ztNetwork := i.(*ztcentral.Network)

	var diags diag.Diagnostics

	diags = append(diags, vs.Set(d, "id", ztNetwork.ID)...)
	diags = append(diags, vs.Set(d, "flow_rules", ztNetwork.RulesSource)...)
	diags = append(diags, vs.Set(d, "name", ztNetwork.Config.Name)...)
	diags = append(diags, vs.Set(d, "mtu", ztNetwork.Config.MTU)...)
	diags = append(diags, vs.Set(d, "creation_time", ztNetwork.Config.CreationTime)...)
	diags = append(diags, vs.Set(d, "route", ztNetwork.Config.Routes)...)
	diags = append(diags, vs.Set(d, "assignment_pool", ztNetwork.Config.IPAssignmentPool)...)
	diags = append(diags, vs.Set(d, "enable_broadcast", ptrBool(ztNetwork.Config.EnableBroadcast))...)
	diags = append(diags, vs.Set(d, "multicast_limit", ztNetwork.Config.MulticastLimit)...)
	diags = append(diags, vs.Set(d, "private", ptrBool(ztNetwork.Config.Private))...)
	diags = append(diags, vs.Set(d, "assign_ipv4", ztNetwork.Config.IPV4AssignMode)...)
	diags = append(diags, vs.Set(d, "assign_ipv6", ztNetwork.Config.IPV6AssignMode)...)

	return diags
}

// ZTNetwork is our internal validated schema. See schemawrap.go.
var ZTNetwork = ValidatedSchema{
	YieldFunc:   ztNetworkYield,
	CollectFunc: ztNetworkCollect,
	Schema: map[string]*SchemaWrap{
		"id": {
			Schema: &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
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
		"flow_rules": {
			Schema: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "accept;",
			},
		},
	},
}

// ZTNetworkDS is our internal validated schema for data sources. See schemawrap.go.
var ZTNetworkDS = ValidatedSchema{
	YieldFunc:   ztNetworkYield,
	CollectFunc: ztNetworkCollect,
	Schema: map[string]*SchemaWrap{
		"creation_time": {
			Schema: &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
		"id": {
			Schema: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
		"name": {
			Schema: &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
		"description": { // FIXME this is currently not working
			Schema: &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
		"enable_broadcast": {
			Schema: &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
		"mtu": {
			Schema: &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
		"multicast_limit": {
			Schema: &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
		"private": {
			Schema: &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
		"route": {
			FromTerraformFunc: mkRoutes,
			ToTerraformFunc:   mktfRoutes,
			Schema: &schema.Schema{
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"via": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"target": {
							Type:     schema.TypeString,
							Computed: true,
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
				Computed: true,
				Elem: &schema.Schema{
					Type:     schema.TypeBool,
					Computed: true,
				},
			},
		},
		"assign_ipv6": {
			FromTerraformFunc: mkipv6assign,
			ToTerraformFunc:   mktfipv6assign,
			Schema: &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
				Elem: &schema.Schema{
					Type:     schema.TypeBool,
					Computed: true,
				},
			},
		},
		"assignment_pool": {
			FromTerraformFunc: mkIPRange,
			ToTerraformFunc:   mktfRanges,
			Schema: &schema.Schema{
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"start": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"end": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"cidr": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
		"flow_rules": {
			Schema: &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	},
}
