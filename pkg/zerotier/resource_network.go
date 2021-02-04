package zerotier

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztcentral"
)

func resourceNetwork() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNetworkCreate,
		ReadContext:   resourceNetworkRead,
		UpdateContext: resourceNetworkUpdate,
		DeleteContext: resourceNetworkDelete,
		Schema: map[string]*schema.Schema{
			"last_updated": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Managed by Terraform",
			},
		},
	}
}

func resourceNetworkCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ztcentral.Client)
	var diags diag.Diagnostics

	n, err := c.NewNetwork(ctx, d.Get("name").(string))
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to create ZeroTier Network",
			Detail:   fmt.Sprintf("CreateNetwork returned error: %v", err),
		})
		return diags
	}

	fmt.Println(n.ID)

	d.SetId(n.ID)

	resourceNetworkRead(ctx, d, m)
	return diags
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

	d.Set("name", ztNetwork.Config.Name)
	d.Set("description", ztNetwork.Description)

	return diags
}

func resourceNetworkUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ztcentral.Client)
	var diags diag.Diagnostics

	ztNetworkID := d.Id()
	ztNetwork, err := c.GetNetwork(ctx, ztNetworkID)
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("description") {
		ztNetwork.Description = d.Get("description").(string)

		f, err := os.Create("test.txt")
		if err != nil {
			return diag.FromErr(err)
		}

		if _, err := f.WriteString(fmt.Sprintf("%v", ztNetwork)); err != nil {
			return diag.FromErr(err)
		}

		_, err = c.UpdateNetwork(ctx, ztNetwork)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Unable to update ZeroTier Network description",
				Detail:   fmt.Sprintf("UpdateNetwork returned error: %v", err),
			})
			return diags
		}
		d.Set("last_updated", time.Now().Format(time.RFC850))
	}

	// return diags
	return resourceNetworkRead(ctx, d, m)
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

//
// helpers
//

func parseV4AssignMode(m interface{}) map[string]interface{} {
	return m.(map[string]interface{})
}

func parseV6AssignMode(m interface{}) map[string]interface{} {
	return m.(map[string]interface{})
}

func parseIPAssignmentPools(m []interface{}) []ztcentral.IPRange {
	var ipRangeList []ztcentral.IPRange
	for _, ipRange := range m {
		r := ipRange.(map[string]interface{})
		ipRangeStart := r["ipRangeStart"].(string)
		ipRangeEnd := r["ipRangeEnd"].(string)

		ipRangeList = append(ipRangeList, ztcentral.IPRange{
			Start: ipRangeStart,
			End:   ipRangeEnd,
		})
	}
	return ipRangeList
}

func parseRoutes(data []interface{}) []ztcentral.Route {
	var routeList []ztcentral.Route
	for _, route := range data {
		r := route.(map[string]interface{})
		via := r["via"].(string)
		target := r["target"].(string)

		routeList = append(routeList, ztcentral.Route{
			Target: target,
			Via:    via,
		})
	}
	return routeList
}
