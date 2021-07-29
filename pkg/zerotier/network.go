package zerotier

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztcentral/pkg/spec"
)

// NetworkSchema is our terraform network resource's schema.
var NetworkSchema = map[string]*schema.Schema{
	"id": {
		Type:        schema.TypeString,
		Optional:    true,
		Computed:    true,
		Description: "ZeroTier's internal network identifier, aka NetworkID",
	},
	"creation_time": {
		Type:        schema.TypeInt,
		Computed:    true,
		Description: "The time at which this network was created, in epoch seconds",
	},
	"name": {
		Type:        schema.TypeString,
		Optional:    true,
		Computed:    true,
		Description: "The name of the network",
	},
	"description": {
		Type:        schema.TypeString,
		Optional:    true,
		Default:     "Managed by Terraform",
		Description: "The description of the network",
	},
	"enable_broadcast": {
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     true,
		Description: "Enable broadcast packets on the network",
	},
	"multicast_limit": {
		Type:        schema.TypeInt,
		Optional:    true,
		Default:     32,
		Description: "Maximum number of recipients per multicast or broadcast. Warning - Setting this to 0 will disable IPv4 communication on your network!",
	},
	"private": {
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     true,
		Description: "Whether or not the network is private.  If false, members will *NOT* need to be authorized to join.",
	},
	"route": {
		Type:     schema.TypeSet,
		Optional: true,
		Computed: true,
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
			Description: "A ipv4 or ipv6 network route",
		},
	},
	"assign_ipv4": {
		Type:     schema.TypeSet,
		Optional: true,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"zerotier": {
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     true,
					Description: "Use zerotier ipv4 addressing",
				},
			},
		},
		Description: "IPv4 Assignment RuleSets",
	},
	"assign_ipv6": {
		Type:     schema.TypeSet,
		Optional: true,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"zerotier": {
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
					Description: "Use zerotier ipv6 manual addressing",
				},
				"sixplane": {
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
					Description: "6PLANE addressing method",
				},
				"rfc4193": {
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
					Description: "RFC4193 addressing method",
				},
			},
		},
		Description: "IPv6 Assignment RuleSets",
	},
	"assignment_pool": {
		Type:     schema.TypeSet,
		Optional: true,
		Computed: true,
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
			},
			Description: "Rules regarding IPv4 and IPv6 assignments",
		},
	},
	"flow_rules": {
		Type:        schema.TypeString,
		Optional:    true,
		Default:     "accept;",
		Description: "The layer 2 flow rules to apply to packets traveling across this network. Please see https://www.zerotier.com/manual/#3_4_1 for more information.",
	},
}

func toNetwork(d *schema.ResourceData) (*spec.Network, diag.Diagnostics) {
	assignmentPools, err := mkIPRange(d.Get("assignment_pool"))
	if err != nil {
		return nil, err
	}

	routes, err := mkRoutes(d.Get("route"))
	if err != nil {
		return nil, err
	}

	v4assign, err := mkipv4assign(d.Get("assign_ipv4"))
	if err != nil {
		return nil, err
	}

	v6assign, err := mkipv6assign(d.Get("assign_ipv6"))
	if err != nil {
		return nil, err
	}

	network := &spec.Network{
		Id:          stringPtr(d.Get("id").(string)),
		RulesSource: stringPtr(d.Get("flow_rules").(string)),
		Description: stringPtr(d.Get("description").(string)),
		Config: &spec.NetworkConfig{
			Name:              stringPtr(d.Get("name").(string)),
			IpAssignmentPools: (assignmentPools).(*[]spec.IPRange),
			Routes:            (routes).(*[]spec.Route),
			V4AssignMode:      (v4assign).(*spec.IPV4AssignMode),
			V6AssignMode:      (v6assign).(*spec.IPV6AssignMode),
			EnableBroadcast:   boolPtr(d.Get("enable_broadcast").(bool)),
			MulticastLimit:    intPtr(d.Get("multicast_limit").(int)),
			Private:           boolPtr(d.Get("private").(bool)),
		},
	}

	return network, nil
}

func networkToTerraform(d *schema.ResourceData, n *spec.Network) diag.Diagnostics {
	d.SetId(*n.Id)
	d.Set("flow_rules", *n.RulesSource)
	d.Set("description", *n.Description)
	d.Set("name", n.Config.Name)
	d.Set("creation_time", *n.Config.CreationTime)
	d.Set("route", mktfRoutes(n.Config.Routes))
	d.Set("assignment_pool", mktfRanges(n.Config.IpAssignmentPools))
	d.Set("enable_broadcast", ptrBool(n.Config.EnableBroadcast))
	d.Set("multicast_limit", n.Config.MulticastLimit)
	d.Set("private", ptrBool(n.Config.Private))
	d.Set("assign_ipv4", mktfipv4assign(n.Config.V4AssignMode))
	d.Set("assign_ipv6", mktfipv6assign(n.Config.V6AssignMode))

	return nil
}
