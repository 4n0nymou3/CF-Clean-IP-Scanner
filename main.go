package main

import (
	"fmt"
	"os"
	"os/signal"
	"sort"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/4n0nymou3/CF-Clean-IP-Scanner/config"
	"github.com/4n0nymou3/CF-Clean-IP-Scanner/scanner"
	"github.com/4n0nymou3/CF-Clean-IP-Scanner/utils"
)

const version = "1.4.0"

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func formatBytes(b int64) string {
	if b < 1024 {
		return fmt.Sprintf("%d B", b)
	} else if b < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(b)/1024)
	}
	return fmt.Sprintf("%.2f MB", float64(b)/1024/1024)
}

func printScanStats(elapsed time.Duration, bytesUsed int64, interrupted bool) {
	fmt.Println()
	cyan := color.New(color.FgCyan, color.Bold)
	cyan.Println("========================================")
	if interrupted {
		color.New(color.FgYellow, color.Bold).Println("         Scan stopped by user")
	} else {
		cyan.Println("      Scan completed successfully!")
	}
	cyan.Println("========================================")
	fmt.Println()
	info := color.New(color.FgCyan)
	info.Printf("  Scan Duration : %s\n", formatDuration(elapsed))
	info.Printf("  Data Used     : %s\n", formatBytes(bytesUsed))
	fmt.Println()
}

func main() {
	utils.PrintHeader()
	utils.PrintDesigner()

	cyan := color.New(color.FgCyan)
	cyan.Printf("Version: %s\n", version)
	cyan.Println("Starting Cloudflare Clean IP Scanner...")
	fmt.Println()

	yellow := color.New(color.FgYellow)
	yellow.Println("Optimized for Iran network conditions")
	yellow.Println("Stage 1 : TCP ping x4 per IP  →  loss rate + avg latency")
	yellow.Println("Stage 2 : Download speed test  →  stops after finding 10 clean IPs")
	yellow.Println("Sorted  : loss rate → latency → download speed")
	fmt.Println()

	time.Sleep(1 * time.Second)

	stopCh := make(chan struct{})
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		signal.Reset(os.Interrupt)
		fmt.Println()
		fmt.Println()
		color.New(color.FgYellow, color.Bold).Println("Interrupt received. Stopping scan and collecting results...")
		close(stopCh)
	}()

	startTime := time.Now()
	var downloadBytes int64

	ipRanges := config.GetCloudflareRanges()
	cyan.Printf("IP Ranges: %d\n", len(ipRanges))
	cyan.Println("Generating IPs from ranges...")
	fmt.Println()

	ips := scanner.GenerateIPs(ipRanges)
	cyan.Printf("Total IPs to test: %d\n\n", len(ips))

	cyanBold := color.New(color.FgCyan, color.Bold)
	cyanBold.Println("========================================")
	cyanBold.Println("      STEP 1: Latency Testing")
	cyanBold.Println("========================================")
	fmt.Println()

	pingResults := scanner.PingIPs(stopCh, ips)

	select {
	case <-stopCh:
		elapsed := time.Since(startTime)
		pingBytes := int64(len(ips)) * 4 * 80
		color.New(color.FgYellow).Println("Scan stopped during latency test. No clean IPs to show yet.")
		printScanStats(elapsed, pingBytes, true)
		return
	default:
	}

	if len(pingResults) == 0 {
		red := color.New(color.FgRed, color.Bold)
		red.Println("No responsive IPs found!")
		fmt.Println()
		yellow.Println("Try running again. Network conditions may vary.")
		elapsed := time.Since(startTime)
		pingBytes := int64(len(ips)) * 4 * 80
		printScanStats(elapsed, pingBytes, false)
		return
	}

	cyan.Printf("Responsive IPs: %d\n\n", len(pingResults))

	cyanBold.Println("========================================")
	cyanBold.Println("      STEP 2: Download Speed Test")
	cyanBold.Println("========================================")
	fmt.Println()

	results := scanner.SpeedTest(stopCh, pingResults, &downloadBytes)

	elapsed := time.Since(startTime)
	pingBytes := int64(len(ips)) * 4 * 80
	totalBytes := pingBytes + atomic.LoadInt64(&downloadBytes)

	interrupted := false
	select {
	case <-stopCh:
		interrupted = true
	default:
	}

	if len(results) == 0 {
		red := color.New(color.FgRed, color.Bold)
		if interrupted {
			red.Println("No clean IPs found before scan was stopped.")
		} else {
			red.Println("No clean IPs found.")
			fmt.Println()
			yellow.Println("Try running again at a different time.")
		}
		printScanStats(elapsed, totalBytes, interrupted)
		return
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].DownloadSpeed > results[j].DownloadSpeed
	})

	topResults := results
	if len(results) > 10 {
		topResults = results[:10]
	}

	if interrupted {
		color.New(color.FgYellow, color.Bold).Printf(
			"\nShowing %d clean IP(s) found before scan was stopped:\n", len(results))
	}

	utils.PrintResults(topResults)

	if err := utils.SaveResults(results, "clean_ips.txt"); err != nil {
		color.New(color.FgRed).Printf("Error saving file: %v\n", err)
	} else {
		color.New(color.FgGreen).Println("Results saved to clean_ips.txt")
		color.New(color.FgGreen).Printf("Total clean IPs found: %d\n", len(results))
	}

	printScanStats(elapsed, totalBytes, interrupted)
}