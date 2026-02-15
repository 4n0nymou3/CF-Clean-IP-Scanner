package utils

import (
	"fmt"

	"github.com/fatih/color"
)

func PrintHeader() {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen, color.Bold)
	yellow := color.New(color.FgYellow)
	
	fmt.Println()
	cyan.Println("========================================")
	green.Println("   CLOUDFLARE CLEAN IP SCANNER")
	yellow.Println("   Find the fastest Cloudflare IPs")
	cyan.Println("========================================")
}