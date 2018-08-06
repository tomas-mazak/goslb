package goslb

import (
	"net"
	"github.com/kentik/patricia/string_tree"
	"github.com/kentik/patricia"
)

type SiteMatcher struct {
	trie *string_tree.TreeV4
}

func (sm *SiteMatcher) AddRecord(ip string, site string) error {
	subnet, _, err := patricia.ParseIPFromString(ip)
	if err != nil {
		return err
	}
	sm.trie.Add(*subnet, site, func(a, b string) bool {
		return a==b
	})
	return nil
}

func (sm *SiteMatcher) GetSite(ip net.IP) (string, bool) {
	found, site, err := sm.trie.FindDeepestTag(patricia.NewIPv4AddressFromBytes(ip, net.IPv4len))
	if err != nil {
		log.WithError(err).Error("Cannot resolve site for %v", ip)
	}
	return site, found
}

var siteMatcher SiteMatcher

func InitSiteMatcher(config *Config) {
	siteMatcher.trie = string_tree.NewTreeV4()
	for k, v := range config.SiteMap {
		for _, ip := range v {
			if err := siteMatcher.AddRecord(ip, k); err != nil {
				log.WithError(err).Fatal("Cannot load sites")
			}
			log.Debugf("SiteMatcher: loaded site %v subnet %v", k, ip)
		}
	}
}