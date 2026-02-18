package scanner

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/fatih/color"
)

const (
	tcpConnectTimeout = 1 * time.Second
	port              = 443
	maxRoutines       = 200
	defaultPingTimes  = 4
)

type PingResult struct {
	IP       *net.IPAddr
	Sended   int
	Received int
	Delay    time.Duration
}

func (p *PingResult) GetLossRate() float32 {
	lost := p.Sended - p.Received
	return float32(lost) / float32(p.Sended)
}

func tcping(ip *net.IPAddr) (bool, time.Duration) {
	start := time.Now()
	var addr string
	if isIPv4(ip.String()) {
		addr = fmt.Sprintf("%s:%d", ip.String(), port)
	} else {
		addr = fmt.Sprintf("[%s]:%d", ip.String(), port)
	}
	conn, err := net.DialTimeout("tcp", addr, tcpConnectTimeout)
	if err != nil {
		return false, 0
	}
	conn.Close()
	return true, time.Since(start)
}

func checkConnection(ip *net.IPAddr) (recv int, totalDelay time.Duration) {
	for i := 0; i < defaultPingTimes; i++ {
		if ok, d := tcping(ip); ok {
			recv++
			totalDelay += d
		}
	}
	return
}

func PingIPs(stopCh <-chan struct{}, ips []*net.IPAddr) []PingResult {
	var results []PingResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	control := make(chan struct{}, maxRoutines)
	total := len(ips)

	cyan := color.New(color.FgCyan)
	cyan.Printf("Start latency test (Mode: TCP, Port: %d, Range: 0 ~ 9999 ms, Packet Loss: 1.00)\n", port)

	bar := newBar(total, "Available:", "")

	for _, ip := range ips {
		select {
		case <-stopCh:
			goto done
		case control <- struct{}{}:
		}

		wg.Add(1)
		go func(ipAddr *net.IPAddr) {
			defer wg.Done()
			defer func() { <-control }()

			recv, totalDelay := checkConnection(ipAddr)

			mu.Lock()
			nowAble := len(results)
			if recv != 0 {
				nowAble++
			}
			bar.grow(1, strconv.Itoa(nowAble))
			if recv > 0 {
				avg := totalDelay / time.Duration(recv)
				results = append(results, PingResult{
					IP:       ipAddr,
					Sended:   defaultPingTimes,
					Received: recv,
					Delay:    avg,
				})
			}
			mu.Unlock()
		}(ip)
	}

done:
	wg.Wait()
	bar.done()

	sort.Slice(results, func(i, j int) bool {
		li, lj := results[i].GetLossRate(), results[j].GetLossRate()
		if li != lj {
			return li < lj
		}
		return results[i].Delay < results[j].Delay
	})

	fmt.Println()
	color.New(color.FgGreen).Printf("Latency test completed: %d responsive IPs found\n\n", len(results))

	return results
}