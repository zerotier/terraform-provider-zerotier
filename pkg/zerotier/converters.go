package zerotier

import (
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztcentral/pkg/spec"
)

func boolPtr(b bool) *bool {
	return &b
}

func ptrBool(p *bool) bool {
	if p != nil && *p {
		return true
	}

	return false
}

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func getMemberIDs(d *schema.ResourceData) (string, string) {
	ztNetworkID := d.Get("network_id").(string)
	memberID := d.Get("member_id").(string)

	if ztNetworkID == "" && memberID == "" {
		parts := strings.Split(d.Id(), "-")
		ztNetworkID, memberID = parts[0], parts[1]
	}
	return ztNetworkID, memberID
}

func fetchStringList(d *schema.ResourceData, attr string) *[]string {
	return toStringList(d.Get(attr).([]interface{})).(*[]string)
}

func fetchStringSet(d *schema.ResourceData, attr string) *[]string {
	return toStringList(d.Get(attr).(*schema.Set).List()).(*[]string)
}

func toStringList(i interface{}) interface{} {
	ray := &[]string{}
	for _, x := range i.([]interface{}) {
		*ray = append(*ray, x.(string))
	}
	return ray
}

func fetchIntList(d *schema.ResourceData, attr string) *[]int {
	return toIntList(d.Get(attr).([]interface{})).(*[]int)
}

func fetchIntSet(d *schema.ResourceData, attr string) *[]int {
	return toIntList(d.Get(attr).(*schema.Set).List()).(*[]int)
}

func fetchTags(d []interface{}) *[][]int {
	tags := [][]int{}

	for _, tagref := range d {
		ref := []int{}
		for _, value := range tagref.([]interface{}) {
			ref = append(ref, value.(int))
		}

		tags = append(tags, ref)
	}

	return &tags
}

func toIntList(i interface{}) interface{} {
	ray := &[]int{}
	for _, x := range i.([]interface{}) {
		*ray = append(*ray, x.(int))
	}
	return ray
}

func mkIPRange(ranges interface{}) (interface{}, diag.Diagnostics) {
	ret := []spec.IPRange{}

	for _, r := range ranges.(*schema.Set).List() {
		m := r.(map[string]interface{})
		var start, end string
		if s, ok := m["start"]; ok && s.(string) != "" {
			start = s.(string)
		} else {
			return ret, diag.FromErr(errors.New("start does not exist"))
		}

		if e, ok := m["end"]; ok && e.(string) != "" {
			end = e.(string)
		} else {
			return ret, diag.FromErr(errors.New("end does not exist"))
		}

		ret = append(ret, spec.IPRange{
			IpRangeStart: &start,
			IpRangeEnd:   &end,
		})
	}

	return &ret, nil
}

func mkRoutes(routes interface{}) (interface{}, diag.Diagnostics) {
	ret := []spec.Route{}

	for _, r := range routes.(*schema.Set).List() {
		m := r.(map[string]interface{})
		var target, via string
		if t, ok := m["target"]; ok && t.(string) != "" {
			target = t.(string)
		} else {
			return ret, diag.FromErr(errors.New("target does not exist"))
		}

		if v, ok := m["via"]; ok && v.(string) != "" {
			via = v.(string)
		} else {
			via = ""
		}

		ret = append(ret, spec.Route{
			Target: &target,
			Via:    &via,
		})
	}

	return &ret, nil
}

func mktfRoutes(routes interface{}) []map[string]interface{} {
	ret := []map[string]interface{}{}

	r := routes.(*[]spec.Route)
	if r == nil {
		return ret
	}

	for _, route := range *r {
		var target, via string
		if route.Target != nil {
			target = *route.Target
		}
		if route.Via != nil {
			via = *route.Via
		}

		m := map[string]interface{}{}

		m["target"] = target
		m["via"] = via

		ret = append(ret, m)
	}

	return ret
}

func mktfRanges(ranges *[]spec.IPRange) []map[string]interface{} {
	ret := []map[string]interface{}{}

	if ranges == nil {
		return ret
	}

	for _, r := range *ranges {
		var start, end string

		if r.IpRangeStart != nil {
			start = *r.IpRangeStart
		}

		if r.IpRangeEnd != nil {
			end = *r.IpRangeEnd
		}

		m := map[string]interface{}{}
		m["start"] = start
		m["end"] = end

		ret = append(ret, m)
	}

	return ret
}

func ipv6set(m interface{}) int {
	ipv6 := m.(map[string]interface{})

	order := []string{"zerotier", "sixplane", "rfc4193"}

	res := 0

	for i, b := range order {
		if ptrBool(ipv6[b].(*bool)) {
			res |= 1 << i
		}
	}

	return res
}

func mktfipv6assign(ipv6 *spec.IPV6AssignMode) *schema.Set {
	return schema.NewSet(ipv6set, []interface{}{
		map[string]interface{}{
			"zerotier": ipv6.Zt,
			"sixplane": ipv6.N6plane,
			"rfc4193":  ipv6.Rfc4193,
		},
	})
}

func ipv4set(m interface{}) int {
	ipv4 := m.(map[string]interface{})

	if ptrBool(ipv4["zerotier"].(*bool)) {
		return 1
	}

	return 0
}

func mktfipv4assign(ipv4 *spec.IPV4AssignMode) *schema.Set {
	return schema.NewSet(ipv4set, []interface{}{map[string]interface{}{"zerotier": ipv4.Zt}})
}

func mkipv4assign(assignments interface{}) (interface{}, diag.Diagnostics) {
	m := assignments.(*schema.Set)
	zt := true

	if m.Len() > 0 {
		for _, set := range m.List() {
			tmp := set.(map[string]interface{})

			if z, ok := tmp["zerotier"]; ok {
				zt = z.(bool)
			}
		}
	}

	return &spec.IPV4AssignMode{Zt: boolPtr(zt)}, nil
}

func mkipv6assign(assignments interface{}) (interface{}, diag.Diagnostics) {
	m := assignments.(*schema.Set)
	var zt, sixPlane, rfc4193 bool

	if m.Len() > 0 {
		for _, set := range m.List() {
			tmp := set.(map[string]interface{})
			if z, ok := tmp["zerotier"]; ok {
				zt = z.(bool)
			}

			if s, ok := tmp["sixplane"]; ok {
				sixPlane = s.(bool)
			}

			if r, ok := tmp["rfc4193"]; ok {
				rfc4193 = r.(bool)
			}
		}
	}

	return &spec.IPV6AssignMode{
		Zt:      boolPtr(zt),
		N6plane: boolPtr(sixPlane),
		Rfc4193: boolPtr(rfc4193),
	}, nil
}
