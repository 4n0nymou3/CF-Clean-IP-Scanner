package scanner

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/fatih/color"
)

const (
	pingTimeout   = 2 * time.Second
	port          = 443
	maxGoroutines = 100
)

type PingResult struct {
	IP      string
	Latency time.Duration
	Success bool
}

func pingIP(ip string) PingResult {
	start := time.Now()
	
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), pingTimeout)
	if err != nil {
		return PingResult{IP: ip, Success: false}
	}
	defer conn.Close()
	
	latency := time.Since(start)
	
	return PingResult{
		IP:      ip,
		Latency: latency,
		Success: true,
	}
}

func PingIPs(ips []string) []PingResult {
	var results []PingResult
	var mu sync.Mutex
	var wg sync.WaitGroup
	
	semaphore := make(chan struct{}, maxGoroutines)
	
	cyan := color.New(color.FgCyan)
	total := len(ips)
	completed := 0
	
	cyan.Printf("Testing latency for %d IPs...\n", total)
	
	for _, ip := range ips {
		wg.Add(1)
		semaphore <- struct{}{}
		
		go func(ipAddr string) {
			defer wg.Done()
			defer func() { <-semaphore }()
			
			result := pingIP(ipAddr)
			
			if result.Success && result.Latency < 500*time.Millisecond {
				mu.Lock()
				results = append(results, result)
				completed++
				if completed%10 == 0 {
					cyan.Printf("Progress: %d/%d IPs tested\n", completed, total)
				}
				mu.Unlock()
			}
		}(ip)
	}
	
	wg.Wait()
	
	green := color.New(color.FgGreen)
	green.Printf("Latency test completed: %d responsive IPs found\n\n", len(results))
	
	return results
}