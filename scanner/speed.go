package scanner

import (
	"context"
	"crypto/tls"
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
	speedTestURL     = "https://cf.xiu2.xyz/url"
	speedTestTimeout = 10 * time.Second
	maxSpeedTests    = 50
)

type IPResult struct {
	IP            string
	Latency       time.Duration
	DownloadSpeed float64
}

func isIPv4(ip string) bool {
	return net.ParseIP(ip).To4() != nil
}

func getDialContext(ip string) func(ctx context.Context, network, address string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		dialer := &net.Dialer{
			Timeout:   speedTestTimeout,
			KeepAlive: 0,
		}
		var addr string
		if isIPv4(ip) {
			addr = fmt.Sprintf("%s:443", ip)
		} else {
			addr = fmt.Sprintf("[%s]:443", ip)
		}
		return dialer.DialContext(ctx, "tcp", addr)
	}
}

func testDownloadSpeed(ip string) (float64, error) {
	transport := &http.Transport{
		DialContext: getDialContext(ip),
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		DisableKeepAlives: true,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   speedTestTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	req, err := http.NewRequest("GET", speedTestURL, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	start := time.Now()
	resp, err := client.Get(speedTestURL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 301 && resp.StatusCode != 302 {
		return 0, fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	buffer := make([]byte, 32*1024)
	totalBytes := int64(0)

	for {
		if time.Since(start) >= speedTestTimeout {
			break
		}

		n, err := resp.Body.Read(buffer)
		totalBytes += int64(n)

		if err != nil {
			if err == io.EOF {
				break
			}
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

	semaphore := make(chan struct{}, 3)

	yellow := color.New(color.FgYellow)
	completed := 0

	yellow.Println("Testing speed... Please wait...")
	fmt.Println()

	for i := 0; i < testCount; i++ {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(pr PingResult, index int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			speed, err := testDownloadSpeed(pr.IP)

			mu.Lock()
			completed++
			if completed%5 == 0 || completed == testCount {
				yellow.Printf("Progress: %d/%d IPs tested\n", completed, testCount)
			}

			if err == nil && speed > 0.5 {
				results = append(results, IPResult{
					IP:            pr.IP,
					Latency:       pr.Latency,
					DownloadSpeed: speed,
				})
			}
			mu.Unlock()
		}(pingResults[i], i)
	}

	wg.Wait()

	fmt.Println()
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