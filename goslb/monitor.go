package goslb

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"time"
	"crypto/tls"
)

type MonitorType interface {
	start()
	stop()
}

type TcpMonitor struct {
	monitor  *Monitor
	endpoint *Endpoint
	stopped  chan bool
}

func (m *TcpMonitor) start() {
	// random sleep up to monitor interval time: prevent bursts on startup
	time.Sleep(time.Duration(rand.Intn(int(m.monitor.Interval)*1000)) * time.Millisecond)

	m.stopped = make(chan bool, 1)
	addrStr := fmt.Sprintf("%v:%v", m.endpoint.IP, m.monitor.Port)

	log.Infof("Starting %v monitor for %v", m.monitor.Type, m.endpoint.IP)
	serverStatus.monitors++
	for {
		conn, err := net.DialTimeout("tcp", addrStr, m.monitor.Timeout*time.Second)
		if err == nil {
			m.endpoint.setHealth(true, nil)
			conn.Close()
		} else {
			m.endpoint.setHealth(false, err)
		}

		select {
		case <-m.stopped:
			log.Infof("Stopping %v monitor for %v", m.monitor.Type, m.endpoint.IP)
			return
		case <-time.After(m.monitor.Interval * time.Second):
			break
		}
	}
}

func (m *TcpMonitor) stop() {
	m.stopped <- true
	serverStatus.monitors--
}

type HttpMonitor struct {
	monitor  *Monitor
	endpoint *Endpoint
	stopped  chan bool
}

func (m *HttpMonitor) start() {
	// random sleep up to monitor interval time: prevent bursts on startup
	time.Sleep(time.Duration(rand.Intn(int(m.monitor.Interval)*1000)) * time.Millisecond)

	m.stopped = make(chan bool, 1)
	tr := &http.Transport{
		// We don't care about certs for health monitors
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    }
	client := &http.Client{
		Timeout: m.monitor.Timeout * time.Second,
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}

	scheme := "http"
	if m.monitor.SSL {
		scheme = "https"
	}
	url := fmt.Sprintf("%v://%v:%v%v", scheme, m.endpoint.IP, m.monitor.Port, m.monitor.Uri)

	method := client.Get
	if m.monitor.Head {
		method = client.Head
	}

	log.Infof("Starting %v monitor for %v", m.monitor.Type, m.endpoint.IP)
	serverStatus.monitors++
	for {
		resp, err := method(url)
		if err != nil {
			m.endpoint.setHealth(false, err)
		} else {
			success := len(m.monitor.SuccessCodes) == 0
			for _, v := range m.monitor.SuccessCodes {
				if resp.StatusCode == v {
					success = true
					break
				}
			}
			if !success {
				err = errors.New(fmt.Sprintf("Invalid HTTP status code: %v", resp.StatusCode))
			}
			m.endpoint.setHealth(success, err)
		}

		select {
		case <-m.stopped:
			log.Infof("Stopping %v monitor for %v", m.monitor.Type, m.endpoint.IP)
			return
		case <-time.After(m.monitor.Interval * time.Second):
			break
		}
	}
}

func (m *HttpMonitor) stop() {
	m.stopped <- true
	serverStatus.monitors--
}
