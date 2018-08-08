package goslb

import "os"

var serverStatus struct {
	nodeName string
	monitors int
}


func StartServer(config *Config) {
	// set the local node name
	var err error
	serverStatus.nodeName, err = os.Hostname()
	if err != nil {
		serverStatus.nodeName = "goslb"
	}

	// setup logging
	InitLogger(config, serverStatus.nodeName)

	// init Site Matcher
	InitSiteMatcher(config)

	// init the ETCD persistent store and load records
	InitEtcdClient(config)

	// init Service Domain
	InitZone(config.Domain)

	// start API
	go InitApiServer(config)

	// start DNS server
	DnsServer(config)
}
