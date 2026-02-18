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
	cyan.Println("===========================================================================")
	cyan.Println("                      CLEAN IPs FOUND")
	cyan.Println("===========================================================================")
	fmt.Println()

	green := color.New(color.FgGreen, color.Bold)
	white := color.New(color.FgWhite)
	yellow := color.New(color.FgYellow, color.Bold)

	green.Printf("%-6s %-20s %-6s %-10s %-10s %-14s %-18s\n",
		"Rank", "IP Address", "Sent", "Received", "Loss", "Avg Delay", "Download Speed")
	cyan.Println("---------------------------------------------------------------------------")

	for i, r := range results {
		rank := fmt.Sprintf("%d.", i+1)
		sent := fmt.Sprintf("%d", r.Sended)
		recv := fmt.Sprintf("%d", r.Received)
		loss := fmt.Sprintf("%.2f", r.LossRate)
		delay := fmt.Sprintf("%dms", r.Delay)
		speed := fmt.Sprintf("%.2f MB/s", r.DownloadSpeed/1024/1024)

		if i == 0 {
			yellow.Printf("%-6s %-20s %-6s %-10s %-10s %-14s %-18s\n",
				rank, r.IP.String(), sent, recv, loss, delay, speed)
		} else if r.LossRate == 0 && r.Delay < 150 {
			white.Printf("%-6s ", rank)
			color.New(color.FgGreen).Printf("%-20s %-6s %-10s %-10s %-14s %-18s\n",
				r.IP.String(), sent, recv, loss, delay, speed)
		} else if r.LossRate == 0 {
			white.Printf("%-6s ", rank)
			color.New(color.FgCyan).Printf("%-20s %-6s %-10s %-10s %-14s %-18s\n",
				r.IP.String(), sent, recv, loss, delay, speed)
		} else {
			white.Printf("%-6s %-20s %-6s %-10s %-10s %-14s %-18s\n",
				rank, r.IP.String(), sent, recv, loss, delay, speed)
		}
	}

	cyan.Println("===========================================================================")
}

func SaveResults(results []scanner.IPResult, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	file.WriteString("# Clean Cloudflare IPs\n")
	file.WriteString(fmt.Sprintf("# Generated at: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	file.WriteString(fmt.Sprintf("# Total IPs found: %d\n", len(results)))
	file.WriteString("#\n")
	file.WriteString("# Format: Rank | IP | Sent | Received | Loss | Avg Delay | Download Speed\n")
	file.WriteString("#===========================================================================\n\n")

	for i, r := range results {
		line := fmt.Sprintf("%d. %s | Sent: %d | Recv: %d | Loss: %.2f | %dms | %.2f MB/s\n",
			i+1,
			r.IP.String(),
			r.Sended,
			r.Received,
			r.LossRate,
			r.Delay,
			r.DownloadSpeed/1024/1024,
		)
		file.WriteString(line)
	}

	file.WriteString("\n# End of results\n")
	return nil
}