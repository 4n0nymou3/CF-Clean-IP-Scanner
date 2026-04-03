package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func GetCloudflareRanges() []string {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not determine executable path: %v\n", err)
		os.Exit(1)
	}
	exeDir := filepath.Dir(exe)
	filePath := filepath.Join(exeDir, "config", "ip_ranges.txt")

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not open %s: %v\n", filePath, err)
		os.Exit(1)
	}
	defer file.Close()

	var ranges []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		ranges = append(ranges, line)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", filePath, err)
		os.Exit(1)
	}
	return ranges
}