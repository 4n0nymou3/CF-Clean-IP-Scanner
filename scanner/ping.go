package scanner

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/fatih/color"
)

const (
	pingTimeout   = 1 * time.Second
	port          = 443
	maxGoroutines = 100
)

type PingResult struct {
	IP      *net.IPAddr
	Latency time.Duration
	Success bool
}

func pingIP(ip *net.IPAddr) PingResult {
	start := time.Now()
	
	var fullAddress string
	if isIPv4(ip.String()) {
		fullAddress = fmt.Sprintf("%s:%d", ip.String(), port)
	} else {
		fullAddress = fmt.Sprintf("[%s]:%d", ip.String(), port)
	}
	
	conn, err := net.DialTimeout("tcp", fullAddress, pingTimeout)
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

func PingIPs(ips []*net.IPAddr) []PingResult {
	var results []PingResult
	var mu sync.Mutex
	var wg sync.WaitGroup
	
	semaphore := make(chan struct{}, maxGoroutines)
	
	cyan := color.New(color.FgCyan)
	total := len(ips)
	completed := 0
	successCount := 0
	
	cyan.Printf("Testing latency for %d IPs...\n", total)
	
	for _, ip := range ips {
		wg.Add(1)
		semaphore <- struct{}{}
		
		go func(ipAddr *net.IPAddr) {
			defer wg.Done()
			defer func() { <-semaphore }()
			
			result := pingIP(ipAddr)
			
			mu.Lock()
			completed++
			if result.Success && result.Latency < 500*time.Millisecond {
				results = append(results, result)
				successCount++
			}
			if completed%100 == 0 {
				cyan.Printf("Progress: %d/%d IPs tested (found: %d)\n", completed, total, successCount)
			}
			mu.Unlock()
		}(ip)
	}
	
	wg.Wait()
	
	green := color.New(color.FgGreen)
	green.Printf("Latency test completed: %d responsive IPs found\n\n", len(results))
	
	return results
}