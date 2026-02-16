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

	"github.com/VividCortex/ewma"
	"github.com/fatih/color"
)

const (
	bufferSize       = 1024
	speedTestTimeout = 20 * time.Second
	maxSpeedTests    = 500
)

var speedTestURLs = []string{
	"https://cloudflare.com/cdn-cgi/trace",
	"https://www.cloudflare.com/",
	"https://1.1.1.1/cdn-cgi/trace",
	"https://cloudflare-dns.com/",
	"https://cf.xiu2.xyz/url",
}

type IPResult struct {
	IP            *net.IPAddr
	Latency       time.Duration
	DownloadSpeed float64
}

func getDialContext(ip *net.IPAddr) func(ctx context.Context, network, address string) (net.Conn, error) {
	var fakeSourceAddr string
	if isIPv4(ip.String()) {
		fakeSourceAddr = fmt.Sprintf("%s:%d", ip.String(), port)
	} else {
		fakeSourceAddr = fmt.Sprintf("[%s]:%d", ip.String(), port)
	}
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{Timeout: 8 * time.Second}).DialContext(ctx, network, fakeSourceAddr)
	}
}

func testDownloadSpeedWithURL(ip *net.IPAddr, testURL string) float64 {
	client := &http.Client{
		Transport: &http.Transport{
			DialContext:           getDialContext(ip),
			DisableKeepAlives:     true,
			ResponseHeaderTimeout: 8 * time.Second,
			MaxIdleConns:          1,
		},
		Timeout: speedTestTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	
	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return 0.0
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	response, err := client.Do(req)
	if err != nil {
		return 0.0
	}
	defer response.Body.Close()
	
	if response.StatusCode != 200 && response.StatusCode != 301 && response.StatusCode != 302 {
		return 0.0
	}
	
	timeStart := time.Now()
	timeEnd := timeStart.Add(speedTestTimeout)

	contentLength := response.ContentLength
	buffer := make([]byte, bufferSize)

	var (
		contentRead     int64 = 0
		timeSlice             = speedTestTimeout / 100
		timeCounter           = 1
		lastContentRead int64 = 0
	)

	var nextTime = timeStart.Add(timeSlice * time.Duration(timeCounter))
	e := ewma.NewMovingAverage()

	for contentLength != contentRead {
		currentTime := time.Now()
		if currentTime.After(nextTime) {
			timeCounter++
			nextTime = timeStart.Add(timeSlice * time.Duration(timeCounter))
			e.Add(float64(contentRead - lastContentRead))
			lastContentRead = contentRead
		}
		
		if currentTime.After(timeEnd) {
			break
		}
		
		bufferRead, err := response.Body.Read(buffer)
		if err != nil {
			if err != io.EOF {
				break
			} else if contentLength == -1 {
				break
			}
			lastTimeSlice := timeStart.Add(timeSlice * time.Duration(timeCounter-1))
			e.Add(float64(contentRead-lastContentRead) / (float64(currentTime.Sub(lastTimeSlice)) / float64(timeSlice)))
		}
		contentRead += int64(bufferRead)
	}
	
	if contentRead < 256 {
		return 0.0
	}
	
	speed := e.Value() / (speedTestTimeout.Seconds() / 120)
	return speed
}

func testDownloadSpeed(ip *net.IPAddr) float64 {
	for _, url := range speedTestURLs {
		speed := testDownloadSpeedWithURL(ip, url)
		if speed > 0 {
			return speed
		}
	}
	return 0.0
}

func ScanIPs(ips []*net.IPAddr) []IPResult {
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
	foundCount := 0

	yellow.Println("Testing download speed...")
	yellow.Println("This may take a few minutes. Please wait...")
	fmt.Println()

	for i := 0; i < testCount; i++ {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(pr PingResult, index int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			speed := testDownloadSpeed(pr.IP)

			mu.Lock()
			completed++
			if completed%5 == 0 || completed == testCount {
				yellow.Printf("Progress: %d/%d IPs tested (found: %d)\n", completed, testCount, foundCount)
			}

			if speed > 0 {
				foundCount++
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