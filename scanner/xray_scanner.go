package scanner

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/VividCortex/ewma"
	"github.com/fatih/color"
	"golang.org/x/net/proxy"
)

const (
	xrayBufferSize      = 1024
	xrayDownloadURL     = "https://speed.cloudflare.com/__down?bytes=52428800"
	xrayDownloadTimeout = 10 * time.Second
	xrayTestNum         = 10
	xrayMinSpeed        = 0.0
	xrayPort            = 443
	xraySocksPort       = 11080
)

type xrayConfig struct {
	Inbounds  []interface{} `json:"inbounds"`
	Outbounds []interface{} `json:"outbounds"`
}

func replaceIPInXrayConfig(ip string) (string, error) {
	configPath := "./config/xray_config.json"
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", err
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", err
	}
	outbounds, ok := cfg["outbounds"].([]interface{})
	if !ok || len(outbounds) == 0 {
		return "", fmt.Errorf("no outbounds found")
	}
	outbound := outbounds[0].(map[string]interface{})
	settings, ok := outbound["settings"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("outbound settings not found")
	}
	vnext, ok := settings["vnext"].([]interface{})
	if !ok || len(vnext) == 0 {
		return "", fmt.Errorf("vnext not found")
	}
	server := vnext[0].(map[string]interface{})
	server["address"] = ip
	server["port"] = float64(xrayPort)
	settings["vnext"] = vnext
	outbound["settings"] = settings
	cfg["outbounds"] = outbounds
	newData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}
	tempFile := fmt.Sprintf("/tmp/xray_config_%d.json", time.Now().UnixNano())
	if err := os.WriteFile(tempFile, newData, 0644); err != nil {
		return "", err
	}
	return tempFile, nil
}

func testViaXray(ip *net.IPAddr) (bool, time.Duration) {
	configFile, err := replaceIPInXrayConfig(ip.String())
	if err != nil {
		return false, 0
	}
	defer os.Remove(configFile)

	xrayBin := "./xray/xray"
	ctx, cancel := context.WithTimeout(context.Background(), xrayDownloadTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, xrayBin, "run", "-c", configFile)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return false, 0
	}
	time.Sleep(500 * time.Millisecond)

	dialer, err := proxy.SOCKS5("tcp", "127.0.0.1:11080", nil, proxy.Direct)
	if err != nil {
		cmd.Process.Kill()
		return false, 0
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			},
		},
		Timeout: 3 * time.Second,
	}

	start := time.Now()
	resp, err := httpClient.Get("https://speed.cloudflare.com/__up?bytes=1000")
	if err != nil {
		cmd.Process.Kill()
		return false, 0
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	elapsed := time.Since(start)

	cmd.Process.Kill()
	return true, elapsed
}

func PingIPsViaXray(stopCh <-chan struct{}, ips []*net.IPAddr) []PingResult {
	var results []PingResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	control := make(chan struct{}, maxRoutines)
	total := len(ips)

	cyan := color.New(color.FgCyan)
	cyan.Printf("Start latency test (Xray mode, Port: %d, Socks: %d)\n", xrayPort, xraySocksPort)

	bar := newBar(total, "Available:", "")

	for _, ip := range ips {
		select {
		case <-stopCh:
			goto done
		case control <- struct{}{}:
		}

		wg.Add(1)
		go func(ipAddr *net.IPAddr) {
			defer wg.Done()
			defer func() { <-control }()

			ok, delay := testViaXray(ipAddr)
			mu.Lock()
			nowAble := len(results)
			if ok {
				nowAble++
				results = append(results, PingResult{
					IP:       ipAddr,
					Sended:   defaultPingTimes,
					Received: 1,
					Delay:    delay,
				})
			}
			bar.grow(1, strconv.Itoa(nowAble))
			mu.Unlock()
		}(ip)
	}

done:
	wg.Wait()
	bar.done()

	sort.Slice(results, func(i, j int) bool {
		li, lj := results[i].GetLossRate(), results[j].GetLossRate()
		if li != lj {
			return li < lj
		}
		return results[i].Delay < results[j].Delay
	})

	fmt.Println()
	color.New(color.FgGreen).Printf("Latency test completed (Xray): %d responsive IPs found\n\n", len(results))

	return results
}

