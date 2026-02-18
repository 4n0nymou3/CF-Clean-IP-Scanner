package scanner

import (
	"fmt"
	"net"
	"sort"
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
		ok, d := tcping(ip)
		if ok {
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
	completed := 0
	successCount := 0

	cyan := color.New(color.FgCyan)
	cyan.Printf("Testing latency for %d IPs... (Mode: TCP, Port: %d, Pings per IP: %d)\n", total, port, defaultPingTimes)
	fmt.Println()

	const barWidth = 50

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
			completed++
			if recv > 0 {
				successCount++
				avg := totalDelay / time.Duration(recv)
				results = append(results, PingResult{
					IP:       ipAddr,
					Sended:   defaultPingTimes,
					Received: recv,
					Delay:    avg,
				})
			}
			progress := float64(completed) / float64(total)
			bar := buildProgressBar(int(progress*float64(barWidth)), barWidth)
			fmt.Printf("\r%s %3d%% (%d/%d) - Available: %d",
				bar, int(progress*100), completed, total, successCount)
			mu.Unlock()
		}(ip)
	}

done:
	wg.Wait()

	fmt.Println()
	fmt.Println()
	color.New(color.FgGreen).Printf("Latency test completed: %d responsive IPs found\n\n", len(results))

	sort.Slice(results, func(i, j int) bool {
		li, lj := results[i].GetLossRate(), results[j].GetLossRate()
		if li != lj {
			return li < lj
		}
		return results[i].Delay < results[j].Delay
	})

	return results
}