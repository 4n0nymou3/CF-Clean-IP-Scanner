package scanner

import (
	"context"
	"crypto/tls"
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
	xrayWorkerCount     = 5
	xrayStartupDelay    = 800 * time.Millisecond
	xrayPortBase        = 11080
)

type xraySocksInfo struct {
	Address string
	Port    int
	User    string
	Pass    string
}

func createTempConfigWithIP(ip string, socksPort int) (string, *xraySocksInfo, error) {
	data, err := os.ReadFile("./config/xray_config.json")
	if err != nil {
		return "", nil, fmt.Errorf("cannot read config: %v", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", nil, fmt.Errorf("invalid JSON in config: %v", err)
	}

	inboundsRaw, ok := cfg["inbounds"]
	if !ok {
		return "", nil, fmt.Errorf("no 'inbounds' field in config")
	}
	inboundsSlice, ok := inboundsRaw.([]interface{})
	if !ok {
		return "", nil, fmt.Errorf("'inbounds' is not an array")
	}

	socksInfo := &xraySocksInfo{Address: "127.0.0.1", Port: socksPort}
	socksFound := false

	for i, in := range inboundsSlice {
		inMap, ok := in.(map[string]interface{})
		if !ok {
			continue
		}
		protocol, _ := inMap["protocol"].(string)
		if strings.ToLower(protocol) != "socks" {
			continue
		}
		inMap["port"] = float64(socksPort)
		if listen, ok := inMap["listen"].(string); ok && listen != "" {
			socksInfo.Address = listen
		} else {
			inMap["listen"] = "127.0.0.1"
		}
		if settings, ok := inMap["settings"].(map[string]interface{}); ok {
			if auth, _ := settings["auth"].(string); auth == "password" {
				if accounts, ok := settings["accounts"].([]interface{}); ok && len(accounts) > 0 {
					if acc, ok := accounts[0].(map[string]interface{}); ok {
						socksInfo.User, _ = acc["user"].(string)
						socksInfo.Pass, _ = acc["pass"].(string)
					}
				}
			}
		}
		inboundsSlice[i] = inMap
		socksFound = true
		break
	}

	if !socksFound {
		return "", nil, fmt.Errorf("no SOCKS inbound found in config")
	}
	cfg["inbounds"] = inboundsSlice

	outboundsRaw, ok := cfg["outbounds"]
	if !ok {
		return "", nil, fmt.Errorf("no 'outbounds' field in config")
	}
	outboundsSlice, ok := outboundsRaw.([]interface{})
	if !ok {
		return "", nil, fmt.Errorf("'outbounds' is not an array")
	}

	ipUpdated := false
	for i, out := range outboundsSlice {
		outMap, ok := out.(map[string]interface{})
		if !ok {
			continue
		}
		protocol, _ := outMap["protocol"].(string)
		protocol = strings.ToLower(protocol)

		settings, ok := outMap["settings"].(map[string]interface{})
		if !ok {
			continue
		}

		switch protocol {
		case "vless", "vmess":
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
			vnextSlice[0] = server
			settings["vnext"] = vnextSlice
			outMap["settings"] = settings
			outboundsSlice[i] = outMap
			ipUpdated = true

		case "trojan", "shadowsocks":
			serversRaw, ok := settings["servers"]
			if !ok {
				continue
			}
			serversSlice, ok := serversRaw.([]interface{})
			if !ok || len(serversSlice) == 0 {
				continue
			}
			server, ok := serversSlice[0].(map[string]interface{})
			if !ok {
				continue
			}
			server["address"] = ip
			server["port"] = float64(xrayPort)
			serversSlice[0] = server
			settings["servers"] = serversSlice
			outMap["settings"] = settings
			outboundsSlice[i] = outMap
			ipUpdated = true
		}

		if ipUpdated {
			break
		}
	}

	if !ipUpdated {
		return "", nil, fmt.Errorf("no supported outbound found (vless/vmess/trojan/shadowsocks)")
	}
	cfg["outbounds"] = outboundsSlice

	cleanCfg := make(map[string]interface{})
	for _, field := range []string{"log", "inbounds", "outbounds", "routing", "dns", "policy", "stats", "api", "transport"} {
		if v, exists := cfg[field]; exists {
			cleanCfg[field] = v
		}
	}

	newData, err := json.MarshalIndent(cleanCfg, "", "  ")
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal config: %v", err)
	}

	tempFile, err := os.CreateTemp("", "xray_cfg_*.json")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp file: %v", err)
	}
	if _, err := tempFile.Write(newData); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return "", nil, fmt.Errorf("failed to write temp config: %v", err)
	}
	tempFile.Close()

	return tempFile.Name(), socksInfo, nil
}

