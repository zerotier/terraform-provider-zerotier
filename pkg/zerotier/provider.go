package zerotier

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztcentral"
)

// HostURL is the URL of the standard ZeroTier client API endpoint.
const HostURL = "https://my.zerotier.com/api"

// Provider -
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"zerotier_controller_url": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ZEROTIER_CONTROLLER_URL", HostURL),
			},
			"zerotier_controller_token": &schema.Schema{
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
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	ztControllerURL := d.Get("zerotier_controller_url").(string)
	ztControllerToken := d.Get("zerotier_controller_token").(string)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	if (ztControllerURL != "") && (ztControllerToken != "") {
		c := ztcentral.NewClient(ztControllerToken)
		return c, diags
	}

	return nil, diag.Errorf("zerotier_controller_token must be specified, or ZEROTIER_CONTROLLER_TOKEN must be specified in environment")
}
