package zerotier

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	zt "github.com/someara/terraform-provider-zerotier/pkg/zerotier-client"
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
	c := m.(*zt.Client)
	var diags diag.Diagnostics

	network := &zt.Network{
		Id:          d.Id(),
		Description: d.Get("description").(string),
		Config: zt.NetworkConfig{
			Name: d.Get("name").(string),
		},
	}

	n, err := c.CreateNetwork(network)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to create ZeroTier Network",
			Detail:   fmt.Sprintf("CreateNetwork returned error: %v", err),
		})
		return diags
	}

	d.SetId(n.Id)

	resourceNetworkRead(ctx, d, m)
	return diags
}

func resourceNetworkRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*zt.Client)
	var diags diag.Diagnostics

	ztNetworkID := d.Id()
	ztNetwork, err := c.GetNetwork(ztNetworkID)

	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to read ZeroTier Network",
			Detail:   fmt.Sprintf("GetNetwork returned error: %v", err),
		})
		return diags
	}

	d.SetId(ztNetworkID)
	d.Set("name", ztNetwork.Config.Name)
	d.Set("description", ztNetwork.Description)

	return diags
}

func resourceNetworkUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*zt.Client)
	var diags diag.Diagnostics

	ztNetworkID := d.Id()
	ztNetwork, err := c.GetNetwork(ztNetworkID)
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

		_, err = c.UpdateNetwork(ztNetworkID, ztNetwork)
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
	c := m.(*zt.Client)
	var diags diag.Diagnostics

	networkID := d.Id()

	err := c.DeleteNetwork(networkID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return diags
}

//
// helpers
//

func parseV4AssignMode(m interface{}) zt.V4AssignMode {
	mode := m.(map[string]interface{})
	return zt.V4AssignMode{
		ZT: mode["zt"].(bool),
	}
}

func parseV6AssignMode(m interface{}) zt.V6AssignMode {
	mode := m.(map[string]interface{})
	return zt.V6AssignMode{
		ZT:       mode["zt"].(bool),
		SixPlane: mode["six_plane"].(bool),
		Rfc4193:  mode["rfc_4193"].(bool),
	}
}

func parseIPAssignmentPools(m []interface{}) []zt.IpRange {
	var ipRangeList []zt.IpRange
	for _, ipRange := range m {
		r := ipRange.(map[string]interface{})
		ipRangeStart := r["ipRangeStart"].(string)
		ipRangeEnd := r["ipRangeEnd"].(string)

		ipRangeList = append(ipRangeList, zt.IpRange{
			Start: ipRangeStart,
			End:   ipRangeEnd,
		})
	}
	return ipRangeList
}

func parseRoutes(data []interface{}) []zt.Route {
	var routeList []zt.Route
	for _, route := range data {
		r := route.(map[string]interface{})
		via := r["via"].(string)
		target := r["target"].(string)

		routeList = append(routeList, zt.Route{
			Target: target,
			Via:    via,
		})
	}
	return routeList
}