func downloadViaXray(ip *net.IPAddr) float64 {
	configFile, err := replaceIPInXrayConfig(ip.String())
	if err != nil {
		return 0.0
	}
	defer os.Remove(configFile)

	xrayBin := "./xray/xray"
	ctx, cancel := context.WithTimeout(context.Background(), xrayDownloadTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, xrayBin, "run", "-c", configFile)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return 0.0
	}
	time.Sleep(500 * time.Millisecond)

	dialer, err := proxy.SOCKS5("tcp", "127.0.0.1:11080", nil, proxy.Direct)
	if err != nil {
		cmd.Process.Kill()
		return 0.0
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			},
		},
		Timeout: xrayDownloadTimeout,
	}

	req, err := http.NewRequest("GET", xrayDownloadURL, nil)
	if err != nil {
		cmd.Process.Kill()
		return 0.0
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.80 Safari/537.36")

	response, err := httpClient.Do(req)
	if err != nil {
		cmd.Process.Kill()
		return 0.0
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		cmd.Process.Kill()
		return 0.0
	}

	timeStart := time.Now()
	timeEnd := timeStart.Add(xrayDownloadTimeout)
	buffer := make([]byte, xrayBufferSize)

	var contentRead int64 = 0
	var lastContentRead int64 = 0
	timeSlice := xrayDownloadTimeout / 100
	timeCounter := 1
	nextTime := timeStart.Add(timeSlice * time.Duration(timeCounter))
	e := ewma.NewMovingAverage()

	for {
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
			} else if response.ContentLength == -1 {
				break
			}
			lastSlice := timeStart.Add(timeSlice * time.Duration(timeCounter-1))
			e.Add(float64(contentRead-lastContentRead) / (float64(currentTime.Sub(lastSlice)) / float64(timeSlice)))
			break
		}
		contentRead += int64(n)
	}

	cmd.Process.Kill()
	return e.Value() / (xrayDownloadTimeout.Seconds() / 120)
}

func SpeedTestViaXray(stopCh <-chan struct{}, pingResults []PingResult) []IPResult {
	testCount := xrayTestNum
	testNum := testCount
	if len(pingResults) < testCount {
		testNum = len(pingResults)
		testCount = testNum
	}

	barPadding := "     "
	for i := 0; i < len(strconv.Itoa(len(pingResults))); i++ {
		barPadding += " "
	}

	color.New(color.FgCyan).Printf("Start download speed test (Xray mode, Minimum speed: %.2f MB/s, Number: %d, Queue: %d)\n", xrayMinSpeed, testCount, testNum)

	bar := newBar(testCount, barPadding, "")

	var results []IPResult

	for i := 0; i < testNum; i++ {
		select {
		case <-stopCh:
			goto done
		default:
		}

		pr := pingResults[i]
		speed := downloadViaXray(pr.IP)

		if speed >= xrayMinSpeed {
			bar.grow(1, "")
			results = append(results, IPResult{
				IP:            pr.IP,
				Sended:        pr.Sended,
				Received:      pr.Received,
				LossRate:      pr.GetLossRate(),
				Delay:         int(pr.Delay.Milliseconds()),
				DownloadSpeed: speed,
			})
			if len(results) == testCount {
				break
			}
		}
	}

done:
	bar.done()

	if len(results) > 0 {
		sort.Slice(results, func(i, j int) bool {
			return results[i].DownloadSpeed > results[j].DownloadSpeed
		})
	}

	fmt.Println()
	color.New(color.FgGreen).Printf("Speed test completed (Xray): %d clean IPs found\n\n", len(results))
	return results
}