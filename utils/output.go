package utils

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/4n0nymou3/CF-Clean-IP-Scanner/scanner"
)

func PrintResults(results []scanner.IPResult) {
	fmt.Println()
	cyan := color.New(color.FgCyan, color.Bold)
	cyan.Println("========================================")
	cyan.Println("        TOP 10 CLEAN IPs FOUND")
	cyan.Println("========================================")
	fmt.Println()
	
	green := color.New(color.FgGreen, color.Bold)
	white := color.New(color.FgWhite)
	yellow := color.New(color.FgYellow)
	
	green.Printf("%-6s %-20s %-15s\n", "Rank", "IP Address", "Latency")
	cyan.Println("-------------------------------------------")
	
	for i, result := range results {
		rank := fmt.Sprintf("%d.", i+1)
		latency := fmt.Sprintf("%dms", result.Latency)
		
		if i == 0 {
			yellow.Printf("%-6s ", rank)
			yellow.Printf("%-20s ", result.IP.String())
			yellow.Printf("%-15s\n", latency)
		} else if result.Latency < 100 {
			white.Printf("%-6s ", rank)
			green.Printf("%-20s ", result.IP.String())
			green.Printf("%-15s\n", latency)
		} else if result.Latency < 200 {
			white.Printf("%-6s ", rank)
			cyan.Printf("%-20s ", result.IP.String())
			cyan.Printf("%-15s\n", latency)
		} else {
			white.Printf("%-6s ", rank)
			white.Printf("%-20s ", result.IP.String())
			white.Printf("%-15s\n", latency)
		}
	}
	
	cyan.Println("========================================")
}

func SaveResults(results []scanner.IPResult, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	file.WriteString("# Clean Cloudflare IPs (Sorted by Latency)\n")
	file.WriteString(fmt.Sprintf("# Generated at: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	file.WriteString(fmt.Sprintf("# Total IPs found: %d\n", len(results)))
	file.WriteString("#\n")
	file.WriteString("# Format: Rank | IP Address | Latency\n")
	file.WriteString("#========================================\n\n")
	
	for i, result := range results {
		line := fmt.Sprintf("%d. %s | %dms\n",
			i+1,
			result.IP.String(),
			result.Latency,
		)
		file.WriteString(line)
	}
	
	file.WriteString("\n# End of results\n")
	
	return nil
}