package zerotier

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztcentral/pkg/spec"
)

func ztNetworkYield(vs ValidatedSchema) interface{} {
	assignmentPools := vs.Get("assignment_pool").(*[]spec.IPRange)
	routes := vs.Get("route").(*[]spec.Route)

	return &spec.Network{
		Id:          stringPtr(vs.Get("id").(string)),
		RulesSource: stringPtr(vs.Get("flow_rules").(string)),
		Config: &spec.NetworkConfig{
			Name:              stringPtr(vs.Get("name").(string)),
			IpAssignmentPools: assignmentPools,
			Routes:            routes,
			V4AssignMode:      vs.Get("assign_ipv4").(*spec.IPV4AssignMode),
			V6AssignMode:      vs.Get("assign_ipv6").(*spec.IPV6AssignMode),
			EnableBroadcast:   boolPtr(vs.Get("enable_broadcast").(bool)),
			Mtu:               intPtr(vs.Get("mtu").(int)),
			MulticastLimit:    intPtr(vs.Get("multicast_limit").(int)),
			Private:           boolPtr(vs.Get("private").(bool)),
		},
	}
}

func ztNetworkCollect(vs ValidatedSchema, d *schema.ResourceData, i interface{}) diag.Diagnostics {
	ztNetwork := i.(spec.Network)

	var diags diag.Diagnostics

	diags = append(diags, vs.Set(d, "id", ztNetwork.Id)...)
	diags = append(diags, vs.Set(d, "flow_rules", ztNetwork.RulesSource)...)
	diags = append(diags, vs.Set(d, "name", ztNetwork.Config.Name)...)
	diags = append(diags, vs.Set(d, "mtu", ztNetwork.Config.Mtu)...)
	diags = append(diags, vs.Set(d, "creation_time", ztNetwork.Config.CreationTime)...)
	diags = append(diags, vs.Set(d, "route", ztNetwork.Config.Routes)...)
	diags = append(diags, vs.Set(d, "assignment_pool", ztNetwork.Config.IpAssignmentPools)...)
	diags = append(diags, vs.Set(d, "enable_broadcast", ptrBool(ztNetwork.Config.EnableBroadcast))...)
	diags = append(diags, vs.Set(d, "multicast_limit", ztNetwork.Config.MulticastLimit)...)
	diags = append(diags, vs.Set(d, "private", ptrBool(ztNetwork.Config.Private))...)
	diags = append(diags, vs.Set(d, "assign_ipv4", ztNetwork.Config.V4AssignMode)...)
	diags = append(diags, vs.Set(d, "assign_ipv6", ztNetwork.Config.V6AssignMode)...)

	return diags
}

