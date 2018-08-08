package goslb_test

import (
	"net"
	"testing"

	"github.com/tomas-mazak/goslb/goslb"
)

func ip(s string) net.IP {
	return net.ParseIP(s)
}

func ips(strs ...string) []net.IP {
	ret := make([]net.IP, len(strs))
	for i, s := range strs {
		ret[i] = ip(s)
	}
	return ret
}

func sortedCompare(a []net.IP, b []net.IP) bool {
	return false
}

// TODO
// Single site, all enabled, all healthy, same priority
//   - 1 endpoint
//   - more than 3 results cropped to 3
//   - randomization
//
// No valid endpoints:
//   - all disabled
//   - all unhealthy
//
// Different priority:
//   - highest priority healthy
//   - highest priority unhealthy, second highest returned
//
// Sites:
//   - local endpoints healthy
//   - local endpoints unhealthy, remote healthy
//
// All filters:
// 2 sites, different priorities, healthy/unhealthy
//
func TestGetOrdered(t *testing.T) {
	// Single site, all enabled, all healthy, same priority
	var good = []struct {
		ep  []goslb.Endpoint
		res []net.IP
	}{
		{[]goslb.Endpoint{goslb.Endpoint{IP: ip("10.0.0.1"), Enabled: true, healthy: true}}, ips("10.0.0.1")},
	}
	//e1 := goslb.Endpoint{IP: net.ParseIP("10.0.0.1"), Enabled: true, healthy: true}
	//e2 := Endpoint{IP: net.ParseIP("10.0.0.2"), Enabled: true, healthy: true}
	//s := goslb.Service{
	//	Domain:    "foo.goslb.",
	//	Endpoints: []goslb.Endpoint{e1},
	//}
	for _, tc := range good {
		res := (&goslb.Service{Endpoints: tc.ep}).GetResponseForClient(ip("127.0.0.1"))
		if !sortedCompare(res, tc.res) {
			t.Errorf("For %v -- expected: %v; got: %v", tc.ep, tc.res, res)
		}
	}
}
