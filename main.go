package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/4n0nymou3/CF-Clean-IP-Scanner/config"
	"github.com/4n0nymou3/CF-Clean-IP-Scanner/scanner"
	"github.com/4n0nymou3/CF-Clean-IP-Scanner/utils"
)

const version = "1.3.0"

var interrupted bool

func main() {
	maxSpeedTests := 500
	
	if len(os.Args) > 1 {
		customCount, err := strconv.Atoi(os.Args[1])
		if err != nil {
			red := color.New(color.FgRed, color.Bold)
			red.Println("\nError: Invalid number!")
			fmt.Println()
			yellow := color.New(color.FgYellow)
			yellow.Println("Usage:")
			yellow.Println("  cf-scanner           (test 500 IPs - default)")
			yellow.Println("  cf-scanner <number>  (test custom number of IPs)")
			fmt.Println()
			yellow.Println("Valid range: 10 to 1000")
			fmt.Println()
			yellow.Println("Examples:")
			yellow.Println("  cf-scanner 100   (test 100 IPs)")
			yellow.Println("  cf-scanner 250   (test 250 IPs)")
			yellow.Println("  cf-scanner 1000  (test 1000 IPs)")
			os.Exit(1)
		}
		
		if customCount < 10 || customCount > 1000 {
			red := color.New(color.FgRed, color.Bold)
			red.Printf("\nError: Number must be between 10 and 1000!\n")
			red.Printf("You entered: %d\n", customCount)
			fmt.Println()
			yellow := color.New(color.FgYellow)
			yellow.Println("Valid range: 10 to 1000")
			fmt.Println()
			yellow.Println("Examples:")
			yellow.Println("  cf-scanner 10    (minimum)")
			yellow.Println("  cf-scanner 500   (default)")
			yellow.Println("  cf-scanner 1000  (maximum)")
			os.Exit(1)
		}
		
		maxSpeedTests = customCount
	}
	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		interrupted = true
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
	yellow.Println("2-Stage Test: Latency (ping < 500ms) + Download")
	yellow.Println("Sorted by: Lowest Latency")
	yellow.Println("Press Ctrl+C anytime to stop and save current results")
	
	if len(os.Args) > 1 {
		green := color.New(color.FgGreen)
		green.Printf("Custom mode: Testing %d IPs in STEP 2\n", maxSpeedTests)
	} else {
		cyan.Printf("Default mode: Testing %d IPs in STEP 2\n", maxSpeedTests)
	}
	fmt.Println()
	
	time.Sleep(1 * time.Second)
	
	startTime := time.Now()
	
	ipRanges := config.GetCloudflareRanges()
	
	cyan.Printf("IP Ranges: %d\n", len(ipRanges))
	cyan.Println("Generating IPs from ranges...\n")
	
	ips := scanner.GenerateIPs(ipRanges, 0)
	
	cyan.Printf("Total IPs to scan: %d\n\n", len(ips))
	
	results, dataUsage := scanner.ScanIPs(ips, maxSpeedTests, &interrupted)
	
	duration := time.Since(startTime)
	
	if len(results) == 0 {
		red := color.New(color.FgRed, color.Bold)
		red.Println("\nNo clean IPs found!")
		red.Println("Possible reasons:")
		red.Println("  - All IPs have high latency (> 500ms)")
		red.Println("  - No IPs can download successfully")
		red.Println("  - Network issues")
		red.Println("  - Scan was stopped too early")
		fmt.Println()
		yellow.Println("Try:")
		yellow.Println("  - Run again at different time (night)")
		yellow.Println("  - Use lower number: cf-scanner 100")
		yellow.Println("  - Let it run longer before stopping")
		
		fmt.Println()
		cyan = color.New(color.FgCyan, color.Bold)
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
		green := color.New(color.FgGreen)
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
		green := color.New(color.FgGreen)
		green.Println("Status: Completed successfully")
	}
	cyan.Println("========================================")
	fmt.Println()
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