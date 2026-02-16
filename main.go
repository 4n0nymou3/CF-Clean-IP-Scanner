package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/fatih/color"
	"github.com/4n0nymou3/CF-Clean-IP-Scanner/config"
	"github.com/4n0nymou3/CF-Clean-IP-Scanner/scanner"
	"github.com/4n0nymou3/CF-Clean-IP-Scanner/utils"
)

const version = "1.1.0"

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
			yellow.Println("  ./cf-scanner           (test 500 IPs - default)")
			yellow.Println("  ./cf-scanner <number>  (test custom number of IPs)")
			fmt.Println()
			yellow.Println("Valid range: 10 to 1000")
			fmt.Println()
			yellow.Println("Examples:")
			yellow.Println("  ./cf-scanner 100   (test 100 IPs)")
			yellow.Println("  ./cf-scanner 250   (test 250 IPs)")
			yellow.Println("  ./cf-scanner 1000  (test 1000 IPs)")
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
			yellow.Println("  ./cf-scanner 10    (minimum)")
			yellow.Println("  ./cf-scanner 500   (default)")
			yellow.Println("  ./cf-scanner 1000  (maximum)")
			os.Exit(1)
		}
		
		maxSpeedTests = customCount
	}
	
	utils.PrintHeader()
	
	utils.PrintDesigner()
	
	cyan := color.New(color.FgCyan)
	cyan.Printf("Version: %s\n", version)
	cyan.Println("Starting Cloudflare Clean IP Scanner...\n")
	
	yellow := color.New(color.FgYellow)
	yellow.Println("Optimized for Iran network conditions")
	yellow.Println("2-Stage Test: Latency (ping < 500ms) + Download")
	yellow.Println("Sorted by: Lowest Latency")
	
	if len(os.Args) > 1 {
		green := color.New(color.FgGreen)
		green.Printf("Custom mode: Testing %d IPs in STEP 2\n", maxSpeedTests)
	} else {
		cyan.Printf("Default mode: Testing %d IPs in STEP 2\n", maxSpeedTests)
	}
	fmt.Println()
	
	time.Sleep(1 * time.Second)
	
	ipRanges := config.GetCloudflareRanges()
	
	cyan.Printf("IP Ranges: %d\n", len(ipRanges))
	cyan.Println("Generating IPs from ranges...\n")
	
	ips := scanner.GenerateIPs(ipRanges, 0)
	
	cyan.Printf("Total IPs to scan: %d\n\n", len(ips))
	
	results := scanner.ScanIPs(ips, maxSpeedTests)
	
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
		yellow.Println("  - Use lower number: ./cf-scanner 100")
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