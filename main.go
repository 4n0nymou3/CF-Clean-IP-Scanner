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
	
	green := color.New(color.FgGreen, color.Bold)
	green.Println("\nDesigned by: Anonymous\n")
	
	cyan := color.New(color.FgCyan)
	cyan.Printf("Version: %s\n", version)
	cyan.Println("Starting Cloudflare Clean IP Scanner...\n")
	
	time.Sleep(1 * time.Second)
	
	ipRanges := config.GetCloudflareRanges()
	
	yellow := color.New(color.FgYellow)
	yellow.Printf("IP Ranges: %d\n", len(ipRanges))
	yellow.Println("Generating random IPs...\n")
	
	ips := scanner.GenerateIPs(ipRanges, 200)
	
	cyan.Printf("Total IPs to scan: %d\n\n", len(ips))
	
	results := scanner.ScanIPs(ips)
	
	if len(results) == 0 {
		red := color.New(color.FgRed, color.Bold)
		red.Println("\nNo clean IPs found!")
		red.Println("This can happen due to:")
		red.Println("  - Network filtering/firewall")
		red.Println("  - Unstable internet connection")
		red.Println("  - All IPs are currently slow")
		fmt.Println()
		yellow.Println("Please try again in a few minutes.")
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