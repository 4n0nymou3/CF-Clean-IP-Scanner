package main

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/4n0nymou3/CF-Clean-IP-Scanner/config"
	"github.com/4n0nymou3/CF-Clean-IP-Scanner/scanner"
	"github.com/4n0nymou3/CF-Clean-IP-Scanner/utils"
)

const version = "1.0.0"

func main() {
	utils.PrintHeader()
	
	utils.PrintDesigner()
	
	cyan := color.New(color.FgCyan)
	cyan.Printf("Version: %s\n", version)
	cyan.Println("Starting Cloudflare Clean IP Scanner...\n")
	
	yellow := color.New(color.FgYellow)
	yellow.Println("Optimized for Iran network conditions")
	yellow.Println("2-Stage Test: Latency (ping < 500ms) + Download")
	yellow.Println("Sorted by: Lowest Latency\n")
	
	time.Sleep(1 * time.Second)
	
	ipRanges := config.GetCloudflareRanges()
	
	cyan.Printf("IP Ranges: %d\n", len(ipRanges))
	cyan.Println("Generating IPs from ranges...\n")
	
	ips := scanner.GenerateIPs(ipRanges, 0)
	
	cyan.Printf("Total IPs to scan: %d\n\n", len(ips))
	
	results := scanner.ScanIPs(ips)
	
	if len(results) == 0 {
		red := color.New(color.FgRed, color.Bold)
		red.Println("\nNo clean IPs found!")
		red.Println("Possible reasons:")
		red.Println("  - All IPs have high latency (> 500ms)")
		red.Println("  - No IPs can download successfully")
		red.Println("  - Network issues")
		fmt.Println()
		yellow.Println("Try:")
		yellow.Println("  - Run again at different time (night)")
		yellow.Println("  - Enable VPN if available")
		yellow.Println("  - Check your internet connection")
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
	cyan.Println("     Scan completed successfully!")
	cyan.Println("========================================")
	fmt.Println()
}