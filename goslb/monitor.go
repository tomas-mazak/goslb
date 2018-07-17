package goslb

import (
	"net"
	"time"
	"fmt"
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

	for {
		select {
		case <-m.stopped:
			log.Debugf("Stopping monitor for %v\n", m.endpoint.IP)
			return
		default:
		}
		conn, err := net.DialTimeout("tcp", addrStr, m.monitor.Timeout * time.Second)
		if err == nil {
			log.Debugf("Endpoint %v healthy\n", m.endpoint.IP)
			m.endpoint.Healthy = true
			conn.Close()
		} else {
			log.Debugf("Endpoint %v unhealthy: %v\n", m.endpoint.IP, err)
			m.endpoint.Healthy = false
		}
		time.Sleep(m.monitor.Interval * time.Second)
	}
}


func (m *TcpMonitor) stop() {
	m.stopped <- true
}

