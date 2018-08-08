package goslb

import (
	"github.com/miekg/dns"
	"math/rand"
	"net"
	"strings"
	"time"
)

type Service struct {
	Domain    string
	Endpoints []Endpoint
	Monitor   Monitor
}

func (s *Service) DnsResponse(clientIP net.IP) (resp []dns.RR) {
	for _, ip := range s.GetResponseForClient(clientIP) {
		rr := &dns.A{
			Hdr: dns.RR_Header{Name: s.Domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 1},
			A:   ip,
		}
		resp = append(resp, rr)
	}
	return resp
}

func (s *Service) GetResponseForClient(clientIP net.IP) []net.IP {
	/*
		Logic:
		1. Filter enabled & healthy endpoints
		2. Filter those with highest priority
		3. Filter those in the nearest site (only local vs remote)
		4. Randomize
		5. Return first 3

		!! Not atomic, think about races
	*/
	clientSite, clientSiteFound := siteMatcher.GetSite(clientIP)
	ret := make([]net.IP, 10)[:0]

	// Filter enabled & healthy endpoints that have the highest priority
	maxPriority := -1
	for _, v := range s.Endpoints {
		if v.Enabled && v.healthy && v.Priority > maxPriority {
			maxPriority = v.Priority
		}
	}
	log.Debugf("Max priority %v", maxPriority)

	// Try to get local endpoints
	if clientSiteFound {
		for _, v := range s.Endpoints {
			if v.Enabled && v.healthy && v.Priority == maxPriority && v.Site == clientSite {
				ret = append(ret, v.IP)
			}
		}
		log.Debugf("Local endpoints %v", len(ret))
	}

	// If no local endpoints, try any
	if len(ret) == 0 {
		for _, v := range s.Endpoints {
			if v.Enabled && v.healthy && v.Priority == maxPriority {
				ret = append(ret, v.IP)
			}
		}
	}
	log.Debugf("Other endpoints %v", len(ret))

	// Randomize
	rand.Shuffle(len(ret), func(i, j int) {
		ret[i], ret[j] = ret[j], ret[i]
	})

	// Return maximum 3 responses
	if len(ret) > 3 {
		return ret[:3]
	}
	return ret
}

type Endpoint struct {
	IP              net.IP
	Enabled         bool
	Priority        int
	Site            string
	healthy         bool
	monitorInstance MonitorType
	lastCheck       time.Time
	lastError       error
}

func (ep *Endpoint) setHealth(healthy bool, err error) {
	if ep.healthy != healthy || ep.lastCheck.IsZero() {
		if healthy {
			log.Infof("Endpoint %v healthy", ep.IP)
		} else {
			log.WithError(err).Warningf("Endpoint %v unhealthy", ep.IP)
		}
	}
	ep.healthy = healthy
	ep.lastError = err
	ep.lastCheck = time.Now()
}

func (ep *Endpoint) startMonitor(m *Monitor) {
	ep.stopMonitor()
	switch strings.ToUpper(m.Type) {
	case "TCP":
		ep.monitorInstance = &TcpMonitor{monitor: m, endpoint: ep}
	case "HTTP":
		ep.monitorInstance = &HttpMonitor{monitor: m, endpoint: ep}
	default:
		log.Infof("Endpoint %v permanently up (no monitor configured)", ep.IP)
		ep.healthy = true
		return
	}
	go ep.monitorInstance.start()
}

func (ep *Endpoint) stopMonitor() {
	if ep.monitorInstance != nil {
		ep.monitorInstance.stop()
		ep.monitorInstance = nil
	}
}

type Monitor struct {
	Type         string        // either TCP or HTTP
	Interval     time.Duration // monitor interval in seconds
	Timeout      time.Duration // request timeout in seconds
	Port         int           // TCP port to be monitored
	Uri          string        // (HTTP only) URI to get
	SuccessCodes []int         // (HTTP only) HTTP codes indicating success
	SSL          bool          // (HTTP only) SSL enabled (use HTTPS)
	Head         bool          // (HTTP only) Use HEAD instead of GET method
}
