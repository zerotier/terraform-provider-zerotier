package zerotier

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztcentral"
)

func resourceNetwork() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNetworkCreate,
		ReadContext:   resourceNetworkRead,
		UpdateContext: resourceNetworkRead, // schemawrap makes these equivalent
		DeleteContext: resourceNetworkDelete,
		Schema:        ZTNetwork.TerraformSchema(),
	}
}

func resourceNetworkCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if err := ZTNetwork.CollectFromTerraform(d); err != nil {
		return err
	}

	c := m.(*ztcentral.Client)
	net := ZTNetwork.Yield().(*ztcentral.Network)

	n, err := c.NewNetwork(ctx, net.Config.Name, net)
	if err != nil {
		return []diag.Diagnostic{{
			Severity: diag.Error,
			Summary:  "Unable to create ZeroTier Network",
			Detail:   fmt.Sprintf("CreateNetwork returned error: %v", err),
		}}
	}

	d.SetId(n.ID)
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

	return ZTNetwork.CollectFromObject(d, ztNetwork)
}

func resourceNetworkUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if err := resourceNetworkRead(ctx, d, m); err != nil {
		return err
	}

	c := m.(*ztcentral.Client)
	ZTNetwork.CollectFromTerraform(d)

	updated, err := c.UpdateNetwork(ctx, ZTNetwork.Yield().(*ztcentral.Network))
	if err != nil {
		return diag.FromErr(err)
	}

	ZTNetwork.CollectFromObject(d, updated)

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
