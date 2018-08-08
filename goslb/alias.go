package goslb

import (
	"github.com/miekg/dns"
	"net"
)

type Alias struct {
	Domain string
	Target string
}

func (a *Alias) DnsResponse(clientIP net.IP) (resp []dns.RR) {
	return []dns.RR{
		&dns.CNAME{
			Hdr: dns.RR_Header{Name: a.Domain, Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: 1},
			Target: a.Target,
		},
	}
}
