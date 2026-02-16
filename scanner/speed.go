package scanner

import (
	"fmt"
	"sort"

	"github.com/fatih/color"
)

type IPResult struct {
	IP            *net.IPAddr
	Latency       int
	DownloadSpeed float64
}

func ScanIPs(ips []*net.IPAddr) []IPResult {
	cyan := color.New(color.FgCyan, color.Bold)
	cyan.Println("========================================")
	cyan.Println("      STEP 1: Latency Testing")
	cyan.Println("========================================")
	fmt.Println()

	pingResults := PingIPs(ips)

	if len(pingResults) == 0 {
		return nil
	}

	sort.Slice(pingResults, func(i, j int) bool {
		return pingResults[i].Latency < pingResults[j].Latency
	})

	cyan.Printf("Responsive IPs: %d\n", len(pingResults))
	fmt.Println()

	cyan.Println("========================================")
	cyan.Println("      STEP 2: Filtering by Latency")
	cyan.Println("========================================")
	fmt.Println()

	yellow := color.New(color.FgYellow)
	yellow.Println("Filtering IPs with latency under 500ms...")
	fmt.Println()

	var results []IPResult
	
	barWidth := 50
	total := len(pingResults)
	
	for i, pr := range pingResults {
		if pr.Latency.Milliseconds() < 500 {
			results = append(results, IPResult{
				IP:            pr.IP,
				Latency:       int(pr.Latency.Milliseconds()),
				DownloadSpeed: 0,
			})
		}
		
		progress := float64(i+1) / float64(total)
		filledWidth := int(progress * float64(barWidth))
		
		bar := "["
		for j := 0; j < barWidth; j++ {
			if j < filledWidth {
				bar += "="
			} else if j == filledWidth {
				bar += ">"
			} else {
				bar += " "
			}
		}
		bar += "]"
		
		fmt.Printf("\r%s %3d%% (%d/%d) - Found: %d", bar, int(progress*100), i+1, total, len(results))
	}
	
	fmt.Println()
	fmt.Println()
	
	green := color.New(color.FgGreen)
	green.Printf("Filtering completed: %d clean IPs found (latency < 500ms)\n\n", len(results))

	sort.Slice(results, func(i, j int) bool {
		return results[i].Latency < results[j].Latency
	})

	return results
}