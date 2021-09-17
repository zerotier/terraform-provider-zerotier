package zerotier

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztcentral"
)

func resourceNetwork() *schema.Resource {
	return &schema.Resource{
		Description:   "Network provider for ZeroTier, allows you to create ZeroTier networks.",
		CreateContext: resourceNetworkCreate,
		ReadContext:   resourceNetworkRead,
		UpdateContext: resourceNetworkUpdate,
		DeleteContext: resourceNetworkDelete,
		Schema:        NetworkSchema,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func resourceNetworkCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ztcentral.Client)
	net, derr := toNetwork(d)
	if derr != nil {
		return derr
	}

	n, err := c.NewNetwork(ctx, *net.Config.Name, net)
	if err != nil {
		return []diag.Diagnostic{{
			Severity: diag.Error,
			Summary:  "Unable to create ZeroTier Network",
			Detail:   fmt.Sprintf("CreateNetwork returned error: %v", err),
		}}
	}

	rs, err := c.UpdateNetworkRules(ctx, *n.Id, d.Get("flow_rules").(string))
	if err != nil {
		return []diag.Diagnostic{{
			Severity: diag.Error,
			Summary:  "Unable to update ZeroTier Network flow rules",
			Detail:   fmt.Sprintf("CreateNetwork returned error: %v", err),
		}}
	}
	n.RulesSource = &rs

	return networkToTerraform(d, n)
}

func resourceNetworkRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ztcentral.Client)
	var diags diag.Diagnostics

	ztNetwork, err := c.GetNetwork(ctx, d.Get("id").(string))
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

func resourceNetworkUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ztcentral.Client)
	net, derr := toNetwork(d)
	if derr != nil {
		return derr
	}

	updated, err := c.UpdateNetwork(ctx, *net.Id, net)
	if err != nil {
		return diag.FromErr(err)
	}

	rs, err := c.UpdateNetworkRules(ctx, *net.Id, d.Get("flow_rules").(string))
	if err != nil {
		return []diag.Diagnostic{{
			Severity: diag.Error,
			Summary:  "Unable to update ZeroTier Network flow rules",
			Detail:   fmt.Sprintf("CreateNetwork returned error: %v", err),
		}}
	}

	updated.RulesSource = &rs

	return networkToTerraform(d, updated)
}

func resourceNetworkDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ztcentral.Client)
	var diags diag.Diagnostics

	networkID := d.Id()

	err := c.DeleteNetwork(ctx, networkID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return diags
}
