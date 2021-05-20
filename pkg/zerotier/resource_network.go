package zerotier

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztcentral"
	"github.com/zerotier/go-ztcentral/pkg/spec"
)

func resourceNetwork() *schema.Resource {
	return &schema.Resource{
		Description:   "Network provider for ZeroTier, allows you to create ZeroTier networks.",
		CreateContext: resourceNetworkCreate,
		ReadContext:   resourceNetworkRead,
		UpdateContext: resourceNetworkRead, // schemawrap makes these equivalent
		DeleteContext: resourceNetworkDelete,
		Schema:        NewNetwork().TerraformSchema(),
	}
}

func resourceNetworkCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	ztn := NewNetwork()
	if err := ztn.CollectFromTerraform(d); err != nil {
		return err
	}

	c := m.(*ztcentral.Client)
	net := ztn.Yield().(*spec.Network)
	rules := net.RulesSource

	n, err := c.NewNetwork(ctx, *net.Config.Name, net)
	if err != nil {
		return []diag.Diagnostic{{
			Severity: diag.Error,
			Summary:  "Unable to create ZeroTier Network",
			Detail:   fmt.Sprintf("CreateNetwork returned error: %v", err),
		}}
	}

	if _, err := c.UpdateNetworkRules(ctx, *n.Id, *rules); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*n.Id)
	d.Set("tf_last_updated", time.Now().Unix())

	return resourceNetworkRead(ctx, d, m)
}

func resourceNetworkRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ztcentral.Client)
	var diags diag.Diagnostics

	ztNetworkID := d.Id()
	ztNetwork, err := c.GetNetwork(ctx, ztNetworkID)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to read ZeroTier Network",
			Detail:   fmt.Sprintf("GetNetwork returned error: %v", err),
		})
		return diags
	}

	return NewNetwork().CollectFromObject(d, ztNetwork, true)
}

func resourceNetworkUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ztcentral.Client)
	ztn := NewNetwork()

	if err := ztn.CollectFromTerraform(d); err != nil {
		return err
	}

	net := ztn.Yield().(*spec.Network)
	rules := net.RulesSource

	if _, err := c.UpdateNetworkRules(ctx, *net.Id, *rules); err != nil {
		return diag.FromErr(err)
	}

	updated, err := c.UpdateNetwork(ctx, *net.Id, net)
	if err != nil {
		return diag.FromErr(err)
	}

	updated.RulesSource = rules

	if err := ztn.CollectFromObject(d, updated, true); err != nil {
		return err
	}

	d.Set("tf_last_updated", time.Now().Unix())

	return nil
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
