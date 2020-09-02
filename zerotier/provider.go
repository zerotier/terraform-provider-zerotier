package zerotier

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	zt "github.com/someara/terraform-provider-zerotier/zerotier-client"
)

// Provider -
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"zerotier_controller_url": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ZEROTIER_CONTROLLER_URL", nil),
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
	zerotier_controller_url := d.Get("zerotier_controller_url").(string)
	zerotier_controller_token := d.Get("zerotier_controller_token").(string)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	if (zerotier_controller_url != "") && (zerotier_controller_token != "") {
		c, err := zt.NewClient(&zerotier_controller_url, &zerotier_controller_token)
		if err != nil {
			return nil, diag.FromErr(err)
		}

		return c, diags
	}

	c, err := zt.NewClient(nil, nil)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return c, diags
}
