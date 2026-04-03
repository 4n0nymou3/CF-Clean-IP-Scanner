package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func GetCloudflareRanges() []string {
	file, err := os.Open("config/ip_ranges.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not open config/ip_ranges.txt: %v\n", err)
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
		fmt.Fprintf(os.Stderr, "Error reading config/ip_ranges.txt: %v\n", err)
		os.Exit(1)
	}
	return ranges
}