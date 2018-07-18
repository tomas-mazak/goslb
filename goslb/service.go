package goslb

import (
	"time"
	"net"
	"encoding/json"
	"math/rand"
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

func (sd *ServiceDomain) Get(s string) *Service  {
	return sd.services[s]
}

func (sd *ServiceDomain) Add(s *Service) error {
	etcdClient.SaveService(s)
	sd.services[s.Domain] = s
	for i := range s.Endpoints {
		s.Endpoints[i].monitorInstance = NewMonitorInstance(&s.Monitor, &s.Endpoints[i])
		go s.Endpoints[i].monitorInstance.start()
	}
	return nil
}

func (sd *ServiceDomain) Delete(name string) error {
	for _, ep := range sd.services[name].Endpoints {
		ep.monitorInstance.stop()
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
			log.WithError(err).Error("Failed to unmarshall service loaded from etcd")
			return err
		}
		serviceDomain.Add(s)
	}
	return nil
}
