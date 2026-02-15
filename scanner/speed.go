package scanner

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/fatih/color"
)

const (
	speedTestURL     = "https://speed.cloudflare.com/__down?bytes=10000000"
	speedTestTimeout = 10 * time.Second
	maxSpeedTests    = 50
)

type IPResult struct {
	IP            string
	Latency       time.Duration
	DownloadSpeed float64
}

func testDownloadSpeed(ip string) (float64, error) {
	dialer := &net.Dialer{
		Timeout: speedTestTimeout,
	}
	
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, fmt.Sprintf("%s:443", ip))
		},
	}
	
	client := &http.Client{
		Transport: transport,
		Timeout:   speedTestTimeout,
	}
	
	start := time.Now()
	resp, err := client.Get(speedTestURL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("bad status: %d", resp.StatusCode)
	}
	
	buffer := make([]byte, 32*1024)
	totalBytes := 0
	
	for {
		n, err := resp.Body.Read(buffer)
		totalBytes += n
		
		if err != nil {
			if err == io.EOF {
				break
			}
			if time.Since(start) >= speedTestTimeout {
				break
			}
		}
		
		if time.Since(start) >= speedTestTimeout {
			break
		}
	}
	
	duration := time.Since(start).Seconds()
	if duration == 0 {
		return 0, fmt.Errorf("duration is zero")
	}
	
	speedMBps := float64(totalBytes) / duration / 1024 / 1024
	
	return speedMBps, nil
}

func ScanIPs(ips []string) []IPResult {
	cyan := color.New(color.FgCyan, color.Bold)
	cyan.Println("========================================")
	cyan.Println("      STEP 1: Latency Testing")
	cyan.Println("========================================")
	fmt.Println()
	
	pingResults := PingIPs(ips)
	
	if len(pingResults) == 0 {
		return nil
	}
	
	sort.Slice(pingResults, func(i, j int) bool {
		return pingResults[i].Latency < pingResults[j].Latency
	})
	
	testCount := len(pingResults)
	if testCount > maxSpeedTests {
		testCount = maxSpeedTests
	}
	
	cyan.Printf("Responsive IPs: %d\n", len(pingResults))
	cyan.Printf("Starting speed test for top %d IPs...\n\n", testCount)
	
	cyan.Println("========================================")
	cyan.Println("      STEP 2: Download Speed Test")
	cyan.Println("========================================")
	fmt.Println()
	
	var results []IPResult
	var mu sync.Mutex
	var wg sync.WaitGroup
	
	semaphore := make(chan struct{}, 5)
	
	completed := 0
	yellow := color.New(color.FgYellow)
	
	for i := 0; i < testCount; i++ {
		wg.Add(1)
		semaphore <- struct{}{}
		
		go func(pr PingResult) {
			defer wg.Done()
			defer func() { <-semaphore }()
			
			speed, err := testDownloadSpeed(pr.IP)
			
			if err == nil && speed > 0.1 {
				mu.Lock()
				results = append(results, IPResult{
					IP:            pr.IP,
					Latency:       pr.Latency,
					DownloadSpeed: speed,
				})
				completed++
				if completed%5 == 0 {
					yellow.Printf("Progress: %d/%d IPs speed tested\n", completed, testCount)
				}
				mu.Unlock()
			}
		}(pingResults[i])
	}
	
	wg.Wait()
	
	green := color.New(color.FgGreen)
	green.Printf("Speed test completed: %d clean IPs found\n\n", len(results))
	
	sort.Slice(results, func(i, j int) bool {
		if results[i].DownloadSpeed != results[j].DownloadSpeed {
			return results[i].DownloadSpeed > results[j].DownloadSpeed
		}
		return results[i].Latency < results[j].Latency
	})
	
	return results
}