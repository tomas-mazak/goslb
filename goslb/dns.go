package goslb

import (
	"github.com/miekg/dns"
)

func DnsServer(config *Config) {
	dns.HandleFunc(config.Domain, handleGSLB)

	Log.Infof("Starting DNS server on %v", config.BindAddrDNS)
	server := &dns.Server{Addr: config.BindAddrDNS, Net: "udp"}
	if err := server.ListenAndServe(); err != nil {
		Log.WithError(err).Fatal("Failed to start DNS server")
	}
}

func handleGSLB(w dns.ResponseWriter, r *dns.Msg) {
	Log.Debugf("Received request from %v for %v", w.RemoteAddr(), r.Question[0].Name)
	m := new(dns.Msg)
	m.SetReply(r)
	if svc, found := Services[r.Question[0].Name]; found {
		Log.Debugf("%v: %v", svc.Domain, r.Question[0].Name)
		for _, ep := range svc.Endpoints {
			if ! ep.Healthy {
				continue
			}
			rr := &dns.A{
				Hdr: dns.RR_Header{Name: svc.Domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 1},
				A: ep.IP,
			}
			m.Answer = append(m.Answer, rr)
		}
	} else {
		Log.Debugf("Domain %v not found", r.Question[0].Name)
		m.Rcode = dns.RcodeNameError
	}
	w.WriteMsg(m)
}
