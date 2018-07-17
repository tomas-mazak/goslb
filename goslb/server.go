package goslb

func StartServer(config *Config) {
		// setup logging
		InitLogger(config)

		// init Site Matcher
		InitSiteMatcher(config)

		// init the ETCD persistent store and load services
		InitEtcdClient(config)

		// init Service Domain
		InitServiceDomain(config.Domain)

		// start API
		go InitApiServer(config)

		// start DNS server
		DnsServer(config)
}
