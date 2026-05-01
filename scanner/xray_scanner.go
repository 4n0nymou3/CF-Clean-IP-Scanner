package scanner

import (
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
	socksLocalPortStart = 11080
)

type xraySocksInfo struct {
	Address string
	Port    int
	User    string
	Pass    string
}

func findSocksInbound(inbounds []interface{}) (*xraySocksInfo, error) {
	for _, in := range inbounds {
		inMap, ok := in.(map[string]interface{})
		if !ok {
			continue
		}
		protocol, _ := inMap["protocol"].(string)
		if protocol != "socks" {
			continue
		}
		listen, _ := inMap["listen"].(string)
		if listen == "" {
			listen = "127.0.0.1"
		}
		portFloat, ok := inMap["port"].(float64)
		if !ok {
			continue
		}
		port := int(portFloat)
		info := &xraySocksInfo{Address: listen, Port: port, User: "", Pass: ""}
		settings, ok := inMap["settings"].(map[string]interface{})
		if !ok {
			return info, nil
		}
		auth, _ := settings["auth"].(string)
		if auth == "password" {
			accounts, _ := settings["accounts"].([]interface{})
			if len(accounts) > 0 {
				acc, _ := accounts[0].(map[string]interface{})
				user, _ := acc["user"].(string)
				pass, _ := acc["pass"].(string)
				info.User = user
				info.Pass = pass
			}
		}
		return info, nil
	}
	return nil, fmt.Errorf("no SOCKS inbound found")
}

func replaceIPInXrayConfig(ip string, socksPort int) (configPath string, socksInfo *xraySocksInfo, err error) {
	originalPath := "./config/xray_config.json"
	data, err := os.ReadFile(originalPath)
	if err != nil {
		return "", nil, fmt.Errorf("cannot read config: %v", err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", nil, fmt.Errorf("invalid JSON: %v", err)
	}
	inboundsRaw, ok := cfg["inbounds"]
	if !ok {
		return "", nil, fmt.Errorf("no 'inbounds' field")
	}
	inboundsSlice, ok := inboundsRaw.([]interface{})
	if !ok {
		return "", nil, fmt.Errorf("'inbounds' is not an array")
	}
	socksInfo, err = findSocksInbound(inboundsSlice)
	if err != nil {
		return "", nil, err
	}
	socksInfo.Port = socksPort
	outboundsRaw, ok := cfg["outbounds"]
	if !ok {
		return "", nil, fmt.Errorf("no 'outbounds' field")
	}
	outboundsSlice, ok := outboundsRaw.([]interface{})
	if !ok {
		return "", nil, fmt.Errorf("'outbounds' is not an array")
	}
	found := false
	for i, out := range outboundsSlice {
		outMap, ok := out.(map[string]interface{})
		if !ok {
			continue
		}
		protocol, _ := outMap["protocol"].(string)
		protocol = strings.ToLower(protocol)
		if protocol != "vless" && protocol != "trojan" && protocol != "vmess" {
			continue
		}
		settings, ok := outMap["settings"].(map[string]interface{})
		if !ok {
			continue
		}
		vnextRaw, ok := settings["vnext"]
		if !ok {
			continue
		}
		vnextSlice, ok := vnextRaw.([]interface{})
		if !ok || len(vnextSlice) == 0 {
			continue
		}
		server, ok := vnextSlice[0].(map[string]interface{})
		if !ok {
			continue
		}
		server["address"] = ip
		server["port"] = float64(xrayPort)
		settings["vnext"] = vnextSlice
		outMap["settings"] = settings
		outboundsSlice[i] = outMap
		found = true
		break
	}
	if !found {
		return "", nil, fmt.Errorf("no suitable outbound (vless/trojan/vmess) with vnext found")
	}
	cfg["outbounds"] = outboundsSlice
	newData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", nil, err
	}
	tempFile := fmt.Sprintf("/tmp/xray_config_%d.json", time.Now().UnixNano())
	if err := os.WriteFile(tempFile, newData, 0644); err != nil {
		return "", nil, err
	}
	return tempFile, socksInfo, nil
}

