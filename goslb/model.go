package goslb

import (
	"net"
	"time"
)

type Monitor struct {
	Type         string
	Interval     time.Duration
	Timeout      time.Duration
	Port         int
	Uri          string
	SuccessCodes []int
}

type Endpoint struct {
	IP net.IP
	Enabled bool
	Priority int
	Site int
	MonitorInstance MonitorType `json:"-"`
	Healthy bool `json:"-"`
}

type Service struct {
	Domain    string
	Endpoints []Endpoint
	Monitor   Monitor
}

var Services = make(map[string]Service)
