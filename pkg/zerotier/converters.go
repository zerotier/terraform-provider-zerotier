package zerotier

import (
	"errors"
	"math/big"
	"net"
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

func fetchStringList(vs ValidatedSchema, attr string) *[]string {
	return toStringList(vs.Get(attr).([]interface{})).(*[]string)
}

func toStringList(i interface{}) interface{} {
	ray := &[]string{}
	for _, x := range i.([]interface{}) {
		*ray = append(*ray, x.(string))
	}
	return ray
}

func fetchIntList(vs ValidatedSchema, attr string) *[]int {
	return toIntList(vs.Get(attr).([]interface{})).(*[]int)
}

func toIntList(i interface{}) interface{} {
	ray := &[]int{}
	for _, x := range i.([]interface{}) {
		*ray = append(*ray, x.(int))
	}
	return ray
}

func mkIPRangeFromCIDR(cidr interface{}) (spec.IPRange, error) {
	iprange := spec.IPRange{}

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

	ipFirst := first.String()
	ipLast := last.String()

	iprange = spec.IPRange{
		IpRangeStart: &ipFirst,
		IpRangeEnd:   &ipLast,
	}

	return iprange, nil
}

func mkIPRange(ranges interface{}) (interface{}, diag.Diagnostics) {
	ret := []spec.IPRange{}

	for _, r := range ranges.(*schema.Set).List() {
		m := r.(map[string]interface{})
		// FIXME: if cidr is supplied, start/end simply are not considered. may want
		//			  to hard-validate this later.
		if cidr, ok := m["cidr"]; ok && cidr.(string) != "" {
			ipRange, err := mkIPRangeFromCIDR(cidr)
			if err != nil {
				return ret, diag.FromErr(err)
			}

			ret = append(ret, ipRange)
		} else {
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
	}

	return ret, nil
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
		}

		ret = append(ret, spec.Route{
			Target: &target,
			Via:    &via,
		})
	}

	return ret, nil
}

func mktfRoutes(routes interface{}) interface{} {
	ret := []map[string]interface{}{}

	for _, route := range routes.([]spec.Route) {
		ret = append(ret, map[string]interface{}{
			"target": *route.Target,
			"via":    *route.Via,
		})
	}

	return ret
}

func mktfRanges(ranges interface{}) interface{} {
	ret := []map[string]interface{}{}

	for _, r := range ranges.([]spec.IPRange) {
		ret = append(ret, map[string]interface{}{
			"start": *r.IpRangeStart,
			"end":   *r.IpRangeEnd,
		})
	}

	return ret
}

func mktfipv6assign(i interface{}) interface{} {
	ipv6 := i.(*spec.IPV6AssignMode)

	m := map[string]interface{}{}

	iter := map[string]*bool{
		"zerotier": ipv6.Zt,
		"sixplane": ipv6.N6plane,
		"rfc4193":  ipv6.Rfc4193,
	}

	for key, b := range iter {
		if b != nil {
			m[key] = ptrBool(b)
		}
	}

	return m
}

func mktfipv4assign(i interface{}) interface{} {
	ipv4 := i.(*spec.IPV4AssignMode)
	m := map[string]interface{}{}
	if ipv4.Zt != nil {
		m["zerotier"] = ptrBool(ipv4.Zt)
	}

	return m
}

func mkipv4assign(assignments interface{}) (interface{}, diag.Diagnostics) {
	m := assignments.(map[string]interface{})
	var zt bool
	if z, ok := m["zerotier"]; ok {
		zt = z.(bool)
	} else {
		zt = true // default
	}

	return &spec.IPV4AssignMode{Zt: boolPtr(zt)}, nil
}

func mkipv6assign(assignments interface{}) (interface{}, diag.Diagnostics) {
	m := assignments.(map[string]interface{})
	var zt bool
	if z, ok := m["zerotier"]; ok {
		zt = z.(bool)
	} else {
		zt = true // default
	}

	var sixPlane bool
	if s, ok := m["sixplane"]; ok {
		sixPlane = s.(bool)
	}

	var rfc4193 bool
	if r, ok := m["rfc4193"]; ok {
		rfc4193 = r.(bool)
	}

	return &spec.IPV6AssignMode{
		Zt:      boolPtr(zt),
		N6plane: boolPtr(sixPlane),
		Rfc4193: boolPtr(rfc4193),
	}, nil
}
