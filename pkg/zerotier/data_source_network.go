package zerotier

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztcentral"
)

func dataSourceNetwork() *schema.Resource {
	return &schema.Resource{
		Description: "Data source for ZeroTier networks, allowing you to find a network by ID",
		ReadContext: dataSourceNetworkRead,
		Schema:      NetworkSchema,
	}
}

func dataSourceNetworkRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ztcentral.Client)
	var diags diag.Diagnostics

	ztNetworkID := d.Get("id").(string)

	ztNetwork, err := c.GetNetwork(ctx, ztNetworkID)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to read ZeroTier Network",
			Detail:   fmt.Sprintf("GetNetwork returned error: %v", err),
		})
		return diags
	}

	return networkToTerraform(d, ztNetwork)
}
