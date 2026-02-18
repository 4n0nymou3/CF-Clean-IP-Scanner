package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/4n0nymou3/CF-Clean-IP-Scanner/config"
	"github.com/4n0nymou3/CF-Clean-IP-Scanner/scanner"
	"github.com/4n0nymou3/CF-Clean-IP-Scanner/utils"
)

const version = "1.3.0"

var (
	interrupted  bool
	interruptMux sync.Mutex
	dataUsage    int64
)

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		interruptMux.Lock()
		interrupted = true
		interruptMux.Unlock()
		yellow := color.New(color.FgYellow, color.Bold)
		fmt.Println()
		fmt.Println()
		yellow.Println("========================================")
		yellow.Println("   Scan interrupted by user (Ctrl+C)")
		yellow.Println("========================================")
		fmt.Println()
		yellow.Println("Saving results found so far...")
		fmt.Println()
	}()
	
	utils.PrintHeader()
	
	utils.PrintDesigner()
	
	cyan := color.New(color.FgCyan)
	cyan.Printf("Version: %s\n", version)
	cyan.Println("Starting Cloudflare Clean IP Scanner...\n")
	
	yellow := color.New(color.FgYellow)
	yellow.Println("Optimized for Iran network conditions")
	yellow.Println("2-Stage Test: Latency (ping < 1000ms) + Download")
	yellow.Println("Sorted by: Lowest Latency")
	yellow.Println("Press Ctrl+C anytime to stop and save current results")
	fmt.Println()
	
	time.Sleep(1 * time.Second)
	
	startTime := time.Now()
	
	ipRanges := config.GetCloudflareRanges()
	
	cyan.Printf("IP Ranges: %d\n", len(ipRanges))
	cyan.Println("Generating IPs from ranges...\n")
	
	ips := scanner.GenerateIPs(ipRanges, 0)
	
	cyan.Printf("Total IPs to scan: %d\n\n", len(ips))
	
	scanner.SetGlobalInterruptFlag(&interrupted, &interruptMux)
	scanner.SetGlobalDataUsage(&dataUsage)
	
	cyan.Println("========================================")
	cyan.Println("      STEP 1: Latency Testing")
	cyan.Println("========================================")
	fmt.Println()
	
	pingResults := scanner.PingIPs(ips)
	
	if len(pingResults) == 0 {
		red := color.New(color.FgRed, color.Bold)
		red.Println("\nNo responsive IPs found!")
		red.Println("Possible reasons:")
		red.Println("  - All IPs have high latency (> 1000ms)")
		red.Println("  - Network connectivity issues")
		red.Println("  - Firewall blocking connections")
		fmt.Println()
		yellow.Println("Try:")
		yellow.Println("  - Check your internet connection")
		yellow.Println("  - Try again later")
		yellow.Println("  - Disable any VPN or proxy")
		
		fmt.Println()
		cyan.Println("========================================")
		cyan.Println("         Scan Statistics")
		cyan.Println("========================================")
		fmt.Printf("Duration: %s\n", formatDuration(time.Since(startTime)))
		cyan.Println("========================================")
		fmt.Println()
		
		os.Exit(1)
	}
	
	if interrupted {
		results := scanner.ConvertPingResultsToIPResults(pingResults)
		saveAndExit(results, time.Since(startTime))
		return
	}
	
	scanner.SortPingResultsByLatency(pingResults)
	
	fmt.Println()
	green := color.New(color.FgGreen)
	green.Printf("Great! Found %d responsive IPs\n\n", len(pingResults))
	
	maxPossible := len(pingResults)
	var maxSpeedTests int
	
	for {
		fmt.Printf("How many IPs do you want to test in STEP 2? (10-%d): ", maxPossible)
		
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		
		if input == "" {
			red := color.New(color.FgRed)
			red.Println("Error: Please enter a number!")
			fmt.Println()
			continue
		}
		
		count, err := strconv.Atoi(input)
		if err != nil {
			red := color.New(color.FgRed)
			red.Println("Error: Invalid number!")
			fmt.Println()
			continue
		}
		
		if count < 10 {
			red := color.New(color.FgRed)
			red.Printf("Error: Number must be at least 10!\n")
			fmt.Println()
			continue
		}
		
		if count > maxPossible {
			red := color.New(color.FgRed)
			red.Printf("Error: Number cannot exceed %d (total found IPs)!\n", maxPossible)
			fmt.Println()
			continue
		}
		
		maxSpeedTests = count
		break
	}
	
	fmt.Println()
	cyan.Printf("Starting speed test for %d IPs...\n\n", maxSpeedTests)
	
	results := scanner.PerformSpeedTest(pingResults, maxSpeedTests)
	
	duration := time.Since(startTime)
	
	if len(results) == 0 {
		red := color.New(color.FgRed, color.Bold)
		red.Println("\nNo clean IPs found in speed test!")
		red.Println("Possible reasons:")
		red.Println("  - URLs are blocked or filtered")
		red.Println("  - Connection timeout")
		red.Println("  - Network instability")
		fmt.Println()
		yellow.Println("Note:")
		yellow.Printf("  - %d IPs had good latency (< 1000ms)\n", len(pingResults))
		yellow.Println("  - But none could complete download test")
		yellow.Println("  - Try again later or use VPN")
		
		fmt.Println()
		cyan.Println("========================================")
		cyan.Println("         Scan Statistics")
		cyan.Println("========================================")
		fmt.Printf("Duration: %s\n", formatDuration(duration))
		fmt.Printf("Data used: %s\n", formatBytes(dataUsage))
		cyan.Println("========================================")
		fmt.Println()
		
		os.Exit(1)
	}
	
	topResults := results
	if len(results) > 10 {
		topResults = results[:10]
	}
	
	utils.PrintResults(topResults)
	
	err := utils.SaveResults(results, "clean_ips.txt")
	if err != nil {
		red := color.New(color.FgRed)
		red.Printf("\nError saving file: %v\n", err)
	} else {
		green = color.New(color.FgGreen)
		green.Println("\nResults saved to clean_ips.txt successfully!")
		green.Printf("Total IPs found: %d\n", len(results))
	}
	
	fmt.Println()
	cyan = color.New(color.FgCyan, color.Bold)
	cyan.Println("========================================")
	cyan.Println("         Scan Statistics")
	cyan.Println("========================================")
	white := color.New(color.FgWhite)
	white.Printf("Duration: %s\n", formatDuration(duration))
	white.Printf("Data used: %s\n", formatBytes(dataUsage))
	if interrupted {
		yellow.Println("Status: Interrupted by user")
	} else {
		green = color.New(color.FgGreen)
		green.Println("Status: Completed successfully")
	}
	cyan.Println("========================================")
	fmt.Println()
}

