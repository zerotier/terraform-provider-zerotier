package zerotier

import (
	"errors"
	"math/big"
	"net"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/zerotier/go-ztcentral"
)

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
