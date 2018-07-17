package goslb

import (
	"github.com/yl2chen/cidranger"
	"net"
)

type SiteMatcher map[string]cidranger.Ranger

func (sm *SiteMatcher) GetSite(ip net.IP) (string, bool) {
	for site, ranger := range *sm {
		if contains, _ := ranger.Contains(ip); contains {
			return site, true
		}
	}
	return "", false
}

var siteMatcher SiteMatcher

func InitSiteMatcher(config *Config) {
	siteMatcher = make(SiteMatcher)
	for k, v := range config.SiteMap {
		siteMatcher[k] = cidranger.NewPCTrieRanger()
		for _, ip := range v {
			_, subnet, _ := net.ParseCIDR(ip)
			siteMatcher[k].Insert(cidranger.NewBasicRangerEntry(*subnet))
			log.Debugf("SiteMatcher: loaded site %v subnet %v", k, subnet)
		}
	}
}