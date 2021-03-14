package zerotier

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztcentral"
)

// Provider -
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"zerotier_central__url": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ZEROTIER_CENTRAL_URL", ztcentral.BaseURLV1),
				Description: "ZeroTier Central API endpoint. Unlikely you'll need to alter this unless you're testing ZeroTier central itself.",
			},
			"zerotier_central__token": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ZEROTIER_CENTRAL_TOKEN", nil),
				Description: "ZeroTier Central API Token; you can generate a new one at https://my.zerotier.com/account.",
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
	ztControllerURL := d.Get("zerotier_central__url").(string)
	ztControllerToken := d.Get("zerotier_central__token").(string)

	if ztControllerToken != "" {
		c := ztcentral.NewClient(ztControllerToken)
		if ztControllerURL != "" {
			c.BaseURL = ztControllerURL
		}

		c.SetUserAgent(fmt.Sprintf("terraform-provider-zerotier/%s", Version))

		return c, nil
	}

	return nil, diag.Errorf("zerotier_central__token must be specified, or ZEROTIER_CENTRAL_TOKEN must be specified in environment")
}
