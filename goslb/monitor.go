package goslb

import (
	"net"
	"time"
	"fmt"
	"net/http"
	"strings"
)

type MonitorType interface {
	start()
	stop()
}


type TcpMonitor struct {
	monitor *Monitor
	endpoint *Endpoint
	stopped chan bool
}

func (m *TcpMonitor) start() {
	m.stopped = make(chan bool, 1)
	addrStr := fmt.Sprintf("%v:%v", m.endpoint.IP, m.monitor.Port)

	log.Infof("Starting %v monitor for %v", m.monitor.Type, m.endpoint.IP)
	for {
		select {
		case <-m.stopped:
			log.Infof("Stopping %v monitor for %v", m.monitor.Type, m.endpoint.IP)
			return
		default:
		}
		conn, err := net.DialTimeout("tcp", addrStr, m.monitor.Timeout * time.Second)
		if err == nil {
			healthy(m.endpoint, true, nil)
			conn.Close()
		} else {
			healthy(m.endpoint, false, err)
		}
		time.Sleep(m.monitor.Interval * time.Second)
	}
}

func (m *TcpMonitor) stop() {
	m.stopped <- true
}


type HttpMonitor struct {
	monitor *Monitor
	endpoint *Endpoint
	stopped chan bool
}

func (m *HttpMonitor) start() {
	m.stopped = make(chan bool, 1)
	client := &http.Client{Timeout: m.monitor.Timeout * time.Second}
	url := fmt.Sprintf("http://%v:%v%v", m.endpoint.IP, m.monitor.Port, m.monitor.Uri)

	log.Infof("Starting %v monitor for %v", m.monitor.Type, m.endpoint.IP)
	for {
		select {
		case <-m.stopped:
			log.Infof("Stopping %v monitor for %v", m.monitor.Type, m.endpoint.IP)
			return
		default:
		}
		resp, err := client.Get(url)
		if err != nil {
			healthy(m.endpoint, false, err)
			continue
		}
		success := false
		for _, v := range m.monitor.SuccessCodes {
			if resp.StatusCode == v {
				success = true
				healthy(m.endpoint, true, nil)
				break
			}
		}
		if !success {
			healthy(m.endpoint, false, nil)
		}
		time.Sleep(m.monitor.Interval * time.Second)
	}
}

func (m *HttpMonitor) stop() {
	m.stopped <- true
}

func healthy(ep *Endpoint, healthy bool, err error) {
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

func NewMonitorInstance(m *Monitor, ep *Endpoint) MonitorType {
	switch strings.ToUpper(m.Type) {
	case "TCP":
		return &TcpMonitor{monitor: m, endpoint: ep}
	case "HTTP":
		return &HttpMonitor{monitor: m, endpoint: ep}
	default:
		return nil
	}
}