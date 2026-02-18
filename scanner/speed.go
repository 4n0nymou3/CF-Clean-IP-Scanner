package scanner

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/VividCortex/ewma"
	"github.com/fatih/color"
)

const (
	bufferSize       = 8192
	speedTestTimeout = 25 * time.Second
	dialTimeout      = 12 * time.Second
	headerTimeout    = 12 * time.Second
	tlsTimeout       = 12 * time.Second
	minValidBytes    = 256
)

var speedTestURLs = []string{
	"https://cloudflare.com/cdn-cgi/trace",
	"https://www.cloudflare.com/cdn-cgi/trace",
	"https://1.1.1.1/cdn-cgi/trace",
	"https://cloudflare-dns.com/",
	"https://cf.xiu2.xyz/url",
	"https://www.microsoft.com",
	"https://www.cloudflare.com/",
}

type IPResult struct {
	IP            *net.IPAddr
	Latency       int
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
		return (&net.Dialer{Timeout: dialTimeout}).DialContext(ctx, network, fakeSourceAddr)
	}
}

func testDownloadSpeedWithURL(ctx context.Context, ip *net.IPAddr, testURL string, bytesUsed *int64) float64 {
	client := &http.Client{
		Transport: &http.Transport{
			DialContext:            getDialContext(ip),
			DisableKeepAlives:      true,
			ResponseHeaderTimeout:  headerTimeout,
			TLSHandshakeTimeout:    tlsTimeout,
			MaxIdleConns:           1,
			MaxResponseHeaderBytes: 64 * 1024,
		},
		Timeout: speedTestTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", testURL, nil)
	if err != nil {
		return 0.0
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Connection", "close")

	response, err := client.Do(req)
	if err != nil {
		return 0.0
	}
	defer response.Body.Close()

	if response.StatusCode != 200 && response.StatusCode != 301 &&
		response.StatusCode != 302 && response.StatusCode != 204 {
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

	nextTime := timeStart.Add(timeSlice * time.Duration(timeCounter))
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
			elapsed := float64(currentTime.Sub(lastTimeSlice)) / float64(timeSlice)
			if elapsed > 0 {
				e.Add(float64(contentRead-lastContentRead) / elapsed)
			}
		}
		contentRead += int64(bufferRead)
	}

	atomic.AddInt64(bytesUsed, contentRead)

	if contentRead < minValidBytes {
		return 0.0
	}

	speed := e.Value() / (speedTestTimeout.Seconds() / 120)
	return speed
}

func testDownloadSpeed(ctx context.Context, ip *net.IPAddr, bytesUsed *int64) float64 {
	for _, url := range speedTestURLs {
		if ctx.Err() != nil {
			return 0.0
		}
		speed := testDownloadSpeedWithURL(ctx, ip, url, bytesUsed)
		if speed > 0 {
			return speed
		}
	}
	return 0.0
}

func SpeedTest(ctx context.Context, pingResults []PingResult, maxCount int, bytesUsed *int64) []IPResult {
	var results []IPResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	testCount := len(pingResults)
	if testCount > maxCount {
		testCount = maxCount
	}

	semaphore := make(chan struct{}, 3)

	yellow := color.New(color.FgYellow)
	completed := 0
	foundCount := 0

	yellow.Println("Testing download speed...")
	yellow.Println("This may take a few minutes. Please wait...")
	yellow.Println("Press Ctrl+C at any time to stop and see results found so far.")
	fmt.Println()

	const barWidth = 50

	for i := 0; i < testCount; i++ {
		select {
		case <-ctx.Done():
			goto waitAndReturn
		case semaphore <- struct{}{}:
		}

		wg.Add(1)
		go func(pr PingResult) {
			defer wg.Done()
			defer func() { <-semaphore }()

			speed := testDownloadSpeed(ctx, pr.IP, bytesUsed)

			mu.Lock()
			completed++
			if speed > 0 {
				foundCount++
				results = append(results, IPResult{
					IP:            pr.IP,
					Latency:       int(pr.Latency.Milliseconds()),
					DownloadSpeed: speed,
				})
			}
			progress := float64(completed) / float64(testCount)
			bar := buildProgressBar(int(progress*float64(barWidth)), barWidth)
			fmt.Printf("\r%s %3d%% (%d/%d) - Found: %d",
				bar, int(progress*100), completed, testCount, foundCount)
			mu.Unlock()
		}(pingResults[i])
	}

waitAndReturn:
	wg.Wait()

	fmt.Println()
	fmt.Println()
	color.New(color.FgGreen).Printf("Speed test completed: %d clean IPs found\n\n", len(results))

	return results
}