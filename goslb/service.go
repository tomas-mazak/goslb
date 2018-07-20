package goslb

import (
	"time"
	"net"
	"encoding/json"
	"math/rand"
	"strings"
	"errors"
	"fmt"
)

type Service struct {
	Domain    string
	Endpoints []Endpoint
	Monitor   Monitor
}

func (s *Service) GetOrdered(clientIP net.IP) []net.IP {
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
		if v.Enabled && v.Healthy && v.Priority > maxPriority {
			maxPriority = v.Priority
		}
	}
	log.Debugf("Max priority %v", maxPriority)

	// Try to get local endpoints
	if clientSiteFound {
		for _, v := range s.Endpoints {
			if v.Enabled && v.Healthy && v.Priority == maxPriority && v.Site == clientSite {
				ret = append(ret, v.IP)
			}
		}
		log.Debugf("Local endpoints %v", len(ret))
	}

	// If no local endpoints, try any
	if len(ret) == 0 {
		for _, v := range s.Endpoints {
			if v.Enabled && v.Healthy && v.Priority == maxPriority {
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
	Healthy         bool
	monitorInstance MonitorType
	lastCheck		time.Time
}

func (ep *Endpoint) setHealth(healthy bool, err error) {
	if ep.Healthy != healthy || ep.lastCheck.IsZero() {
		if healthy {
			log.Infof("Endpoint %v healthy", ep.IP)
		} else {
			log.WithError(err).Warningf("Endpoint %v unhealthy", ep.IP)
		}
	}
	ep.Healthy = healthy
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
	Type         string
	Interval     time.Duration
	Timeout      time.Duration
	Port         int
	Uri          string
	SuccessCodes []int
}

type ServiceDomain struct {
	domain string
	services map[string]*Service
}

func (sd *ServiceDomain) Exists(s string) bool {
	_, found := sd.services[s]
	return found
}

func (sd *ServiceDomain) IsValid(s *Service) error {
	// Validate service name "belongs to the service domain"
	if !strings.HasSuffix(s.Domain, "."+sd.domain) {
		return errors.New(fmt.Sprintf("Service '%v' does not belong to service domain '%v'", s.Domain, sd.domain))
	}

	// Validate and normalize monitor type
	mt := strings.ToUpper(s.Monitor.Type)
	if mt != "TCP" && mt != "HTTP" {
		return errors.New(fmt.Sprintf("Monitor type '%v' of service '%v' is not supported", s.Monitor.Type, s.Domain))
	}
	s.Monitor.Type = mt

	return nil
}

func (sd *ServiceDomain) Get(s string) *Service  {
	return sd.services[s]
}

func (sd *ServiceDomain) Add(s *Service) error {
	if err := etcdClient.SaveService(s); err != nil {
		return err
	}

	// Add service to the domain and start health monitors
	sd.services[s.Domain] = s
	for i := range s.Endpoints {
		if s.Endpoints[i].Enabled {
			s.Endpoints[i].startMonitor(&s.Monitor)
		}
	}
	return nil
}

func (sd *ServiceDomain) Update(new *Service) error {
	if err := etcdClient.SaveService(new); err != nil {
		return err
	}
	current := sd.services[new.Domain]
	oldEndpoints := current.Endpoints

	// Carry over current health status and start monitors
	for i := range new.Endpoints {
		for j := range current.Endpoints {
			if new.Endpoints[i].IP.Equal(current.Endpoints[j].IP) {
				new.Endpoints[i].Healthy = current.Endpoints[j].Healthy
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

func (sd *ServiceDomain) Delete(name string) error {
	for _, ep := range sd.services[name].Endpoints {
		ep.stopMonitor()
	}
	delete(sd.services, name)
	return etcdClient.DeleteService(name)
}

var serviceDomain ServiceDomain

func InitServiceDomain(domain string) error {
	serviceDomain = ServiceDomain{domain: domain, services: make(map[string]*Service)}

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
		serviceDomain.Add(s)
	}
	return nil
}