// ZTNetwork is our internal validated schema. See schemawrap.go.
var ZTNetwork = ValidatedSchema{
	YieldFunc:   ztNetworkYield,
	CollectFunc: ztNetworkCollect,
	Schema: map[string]*SchemaWrap{
		"id": {
			Schema: &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ZeroTier's internal network identifier, aka NetworkID",
			},
		},
		"creation_time": {
			Schema: &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The time at which this network was created, in epoch seconds",
			},
		},
		"tf_last_updated": {
			Schema: &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The time at which this terraform was last updated, in epoch seconds",
			},
		},
		"name": {
			Schema: &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the network",
			},
			ValidatorFunc: strNonEmpty,
		},
		"description": { // FIXME this is currently not working
			Schema: &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "Managed by Terraform",
				Description: "The description of the network",
			},
		},
		"enable_broadcast": {
			Schema: &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Enable broadcast packets on the network",
			},
		},
		"mtu": {
			Schema: &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "MTU to set on the client virtual network adapter",
			},
		},
		"multicast_limit": {
			Schema: &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     32,
				Description: "Maximum number of recipients per multicast or broadcast. Warning - Setting this to 0 will disable IPv4 communication on your network!",
			},
		},
		"private": {
			Schema: &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether or not the network is private.  If false, members will *NOT* need to be authorized to join.",
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
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Gateway address",
						},
						"target": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Network to route for",
						},
					},
				},
				Description: "A ipv4 or ipv6 network route",
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
					Type:        schema.TypeBool,
					Optional:    true,
					ForceNew:    true,
					Default:     true,
					Description: "Allowed map keys: `zerotier`, which must be true to gain ipv4 addressing automatically",
				},
				Description: "IPv4 Assignment RuleSets",
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
					Description: `
					Allowed map keys:
					- zerotier: standard ZeroTier ipv6 1:1 addressing
					- sixplane: 6PLANE assigns every host on a ZeroTier virtual network an IPv6 address within a private /40 network. More information: https://zerotier.atlassian.net/wiki/spaces/SD/pages/7274520/Using+NDP+Emulated+6PLANE+Addressing+With+Docker
					- rfc4193: RFC 4193 support. https://tools.ietf.org/html/rfc4193
					`,
				},
				Description: "IPv6 Assignment RuleSets",
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
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The first address in the assignment rule. This must be the lowest number in the pool. `start` must also be accompanied by `end`.",
						},
						"end": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The last address in the assignment rule. This must be the highest number in the pool. end must also be accompanied by start.",
						},
						"cidr": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "An address range in CIDR notation. This must have no other keys assigned to this block as CIDR denotes the start and end address automatically",
						},
					},
				},
				Description: "Rules regarding IPv4 and IPv6 assignments",
			},
		},
		"flow_rules": {
			Schema: &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "accept;",
				Description: "The layer 2 flow rules to apply to packets traveling across this network. Please see https://www.zerotier.com/manual/#3_4_1 for more information.",
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
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The time at which this network was created, in epoch seconds",
			},
		},
		"id": {
			Schema: &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "ZeroTier's internal network identifier, aka NetworkID",
			},
		},
		"name": {
			Schema: &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the network",
			},
		},
		"description": { // FIXME this is currently not working
			Schema: &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The description of the network",
			},
		},
		"enable_broadcast": {
			Schema: &schema.Schema{
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Enable broadcast packets on the network",
			},
		},
		"mtu": {
			Schema: &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "MTU to set on the client virtual network adapter",
			},
		},
		"multicast_limit": {
			Schema: &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Maximum number of recipients per multicast or broadcast. Warning - Setting this to 0 will disable IPv4 communication on your network!",
			},
		},
		"private": {
			Schema: &schema.Schema{
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether or not the network is private.  If false, members will *NOT* need to be authorized to join.",
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
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Gateway address",
						},
						"target": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Network to route for",
						},
					},
				},
				Description: "A ipv4 or ipv6 network route",
			},
		},
		"assign_ipv4": {
			FromTerraformFunc: mkipv4assign,
			ToTerraformFunc:   mktfipv4assign,
			Schema: &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
				Elem: &schema.Schema{
					Type:        schema.TypeBool,
					Computed:    true,
					Description: "Allowed map keys: `zerotier`, which must be true to gain ipv4 addressing automatically",
				},
				Description: "IPv4 Assignment RuleSets",
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
					Description: `
					Allowed map keys:
					- zerotier: standard ZeroTier ipv6 1:1 addressing
					- sixplane: 6PLANE assigns every host on a ZeroTier virtual network an IPv6 address within a private /40 network. More information: https://zerotier.atlassian.net/wiki/spaces/SD/pages/7274520/Using+NDP+Emulated+6PLANE+Addressing+With+Docker
					- rfc4193: RFC 4193 support. https://tools.ietf.org/html/rfc4193
					`,
				},
				Description: "IPv6 Assignment RuleSets",
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
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The first address in the assignment rule. This must be the lowest number in the pool. `start` must also be accompanied by `end`.",
						},
						"end": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The last address in the assignment rule. This must be the highest number in the pool. end must also be accompanied by start.",
						},
						"cidr": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "An address range in CIDR notation. This must have no other keys assigned to this block as CIDR denotes the start and end address automatically",
						},
					},
				},
				Description: "Rules regarding IPv4 and IPv6 assignments",
			},
		},
		"flow_rules": {
			Schema: &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The layer 2 flow rules to apply to packets traveling across this network. Please see https://www.zerotier.com/manual/#3_4_1 for more information.",
			},
		},
	},
}
