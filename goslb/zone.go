package goslb

import (
	"strings"
	"fmt"
	"encoding/json"
	"errors"
	"net"
	"github.com/miekg/dns"
)

type Record interface {
	DnsResponse(clientIP net.IP) []dns.RR
}

type Zone struct {
	domain  string
	records map[string]*Record
}

func (z *Zone) DomainExists(s string) bool {
	_, found := z.records[s]
	return found
}

func (z *Zone) IsValid(s *Record) error {
	// Validate service name "belongs to the service domain"
	if !strings.HasSuffix(s.Domain, "."+z.domain) {
		return errors.New(fmt.Sprintf("Service '%v' does not belong to service domain '%v'", s.Domain, z.domain))
	}

	// Validate and normalize monitor type
	mt := strings.ToUpper(s.Monitor.Type)
	if mt != "TCP" && mt != "HTTP" && mt != "" {
		return errors.New(fmt.Sprintf("Monitor type '%v' of service '%v' is not supported", s.Monitor.Type, s.Domain))
	}
	s.Monitor.Type = mt

	return nil
}

func (z *Zone) Count() int {
	return len(z.records)
}

func (z *Zone) List() (r []*Record) {
	for _, v := range z.records {
		r = append(r, v)
	}
	return r
}

func (z *Zone) GetRecord(s string) *Record  {
	return z.records[s]
}

func (z *Zone) Add(s *Service) error {
	if err := etcdClient.SaveService(s); err != nil {
		return err
	}

	// Add service to the domain and start health monitors
	z.records[s.Domain] = s
	for i := range s.Endpoints {
		if s.Endpoints[i].Enabled {
			s.Endpoints[i].startMonitor(&s.Monitor)
		}
	}
	return nil
}

func (z *Zone) Update(new *Service) error {
	if err := etcdClient.SaveService(new); err != nil {
		return err
	}
	current := z.records[new.Domain]
	oldEndpoints := current.Endpoints

	// Carry over current health status and start monitors
	for i := range new.Endpoints {
		for j := range current.Endpoints {
			if new.Endpoints[i].IP.Equal(current.Endpoints[j].IP) {
				new.Endpoints[i].healthy = current.Endpoints[j].healthy
			}
		}
		if new.Endpoints[i].Enabled {
			new.Endpoints[i].startMonitor(&new.Monitor)
		}
	}

	// update objects
	current.Monitor = new.Monitor
	current.Endpoints = new.Endpoints

	// stop old monitors
	for i := range oldEndpoints {
		oldEndpoints[i].stopMonitor()
	}

	return nil
}

func (z *Zone) Delete(name string) error {
	for _, ep := range z.records[name].Endpoints {
		ep.stopMonitor()
	}
	delete(z.records, name)
	return etcdClient.DeleteService(name)
}

var zone Zone

func InitZone(domain string) error {
	zone = Zone{domain: domain, records: make(map[string]*Service)}

	services, err := etcdClient.ListServices()
	if err != nil {
		return err
	}
	for _, svcstr := range services {
		s := &Service{}
		if err := json.Unmarshal(svcstr, &s); err != nil {
			log.WithError(err).Error("Failed to unmarshal service loaded from etcd")
			return err
		}
		zone.Add(s)
	}
	return nil
}