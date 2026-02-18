package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/4n0nymou3/CF-Clean-IP-Scanner/config"
	"github.com/4n0nymou3/CF-Clean-IP-Scanner/scanner"
	"github.com/4n0nymou3/CF-Clean-IP-Scanner/utils"
)

const version = "1.3.0"

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
		yellow := color.New(color.FgYellow, color.Bold)
		yellow.Println("         Scan stopped by user")
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

func askSpeedTestCount(ctx context.Context, max int) (int, bool) {
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed, color.Bold)

	inputCh := make(chan string)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			text, err := reader.ReadString('\n')
			if err != nil {
				close(inputCh)
				return
			}
			select {
			case inputCh <- strings.TrimSpace(text):
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		yellow.Printf("How many of these %d IPs do you want to speed test?\n", max)
		yellow.Printf("Enter a number between 10 and %d: ", max)

		select {
		case <-ctx.Done():
			fmt.Println()
			return 0, true
		case text, ok := <-inputCh:
			if !ok {
				return 0, true
			}
			n, err := strconv.Atoi(text)
			if err != nil || n < 10 || n > max {
				fmt.Println()
				red.Printf("Invalid input! Please enter a whole number between 10 and %d.\n", max)
				fmt.Println()
				continue
			}
			return n, false
		}
	}
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
	yellow.Println("2-Stage Test: Latency (ping < 1000ms) + Download Speed")
	yellow.Println("Sorted by: Lowest Latency")
	fmt.Println()

	time.Sleep(1 * time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		signal.Reset(os.Interrupt)
		fmt.Println()
		fmt.Println()
		color.New(color.FgYellow, color.Bold).Println("Interrupt received. Stopping scan and collecting results...")
		cancel()
	}()

	startTime := time.Now()
	var bytesUsed int64

	ipRanges := config.GetCloudflareRanges()
	cyan.Printf("IP Ranges: %d\n", len(ipRanges))
	cyan.Println("Generating IPs from ranges...")
	fmt.Println()

	ips := scanner.GenerateIPs(ipRanges, 0)
	cyan.Printf("Total IPs to scan: %d\n\n", len(ips))

	cyanBold := color.New(color.FgCyan, color.Bold)
	cyanBold.Println("========================================")
	cyanBold.Println("      STEP 1: Latency Testing")
	cyanBold.Println("========================================")
	fmt.Println()

	pingResults := scanner.PingIPs(ctx, ips, &bytesUsed)

	if ctx.Err() != nil {
		elapsed := time.Since(startTime)
		color.New(color.FgYellow).Println("Scan was stopped during latency test. No clean IPs to show yet.")
		printScanStats(elapsed, atomic.LoadInt64(&bytesUsed), true)
		return
	}

	if len(pingResults) == 0 {
		red := color.New(color.FgRed, color.Bold)
		red.Println("No responsive IPs found!")
		fmt.Println()
		yellow.Println("Possible reasons:")
		yellow.Println("  - All IPs have high latency (> 1000ms)")
		yellow.Println("  - Network connection issues")
		yellow.Println("  - Try again at a different time (night hours often better)")
		elapsed := time.Since(startTime)
		printScanStats(elapsed, atomic.LoadInt64(&bytesUsed), false)
		return
	}

	sort.Slice(pingResults, func(i, j int) bool {
		return pingResults[i].Latency < pingResults[j].Latency
	})

	cyan.Printf("Responsive IPs: %d\n\n", len(pingResults))

	cyanBold.Println("========================================")
	cyanBold.Println("      STEP 2: Download Speed Test")
	cyanBold.Println("========================================")
	fmt.Println()

	count, cancelled := askSpeedTestCount(ctx, len(pingResults))
	if cancelled {
		elapsed := time.Since(startTime)
		color.New(color.FgYellow).Println("Scan was stopped before speed test. No clean IPs to show yet.")
		printScanStats(elapsed, atomic.LoadInt64(&bytesUsed), true)
		return
	}

	color.New(color.FgGreen).Printf("Starting speed test for %d IPs...\n\n", count)

	results := scanner.SpeedTest(ctx, pingResults, count, &bytesUsed)

	elapsed := time.Since(startTime)
	interrupted := ctx.Err() != nil

	if len(results) == 0 {
		red := color.New(color.FgRed, color.Bold)
		if interrupted {
			red.Println("No clean IPs found before scan was stopped.")
		} else {
			red.Println("No clean IPs found!")
			fmt.Println()
			yellow.Println("Possible reasons:")
			yellow.Println("  - No IPs could complete a download successfully")
			yellow.Println("  - Network issues or heavy filtering")
			yellow.Println("  - Try again at a different time")
		}
		printScanStats(elapsed, atomic.LoadInt64(&bytesUsed), interrupted)
		return
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Latency < results[j].Latency
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

	printScanStats(elapsed, atomic.LoadInt64(&bytesUsed), interrupted)
}