func createSocksDialer(socksInfo *xraySocksInfo) (proxy.Dialer, error) {
	addr := fmt.Sprintf("%s:%d", socksInfo.Address, socksInfo.Port)
	if socksInfo.User != "" && socksInfo.Pass != "" {
		auth := proxy.Auth{User: socksInfo.User, Password: socksInfo.Pass}
		return proxy.SOCKS5("tcp", addr, &auth, proxy.Direct)
	}
	return proxy.SOCKS5("tcp", addr, nil, proxy.Direct)
}

func testIPViaXray(ip *net.IPAddr, socksPort int) (recv int, totalDelay time.Duration) {
	configPath, socksInfo, err := createTempConfigWithIP(ip.String(), socksPort)
	if err != nil {
		return
	}
	defer os.Remove(configPath)

	cmd := exec.Command("./xray/xray", "run", "-c", configPath)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
	}()

	time.Sleep(xrayStartupDelay)

	dialer, err := createSocksDialer(socksInfo)
	if err != nil {
		return
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			},
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	for i := 0; i < defaultPingTimes; i++ {
		start := time.Now()
		resp, err := httpClient.Get("https://cp.cloudflare.com/generate_204")
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode == 200 || resp.StatusCode == 204 {
				recv++
				totalDelay += time.Since(start)
			}
		}
		if i < defaultPingTimes-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}
	return
}

func PingIPsViaXray(stopCh <-chan struct{}, ips []*net.IPAddr) []PingResult {
	if _, err := os.Stat("./xray/xray"); os.IsNotExist(err) {
		color.New(color.FgRed).Println("ERROR: Xray binary not found at ./xray/xray")
		return nil
	}
	if _, err := os.Stat("./config/xray_config.json"); os.IsNotExist(err) {
		color.New(color.FgRed).Println("ERROR: Xray config not found at ./config/xray_config.json")
		return nil
	}

	var results []PingResult
	var mu sync.Mutex
	total := len(ips)

	color.New(color.FgCyan).Printf("Start latency test (Xray mode - %d attempts per IP, %d workers)\n", defaultPingTimes, xrayWorkerCount)
	bar := newBar(total, "Available:", "")

	ipChan := make(chan *net.IPAddr, total)
	for _, ip := range ips {
		select {
		case <-stopCh:
		default:
			ipChan <- ip
		}
	}
	close(ipChan)

	var wg sync.WaitGroup
	for w := 0; w < xrayWorkerCount; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			socksPort := xrayPortBase + workerID

			for ipAddr := range ipChan {
				select {
				case <-stopCh:
					return
				default:
				}

				recv, totalDelay := testIPViaXray(ipAddr, socksPort)

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
			}
		}(w)
	}

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
	configPath, socksInfo, err := createTempConfigWithIP(ip.String(), socksPort)
	if err != nil {
		return 0.0
	}
	defer os.Remove(configPath)

	cmd := exec.Command("./xray/xray", "run", "-c", configPath)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return 0.0
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
	}()

	time.Sleep(xrayStartupDelay)

	dialer, err := createSocksDialer(socksInfo)
	if err != nil {
		return 0.0
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			},
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: xrayDownloadTimeout,
	}

	req, err := http.NewRequest("GET", xrayDownloadURL, nil)
	if err != nil {
		return 0.0
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.80 Safari/537.36")

	response, err := httpClient.Do(req)
	if err != nil {
		return 0.0
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
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
	speedPort := xrayPortBase + xrayWorkerCount

	for i := 0; i < testNum; i++ {
		select {
		case <-stopCh:
			goto done
		default:
		}

		pr := pingResults[i]
		speedMBps := downloadSpeedViaXray(pr.IP, speedPort)

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