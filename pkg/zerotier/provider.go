package zerotier

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/sirupsen/logrus"
	"github.com/zerotier/go-ztcentral"
)

// Provider -
func Provider() *schema.Provider {
	logrus.SetOutput(os.Stderr)
	level, err := logrus.ParseLevel(os.Getenv("TF_LOG"))
	if err != nil {
		level = logrus.InfoLevel
	}

	logrus.SetLevel(level)
	logrus.Infof("ZeroTier %s provider initialized", Version)

	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"zerotier_central_url": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ZEROTIER_CENTRAL_URL", ztcentral.BaseURLV1),
				Description: "ZeroTier Central API endpoint. Unlikely you'll need to alter this unless you're testing ZeroTier central itself.",
			},
			"zerotier_central_token": {
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
			"zerotier_token":    resourceToken(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"zerotier_network": dataSourceNetwork(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	defer logrus.Debug("ZeroTier provider configured")

	ztControllerToken := d.Get("zerotier_central_token").(string)

	if ztControllerToken != "" {
		c, err := ztcentral.NewClient(ztControllerToken)
		if err != nil {
			return nil, diag.FromErr(err)
		}

		logrus.Debug("Token configured successfully")

		c.SetUserAgent(fmt.Sprintf("terraform-provider-zerotier/%s", Version))

		return c, nil
	}

	return nil, diag.Errorf("zerotier_central_token must be specified, or ZEROTIER_CENTRAL_TOKEN must be specified in environment")
}
