package zerotier

import (
	"reflect"
	"testing"

	"github.com/zerotier/go-ztcentral"
)

func TestCIDR(t *testing.T) {
	type test struct {
		fail bool
		r    ztcentral.IPRange
	}

	table := map[string]test{
		"10.0.0.0/24": {
			r: ztcentral.IPRange{
				Start: "10.0.0.0",
				End:   "10.0.0.255",
			},
		},
		"10.0.0.0/20": {
			r: ztcentral.IPRange{
				Start: "10.0.0.0",
				End:   "10.0.15.255",
			},
		},
		"10.0.0.0/16": {
			r: ztcentral.IPRange{
				Start: "10.0.0.0",
				End:   "10.0.255.255",
			},
		},
		"10.0.0.0/12": {
			r: ztcentral.IPRange{
				Start: "10.0.0.0",
				End:   "10.15.255.255",
			},
		},
		"10.0.0.0/8": {
			r: ztcentral.IPRange{
				Start: "10.0.0.0",
				End:   "10.255.255.255",
			},
		},
		"10.0.0.0/1234": {
			fail: true,
		},
		"10.0.0.0/4141": {
			fail: true,
		},
		"10.0.0.0/128": {
			fail: true,
		},
		"0.0.0.0/0": {
			r: ztcentral.IPRange{
				Start: "0.0.0.0",
				End:   "255.255.255.255",
			},
		},

		// ipv6 now!

		"fe80::/96": {
			r: ztcentral.IPRange{
				Start: "fe80::",
				End:   "fe80::ffff:ffff",
			},
		},
		"fe80::/48": {
			r: ztcentral.IPRange{
				Start: "fe80::",
				End:   "fe80::ffff:ffff:ffff:ffff:ffff",
			},
		},
		"fe80::1/128": {
			r: ztcentral.IPRange{
				Start: "fe80::1",
				End:   "fe80::1",
			},
		},
	}

	for cidr, r := range table {
		compareRange, err := mkIPRangeFromCIDR(cidr)
		if r.fail && err == nil {
			t.Fatalf("Test %q was supposed to fail and did not.", cidr)
		} else if !r.fail && err != nil {
			t.Fatalf("Positive test %q failed with error: %v", cidr, err)
		} else if !r.fail && !reflect.DeepEqual(compareRange, r.r) {
			t.Fatalf("Test %q was supposed to pass, but yielded an invalid value: %v: %v", cidr, compareRange, r.r)
		}
	}
}
