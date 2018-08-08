package goslb

import (
	"github.com/miekg/dns"
	"net"
)

func DnsServer(config *Config) {
	dns.HandleFunc(config.Domain, handleGSLB)

	log.Infof("Starting DNS server on %v", config.BindAddrDNS)
	server := &dns.Server{Addr: config.BindAddrDNS, Net: "udp"}
	if err := server.ListenAndServe(); err != nil {
		log.WithError(err).Fatal("Failed to start DNS server")
	}
}

func handleGSLB(w dns.ResponseWriter, r *dns.Msg) {
	clientIP := w.RemoteAddr().(*net.UDPAddr).IP
	log.Debugf("Received request from %v for %v", clientIP, r.Question[0].Name)
	m := new(dns.Msg)
	m.SetReply(r)
	if zone.DomainExists(r.Question[0].Name) {
		record := zone.GetRecord(r.Question[0].Name)
		m.Answer = record.DnsResponse(clientIP)
	} else {
		log.Debugf("Domain %v not found", r.Question[0].Name)
		m.Rcode = dns.RcodeNameError
	}
	w.WriteMsg(m)
}
