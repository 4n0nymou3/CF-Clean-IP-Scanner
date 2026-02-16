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
	fmt.Println()
	
	barWidth := 50
	
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
			
			progress := float64(completed) / float64(total)
			filledWidth := int(progress * float64(barWidth))
			
			bar := "["
			for j := 0; j < barWidth; j++ {
				if j < filledWidth {
					bar += "="
				} else if j == filledWidth {
					bar += ">"
				} else {
					bar += " "
				}
			}
			bar += "]"
			
			fmt.Printf("\r%s %3d%% (%d/%d) - Found: %d", bar, int(progress*100), completed, total, successCount)
			
			mu.Unlock()
		}(ip)
	}
	
	wg.Wait()
	
	fmt.Println()
	fmt.Println()
	green := color.New(color.FgGreen)
	green.Printf("Latency test completed: %d responsive IPs found\n\n", len(results))
	
	return results
}