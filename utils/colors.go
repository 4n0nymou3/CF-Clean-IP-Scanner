package utils

import (
	"fmt"

	"github.com/fatih/color"
)

func PrintHeader() {
	cyan := color.New(color.FgCyan, color.Bold)
	cyan.Println("========================================")
	cyan.Println("   CLOUDFLARE CLEAN IP SCANNER")
	cyan.Println("   Find the fastest Cloudflare IPs")
	cyan.Println("========================================")
}

func PrintDesigner() {
	green := color.New(color.FgGreen, color.Bold)
	green.Println("...:..::.::: Designed by: Anonymous :::.::..:...")
	fmt.Println()
}