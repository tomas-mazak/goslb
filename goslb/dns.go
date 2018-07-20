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
	if serviceDomain.Exists(r.Question[0].Name) {
		svc := serviceDomain.Get(r.Question[0].Name)
		log.Debugf("%v: %v", svc.Domain, r.Question[0].Name)
		for _, ip := range svc.GetOrdered(clientIP) {
			rr := &dns.A{
				Hdr: dns.RR_Header{Name: svc.Domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 1},
				A: ip,
			}
			m.Answer = append(m.Answer, rr)
		}
	} else {
		log.Debugf("Domain %v not found", r.Question[0].Name)
		m.Rcode = dns.RcodeNameError
	}
	w.WriteMsg(m)
}
