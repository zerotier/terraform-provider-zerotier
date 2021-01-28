package zerotier

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	zt "github.com/someara/terraform-provider-zerotier/zerotier-client"
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
			Detail:   "CreateNetwork returned error",
		})
		return diag.FromErr(err)
	}

	d.SetId(n.Id)

	resourceNetworkRead(ctx, d, m)
	return diags
}

func resourceNetworkRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*zt.Client)
	var diags diag.Diagnostics

	zerotier_network_id := d.Id()
	zerotier_network, err := c.GetNetwork(zerotier_network_id)

	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to read ZeroTier Network",
			Detail:   "GetNetwork returned error",
		})
		return diag.FromErr(err)
	}

	d.SetId(zerotier_network_id)
	d.Set("name", zerotier_network.Config.Name)
	d.Set("description", zerotier_network.Description)

	return diags
}

func resourceNetworkUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*zt.Client)
	var diags diag.Diagnostics

	zerotier_network_id := d.Id()
	zerotier_network, err := c.GetNetwork(zerotier_network_id)

	if d.HasChange("description") {
		zerotier_network.Description = d.Get("description").(string)

		f, _ := os.Create("test.txt")
		f.WriteString(fmt.Sprintf("%v", zerotier_network))

		_, err = c.UpdateNetwork(zerotier_network_id, zerotier_network)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Unable to update ZeroTier Network description",
				Detail:   "UpdateNetwork returned error",
			})
			return diag.FromErr(err)
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

func parseIpAssignmentPools(m []interface{}) []zt.IpRange {
	var ip_range_list []zt.IpRange
	for _, ip_range := range m {
		r := ip_range.(map[string]interface{})
		ip_range_start := r["ip_range_start"].(string)
		ip_range_end := r["ip_range_end"].(string)

		ip_range_list = append(ip_range_list, zt.IpRange{
			Start: ip_range_start,
			End:   ip_range_end,
		})
	}
	return ip_range_list
}

func parseRoutes(data []interface{}) []zt.Route {
	var route_list []zt.Route
	for _, route := range data {
		r := route.(map[string]interface{})
		via := r["via"].(string)
		target := r["target"].(string)

		route_list = append(route_list, zt.Route{
			Target: target,
			Via:    via,
		})
	}
	return route_list
}
