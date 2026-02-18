package scanner

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
)

const (
	pingTimeout   = 2 * time.Second
	port          = 443
	maxGoroutines = 100
	latencyLimit  = 1000 * time.Millisecond
)

type PingResult struct {
	IP      *net.IPAddr
	Latency time.Duration
	Success bool
}

func pingIP(ctx context.Context, ip *net.IPAddr) PingResult {
	start := time.Now()

	var fullAddress string
	if isIPv4(ip.String()) {
		fullAddress = fmt.Sprintf("%s:%d", ip.String(), port)
	} else {
		fullAddress = fmt.Sprintf("[%s]:%d", ip.String(), port)
	}

	dialer := &net.Dialer{Timeout: pingTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", fullAddress)
	if err != nil {
		return PingResult{IP: ip, Success: false}
	}
	defer conn.Close()

	latency := time.Since(start)
	return PingResult{IP: ip, Latency: latency, Success: true}
}

func PingIPs(ctx context.Context, ips []*net.IPAddr, bytesUsed *int64) []PingResult {
	var results []PingResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, maxGoroutines)
	total := len(ips)
	completed := 0
	successCount := 0

	cyan := color.New(color.FgCyan)
	cyan.Printf("Testing latency for %d IPs...\n", total)
	fmt.Println()

	const barWidth = 50

	for _, ip := range ips {
		select {
		case <-ctx.Done():
			goto waitAndReturn
		case semaphore <- struct{}{}:
		}

		wg.Add(1)
		go func(ipAddr *net.IPAddr) {
			defer wg.Done()
			defer func() { <-semaphore }()

			atomic.AddInt64(bytesUsed, 350)
			result := pingIP(ctx, ipAddr)

			mu.Lock()
			completed++
			if result.Success && result.Latency < latencyLimit {
				results = append(results, result)
				successCount++
			}
			progress := float64(completed) / float64(total)
			bar := buildProgressBar(int(progress*float64(barWidth)), barWidth)
			fmt.Printf("\r%s %3d%% (%d/%d) - Found: %d",
				bar, int(progress*100), completed, total, successCount)
			mu.Unlock()
		}(ip)
	}

waitAndReturn:
	wg.Wait()

	fmt.Println()
	fmt.Println()
	color.New(color.FgGreen).Printf("Latency test completed: %d responsive IPs found\n\n", len(results))

	return results
}