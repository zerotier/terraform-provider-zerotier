package zerotier

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztcentral"
)

// Provider -
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"zerotier_controller_url": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ZEROTIER_CONTROLLER_URL", ztcentral.BaseURLV1),
			},
			"zerotier_controller_token": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ZEROTIER_CONTROLLER_TOKEN", nil),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"zerotier_identity": resourceIdentity(),
			"zerotier_network":  resourceNetwork(),
			"zerotier_member":   resourceMember(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"zerotier_network": dataSourceNetwork(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	ztControllerURL := d.Get("zerotier_controller_url").(string)
	ztControllerToken := d.Get("zerotier_controller_token").(string)

	if ztControllerToken != "" {
		c := ztcentral.NewClient(ztControllerToken)
		if ztControllerURL != "" {
			c.BaseURL = ztControllerURL
		}

		return c, nil
	}

	return nil, diag.Errorf("zerotier_controller_token must be specified, or ZEROTIER_CONTROLLER_TOKEN must be specified in environment")
}
