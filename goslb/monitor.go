package goslb

import (
	"net"
	"time"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"math/rand"
	"log"
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
			fmt.Printf("Stopping monitor for %v\n", m.endpoint.IP)
			return
		default:
		}
		conn, err := net.DialTimeout("tcp", addrStr, m.monitor.Timeout * time.Second)
		if err == nil {
			fmt.Printf("Endpoint %v healthy\n", m.endpoint.IP)
			m.endpoint.Healthy = true
			conn.Close()
		} else {
			fmt.Printf("Endpoint %v unhealthy: %v\n", m.endpoint.IP, err)
			m.endpoint.Healthy = false
		}
		time.Sleep(m.monitor.Interval * time.Second)
	}
}


func (m *TcpMonitor) stop() {
	m.stopped <- true
}

type PingMonitor struct {
	monitor *Monitor
	endpoint *Endpoint
}

func (m *PingMonitor) start() {

	fmt.Printf("Monitoring %v\n", m.endpoint.IP)
	var seq = 0

	conn, err := icmp.ListenPacket("ip4:icmp", "")
	if err != nil {
		log.Fatalln("Can't open ICMP session")
	}
	defer conn.Close()

	for {
		msg, _ := (&icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID: rand.Intn(65535),
				Seq: seq,
				Data: timeToBytes(time.Now()),
		}}).Marshal(nil)
		conn.WriteTo(msg, &net.IPAddr{IP: m.endpoint.IP})

		conn.SetReadDeadline(time.Now().Add(time.Second * m.monitor.Timeout))
		n, _, err := conn.ReadFrom(msg)
		if err != nil {
			if neterr, ok := err.(*net.OpError); ok {
				if neterr.Timeout() {
					// Read timeout
					m.endpoint.Healthy = false
					fmt.Printf("Endpoint %v failed\n", m.endpoint.IP)
				} else {
					log.Fatalln("ICMP connection error")
				}
			}
		} else {
			pkt, err := icmp.ParseMessage(1, msg[:n])
			if err != nil {
				fmt.Printf("Can't parse ICMP packet: %v\n", err)
				continue
			}
			if pkt.Type == ipv4.ICMPTypeEchoReply {
				m.endpoint.Healthy = true
				fmt.Printf("Endpoint %v healthy\n", m.endpoint.IP)
			} else {
				fmt.Printf("Weird ICMP packet received: %v\n", pkt.Type)
			}
		}

		seq++
		time.Sleep(m.monitor.Interval * time.Second)
	}
}




func bytesToTime(b []byte) time.Time {
	var nsec int64
	for i := uint8(0); i < 8; i++ {
		nsec += int64(b[i]) << ((7 - i) * 8)
	}
	return time.Unix(nsec/1000000000, nsec%1000000000)
}

func timeToBytes(t time.Time) []byte {
	nsec := t.UnixNano()
	b := make([]byte, 8)
	for i := uint8(0); i < 8; i++ {
		b[i] = byte((nsec >> ((7 - i) * 8)) & 0xff)
	}
	return b
}
