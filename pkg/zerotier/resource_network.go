package zerotier

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net"
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
			"last_updated": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Managed by Terraform",
			},
			"route": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"via": {
							Type:     schema.TypeString,
							Required: true,
						},
						"target": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"assignment_pool": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"start": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"end": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

// FIXME keep this. we'll use it later.
func mkIPRangeFromCIDR(cidr interface{}) (ztcentral.IPRange, error) {
	iprange := ztcentral.IPRange{}

	first, nw, err := net.ParseCIDR(cidr.(string))
	if err != nil {
		return iprange, err
	}

	var last net.IP

	prefixLen, bits := nw.Mask.Size()

	if prefixLen == bits {
		last = first
	} else {
		val := big.NewInt(0)
		val.SetBytes(first)
		lastVal := big.NewInt(1)
		lastVal.Lsh(lastVal, uint(bits-prefixLen))
		lastVal.Sub(lastVal, big.NewInt(1))
		lastVal.Or(lastVal, val)

		last = net.IP(make([]byte, len(first)))
		b := lastVal.Bytes()
		for i := 1; i <= len(b); i++ {
			last[len(last)-i] = b[len(b)-i]
		}

		first = net.IP(make([]byte, len(first)))
		b = val.Bytes()
		for i := 1; i <= len(b); i++ {
			first[len(first)-i] = b[len(b)-i]
		}
	}

	iprange = ztcentral.IPRange{
		Start: first.String(),
		End:   last.String(),
	}

	return iprange, nil
}

func mkIPRange(ranges interface{}) ([]ztcentral.IPRange, error) {
	ret := []ztcentral.IPRange{}

	for _, r := range ranges.(*schema.Set).List() {
		m := r.(map[string]interface{})
		var start, end string
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
	ret := []ztcentral.Route{}

	for _, r := range routes.(*schema.Set).List() {
		m := r.(map[string]interface{})
		var target, via string
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
			"end":   r.End,
		})
	}

	return ret
}

func resourceNetworkCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ztcentral.Client)
	var diags diag.Diagnostics

	routes, err := mkRoutes(d.Get("route"))
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
		IPV4AssignMode:   ztcentral.IPV4AssignMode{ZeroTier: true},
		IPV6AssignMode:   ztcentral.IPV6AssignMode{ZeroTier: true},
	})

	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to create ZeroTier Network",
			Detail:   fmt.Sprintf("CreateNetwork returned error: %v", err),
		})
		return diags
	}

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
	d.Set("route", mktfRoutes(ztNetwork.Config.Routes))
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

	if d.HasChange("route") {
		changed = true
		var err error
		ztNetwork.Config.Routes, err = mkRoutes(d.Get("route"))
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
