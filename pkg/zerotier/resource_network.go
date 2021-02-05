package zerotier

import (
	"context"
	"errors"
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
			"routes": &schema.Schema{
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeMap,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				Optional: true,
			},
			"assignment_pool": &schema.Schema{
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeMap,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				Optional: true,
			},
		},
	}
}

func mkIPRange(ranges interface{}) ([]ztcentral.IPRange, error) {
	if ranges == nil {
		return []ztcentral.IPRange{}, nil
	}

	ret := []ztcentral.IPRange{}

	for _, r := range ranges.([]interface{}) {
		var start, end string
		m := r.(map[string]interface{})
		if s, ok := m["start"]; ok {
			start = s.(string)
		} else {
			return ret, errors.New("start does not exist")
		}

		if e, ok := m["end"]; ok {
			end = e.(string)
		} else {
			return ret, errors.New("end does not exist")
		}

		ret = append(ret, ztcentral.IPRange{
			Start: start,
			End:   end,
		})
	}

	return ret, nil
}

func mkRoutes(routes interface{}) ([]ztcentral.Route, error) {
	if routes == nil {
		return []ztcentral.Route{}, nil
	}

	ret := []ztcentral.Route{}

	for _, route := range routes.([]interface{}) {
		var target, via string
		m := route.(map[string]interface{})
		if t, ok := m["target"]; ok {
			target = t.(string)
		} else {
			return ret, errors.New("target does not exist")
		}

		if v, ok := m["via"]; ok {
			via = v.(string)
		} else {
			return ret, errors.New("target does not exist")
		}

		ret = append(ret, ztcentral.Route{
			Target: target,
			Via:    via,
		})
	}

	return ret, nil
}

func mktfRoutes(routes []ztcentral.Route) interface{} {
	ret := []map[string]interface{}{}

	for _, route := range routes {
		ret = append(ret, map[string]interface{}{
			"target": route.Target,
			"via":    route.Via,
		})
	}

	return ret
}

func mktfRanges(ranges []ztcentral.IPRange) interface{} {
	ret := []map[string]interface{}{}

	for _, r := range ranges {
		ret = append(ret, map[string]interface{}{
			"start": r.Start,
			"via":   r.End,
		})
	}

	return ret
}

func resourceNetworkCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ztcentral.Client)
	var diags diag.Diagnostics

	routes, err := mkRoutes(d.Get("routes"))
	if err != nil {
		return diag.FromErr(err)
	}

	ipranges, err := mkIPRange(d.Get("assignment_pool"))
	if err != nil {
		return diag.FromErr(err)
	}

	n, err := c.NewNetwork(ctx, d.Get("name").(string), &ztcentral.NetworkConfig{
		IPAssignmentPool: ipranges,
		Routes:           routes,
	})

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
	d.Set("routes", mktfRoutes(ztNetwork.Config.Routes))
	d.Set("assignment_pool", mktfRanges(ztNetwork.Config.IPAssignmentPool))

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

	var changed bool

	if d.HasChange("description") {
		changed = true
		ztNetwork.Description = d.Get("description").(string)
	}

	if d.HasChange("routes") {
		changed = true
		var err error
		ztNetwork.Config.Routes, err = mkRoutes(d.Get("routes"))
		if err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if d.HasChange("assignment_pool") {
		changed = true
		var err error
		ztNetwork.Config.IPAssignmentPool, err = mkIPRange(d.Get("assignment_pool"))
		if err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	}

	if changed {
		if _, err := c.UpdateNetwork(ctx, ztNetwork); err != nil {
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