func createSocksDialer(socksInfo *xraySocksInfo) (proxy.Dialer, error) {
	addr := fmt.Sprintf("%s:%d", socksInfo.Address, socksInfo.Port)
	if socksInfo.User != "" && socksInfo.Pass != "" {
		auth := proxy.Auth{User: socksInfo.User, Password: socksInfo.Pass}
		return proxy.SOCKS5("tcp", addr, &auth, proxy.Direct)
	}
	return proxy.SOCKS5("tcp", addr, nil, proxy.Direct)
}

func testSingleViaXray(ip *net.IPAddr, socksPort int) (bool, time.Duration) {
	configFile, socksInfo, err := replaceIPInXrayConfig(ip.String(), socksPort)
	if err != nil {
		return false, 0
	}
	defer os.Remove(configFile)
	ctx, cancel := context.WithTimeout(context.Background(), xrayDownloadTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "./xray/xray", "run", "-c", configFile)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return false, 0
	}
	time.Sleep(600 * time.Millisecond)
	dialer, err := createSocksDialer(socksInfo)
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
	resp, err := httpClient.Get("https://www.gstatic.com/generate_204")
	if err != nil {
		cmd.Process.Kill()
		return false, 0
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		cmd.Process.Kill()
		return false, 0
	}
	io.Copy(io.Discard, resp.Body)
	elapsed := time.Since(start)
	cmd.Process.Kill()
	return true, elapsed
}

func checkConnectionViaXray(ip *net.IPAddr, socksPort int) (recv int, totalDelay time.Duration) {
	for i := 0; i < defaultPingTimes; i++ {
		if ok, d := testSingleViaXray(ip, socksPort); ok {
			recv++
			totalDelay += d
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
	return
}

func PingIPsViaXray(stopCh <-chan struct{}, ips []*net.IPAddr) []PingResult {
	var results []PingResult
	var mu sync.Mutex
	var wg sync.WaitGroup
	control := make(chan struct{}, maxRoutines)
	total := len(ips)
	cyan := color.New(color.FgCyan)
	cyan.Printf("Start latency test (Xray mode - %d attempts per IP)\n", defaultPingTimes)
	var socksPortCounter int = socksLocalPortStart
	portMu := sync.Mutex{}
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
			portMu.Lock()
			socksPort := socksPortCounter
			socksPortCounter++
			portMu.Unlock()
			recv, totalDelay := checkConnectionViaXray(ipAddr, socksPort)
			mu.Lock()
			nowAble := len(results)
			if recv > 0 {
				nowAble++
				avgDelay := totalDelay / time.Duration(recv)
				results = append(results, PingResult{
					IP:       ipAddr,
					Sended:   defaultPingTimes,
					Received: recv,
					Delay:    avgDelay,
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

func downloadSpeedViaXray(ip *net.IPAddr, socksPort int) float64 {
	configFile, socksInfo, err := replaceIPInXrayConfig(ip.String(), socksPort)
	if err != nil {
		return 0.0
	}
	defer os.Remove(configFile)
	ctx, cancel := context.WithTimeout(context.Background(), xrayDownloadTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "./xray/xray", "run", "-c", configFile)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return 0.0
	}
	time.Sleep(600 * time.Millisecond)
	dialer, err := createSocksDialer(socksInfo)
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
			}
			if response.ContentLength == -1 {
				break
			}
			lastSlice := timeStart.Add(timeSlice * time.Duration(timeCounter-1))
			if currentTime.After(lastSlice) {
				ratio := float64(currentTime.Sub(lastSlice)) / float64(timeSlice)
				if ratio > 0 {
					e.Add(float64(contentRead-lastContentRead) / ratio)
				}
			}
			break
		}
		contentRead += int64(n)
	}
	cmd.Process.Kill()
	avgBytesPerSec := e.Value() * 100 / xrayDownloadTimeout.Seconds()
	return avgBytesPerSec / (1024 * 1024)
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
	portCounter := xrayTestNum + 10000
	for i := 0; i < testNum; i++ {
		select {
		case <-stopCh:
			goto done
		default:
		}
		pr := pingResults[i]
		speedMBps := downloadSpeedViaXray(pr.IP, portCounter)
		portCounter++
		if speedMBps >= xrayMinSpeed {
			bar.grow(1, "")
			results = append(results, IPResult{
				IP:            pr.IP,
				Sended:        pr.Sended,
				Received:      pr.Received,
				LossRate:      pr.GetLossRate(),
				Delay:         int(pr.Delay.Milliseconds()),
				DownloadSpeed: speedMBps * 1024 * 1024,
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