func saveAndExit(results []scanner.IPResult, duration time.Duration) {
	if len(results) == 0 {
		fmt.Println()
		red := color.New(color.FgRed)
		red.Println("No results to save.")
		os.Exit(1)
	}
	
	topResults := results
	if len(results) > 10 {
		topResults = results[:10]
	}
	
	utils.PrintResults(topResults)
	
	err := utils.SaveResults(results, "clean_ips.txt")
	if err != nil {
		red := color.New(color.FgRed)
		red.Printf("\nError saving file: %v\n", err)
	} else {
		green := color.New(color.FgGreen)
		green.Println("\nResults saved to clean_ips.txt successfully!")
		green.Printf("Total IPs found: %d\n", len(results))
	}
	
	fmt.Println()
	cyan := color.New(color.FgCyan, color.Bold)
	cyan.Println("========================================")
	cyan.Println("         Scan Statistics")
	cyan.Println("========================================")
	white := color.New(color.FgWhite)
	white.Printf("Duration: %s\n", formatDuration(duration))
	white.Printf("Data used: %s\n", formatBytes(dataUsage))
	yellow := color.New(color.FgYellow)
	yellow.Println("Status: Interrupted by user")
	cyan.Println("========================================")
	fmt.Println()
	
	os.Exit(0)
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	
	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}