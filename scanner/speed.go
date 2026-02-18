package scanner

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"sync/atomic"
	"time"

	"github.com/VividCortex/ewma"
	"github.com/fatih/color"
)

const (
	bufferSize      = 1024
	testURL         = "https://cf.xiu2.xyz/url"
	downloadTimeout = 10 * time.Second
	defaultTestNum  = 10
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
			if req.Header.Get("Referer") == testURL {
				req.Header.Del("Referer")
			}
			return nil
		},
	}

	req, err := http.NewRequest("GET", testURL, nil)
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

func SpeedTest(stopCh <-chan struct{}, pingResults []PingResult, bytesUsed *int64) []IPResult {
	testCount := defaultTestNum
	testNum := len(pingResults)
	if testNum < testCount {
		testCount = testNum
	}

	yellow := color.New(color.FgYellow)
	yellow.Printf("Start download speed test (Number: %d, Queue: %d)\n", testCount, testNum)
	yellow.Println("Press Ctrl+C at any time to stop and see results found so far.")
	fmt.Println()

	const barWidth = 50
	var results []IPResult
	foundCount := 0
	tested := 0

	for i := 0; i < testNum; i++ {
		select {
		case <-stopCh:
			goto done
		default:
		}

		pr := pingResults[i]
		speed := downloadHandler(pr.IP, bytesUsed)
		tested++

		if speed >= 0.0 && speed > 0 {
			foundCount++
			results = append(results, IPResult{
				IP:            pr.IP,
				Sended:        pr.Sended,
				Received:      pr.Received,
				LossRate:      pr.GetLossRate(),
				Delay:         int(pr.Delay.Milliseconds()),
				DownloadSpeed: speed,
			})
			if foundCount == testCount {
				tested = i + 1
				goto done
			}
		}

		progress := float64(foundCount) / float64(testCount)
		bar := buildProgressBar(int(progress*float64(barWidth)), barWidth)
		fmt.Printf("\r%s  %d/%d  tested: %d",
			bar, foundCount, testCount, tested)
	}

done:
	fmt.Println()
	fmt.Println()

	if len(results) > 0 {
		sort.Slice(results, func(i, j int) bool {
			return results[i].DownloadSpeed > results[j].DownloadSpeed
		})
	}

	color.New(color.FgGreen).Printf("Speed test completed: %d clean IPs found (tested %d IPs)\n\n", len(results), tested)
	return results
}