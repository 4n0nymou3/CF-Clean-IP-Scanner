package scanner

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/VividCortex/ewma"
	"github.com/fatih/color"
)

const (
	bufferSize      = 1024
	downloadURL     = "https://speed.cloudflare.com/__down?bytes=52428800"
	downloadTimeout = 10 * time.Second
)

type IPResult struct {
	IP            *net.IPAddr
	Sended        int
	Received      int
	LossRate      float32
	Delay         int
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
		return (&net.Dialer{}).DialContext(ctx, network, fakeSourceAddr)
	}
}

func downloadHandler(ip *net.IPAddr, bytesUsed *int64) float64 {
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: getDialContext(ip),
		},
		Timeout: downloadTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return 0.0
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.80 Safari/537.36")

	response, err := client.Do(req)
	if err != nil {
		return 0.0
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return 0.0
	}

	timeStart := time.Now()
	timeEnd := timeStart.Add(downloadTimeout)
	contentLength := response.ContentLength
	buffer := make([]byte, bufferSize)

	var (
		contentRead     int64 = 0
		timeSlice             = downloadTimeout / 100
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
		n, err := response.Body.Read(buffer)
		if err != nil {
			if err != io.EOF {
				break
			} else if contentLength == -1 {
				break
			}
			lastSlice := timeStart.Add(timeSlice * time.Duration(timeCounter-1))
			e.Add(float64(contentRead-lastContentRead) / (float64(currentTime.Sub(lastSlice)) / float64(timeSlice)))
		}
		contentRead += int64(n)
	}

	atomic.AddInt64(bytesUsed, contentRead)
	return e.Value() / (downloadTimeout.Seconds() / 120)
}

func SpeedTest(stopCh <-chan struct{}, pingResults []PingResult, maxCount int, bytesUsed *int64) []IPResult {
	testCount := len(pingResults)
	if testCount > maxCount {
		testCount = maxCount
	}

	yellow := color.New(color.FgYellow)
	yellow.Printf("Testing download speed for top %d IPs (one at a time)...\n", testCount)
	yellow.Println("Press Ctrl+C at any time to stop and see results found so far.")
	fmt.Println()

	const barWidth = 50
	var results []IPResult
	foundCount := 0

	for i := 0; i < testCount; i++ {
		select {
		case <-stopCh:
			goto done
		default:
		}

		pr := pingResults[i]
		speed := downloadHandler(pr.IP, bytesUsed)

		if speed > 0 {
			foundCount++
			results = append(results, IPResult{
				IP:            pr.IP,
				Sended:        pr.Sended,
				Received:      pr.Received,
				LossRate:      pr.GetLossRate(),
				Delay:         int(pr.Delay.Milliseconds()),
				DownloadSpeed: speed,
			})
		}

		progress := float64(i+1) / float64(testCount)
		bar := buildProgressBar(int(progress*float64(barWidth)), barWidth)
		fmt.Printf("\r%s %3d%% (%d/%d) - Found: %d",
			bar, int(progress*100), i+1, testCount, foundCount)
	}

done:
	fmt.Println()
	fmt.Println()
	color.New(color.FgGreen).Printf("Speed test completed: %d clean IPs found\n\n", len(results))
	return results
}