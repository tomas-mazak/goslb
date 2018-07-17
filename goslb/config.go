package goslb

type Config struct {
	LogLevel string
	BindAddrAPI string
	BindAddrDNS string
	EtcdServers []string
	Domain string
	SiteMap map[string][]string
}
