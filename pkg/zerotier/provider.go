package zerotier

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/sirupsen/logrus"
	"github.com/zerotier/go-ztcentral"
	"github.com/zerotier/terraform-provider-zerotier/pkg/providerschema"
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

	// The provider schema is shared with the Central V2 half of the provider
	// (pkg/zerotierv2) and must stay identical to it; see providerschema.
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			providerschema.CentralURL: {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ZEROTIER_CENTRAL_URL", ztcentral.BaseURLV1),
				Description: providerschema.CentralURLDescription,
			},
			providerschema.CentralToken: {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ZEROTIER_CENTRAL_TOKEN", nil),
				Description: providerschema.CentralTokenDescription,
			},
			providerschema.CentralV2URL: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: providerschema.CentralV2URLDescription,
			},
			providerschema.CentralV2APIKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: providerschema.CentralV2APIKeyDescription,
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
			"zerotier_members": dataSourceMembers(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	defer logrus.Debug("ZeroTier provider configured")

	ztControllerToken := d.Get(providerschema.CentralToken).(string)
	ztControllerURL := d.Get(providerschema.CentralURL).(string)

	// The token is optional so the provider can be used for Central V2 alone.
	// V1 resources report a missing token when they are used (see v1Client).
	if ztControllerToken == "" {
		return nil, nil
	}

	// NOTE this whole block is extremely order-dependent
	//      deal with it
	if ztControllerURL != "" {
		ztcentral.BaseURLV1 = ztControllerURL
	}

	c, err := ztcentral.NewClient(ztControllerToken)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	logrus.Debug("Token configured successfully")

	c.SetUserAgent(fmt.Sprintf("terraform-provider-zerotier/%s", Version))

	return c, nil
}

// v1Client returns the Central V1 client, or an error if no V1 token was
// configured.
func v1Client(m interface{}) (*ztcentral.Client, error) {
	c, ok := m.(*ztcentral.Client)
	if !ok || c == nil {
		return nil, errors.New("zerotier_central_token must be specified, or ZEROTIER_CENTRAL_TOKEN must be specified in environment")
	}

	return c, nil
}
