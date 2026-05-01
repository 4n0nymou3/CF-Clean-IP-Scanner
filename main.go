package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/4n0nymou3/CF-Clean-IP-Scanner/config"
	"github.com/4n0nymou3/CF-Clean-IP-Scanner/scanner"
	"github.com/4n0nymou3/CF-Clean-IP-Scanner/utils"
)

const version = "2.0.0"

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func printScanStats(elapsed time.Duration, interrupted bool) {
	fmt.Println()
	cyan := color.New(color.FgCyan, color.Bold)
	cyan.Println("========================================")
	if interrupted {
		color.New(color.FgYellow, color.Bold).Println("         Scan stopped by user")
	} else {
		cyan.Println("      Scan completed successfully!")
	}
	cyan.Println("========================================")
	fmt.Println()
	color.New(color.FgCyan).Printf("  Scan Duration : %s\n", formatDuration(elapsed))
	fmt.Println()
}

func askScanMode() int {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println()
		color.New(color.FgCyan, color.Bold).Println("Select scan mode:")
		color.New(color.FgWhite).Println("  [1] Normal scan (TCP ping + speed test)")
		color.New(color.FgWhite).Println("  [2] Xray scan (uses Xray core with your config)")
		fmt.Print("Enter 1 or 2: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "1" {
			return 1
		} else if input == "2" {
			if _, err := os.Stat("./xray/xray"); os.IsNotExist(err) {
				color.New(color.FgRed).Println("Error: Xray binary not found. Please reinstall the tool.")
				os.Exit(1)
			}
			if _, err := os.Stat("./config/xray_config.json"); os.IsNotExist(err) {
				color.New(color.FgRed).Println("Error: Xray config not found at config/xray_config.json")
				color.New(color.FgYellow).Println("Please edit the sample config file first.")
				os.Exit(1)
			}
			return 2
		} else {
			color.New(color.FgRed).Println("Invalid choice. Please enter 1 or 2.")
		}
	}
}

func main() {
	utils.PrintHeader()
	utils.PrintDesigner()

	cyan := color.New(color.FgCyan)
	cyan.Printf("Version: %s\n", version)
	fmt.Println()

	color.New(color.FgYellow).Println("Optimized for Iran network conditions")
	color.New(color.FgYellow).Println("Press Ctrl+C at any time to stop and see results found so far.")
	fmt.Println()

	mode := askScanMode()

	time.Sleep(500 * time.Millisecond)

	stopPingCh := make(chan struct{})
	stopSpeedCh := make(chan struct{})
	inSpeedPhase := int32(0)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		for {
			<-sigChan
			fmt.Println()
			if atomic.LoadInt32(&inSpeedPhase) == 0 {
				color.New(color.FgYellow, color.Bold).Println("Interrupt received. Stopping ping phase and proceeding to speed test with IPs found so far...")
				select {
				case <-stopPingCh:
				default:
					close(stopPingCh)
				}
			} else {
				color.New(color.FgYellow, color.Bold).Println("Interrupt received. Stopping speed test and collecting results...")
				signal.Reset(os.Interrupt)
				select {
				case <-stopSpeedCh:
				default:
					close(stopSpeedCh)
				}
				return
			}
		}
	}()

	startTime := time.Now()

	ipRanges := config.GetCloudflareRanges()
	ips := scanner.GenerateIPs(ipRanges)

	fmt.Println()

	var pingResults []scanner.PingResult
	var pingWasStopped bool

	if mode == 1 {
		pingResults = scanner.PingIPs(stopPingCh, ips)

		select {
		case <-stopPingCh:
			pingWasStopped = true
		default:
		}
	} else {
		pingResults = scanner.PingIPsViaXray(stopPingCh, ips)

		select {
		case <-stopPingCh:
			pingWasStopped = true
		default:
		}
	}

	if pingWasStopped && len(pingResults) == 0 {
		elapsed := time.Since(startTime)
		color.New(color.FgYellow).Println("Scan stopped during latency test. No responsive IPs found yet.")
		printScanStats(elapsed, true)
		return
	}

	if !pingWasStopped && len(pingResults) == 0 {
		color.New(color.FgRed, color.Bold).Println("No responsive IPs found!")
		fmt.Println()
		color.New(color.FgYellow).Println("Try running again. Network conditions may vary.")
		elapsed := time.Since(startTime)
		printScanStats(elapsed, false)
		return
	}

	fmt.Println()

	atomic.StoreInt32(&inSpeedPhase, 1)
	var results []scanner.IPResult
	if mode == 1 {
		results = scanner.SpeedTest(stopSpeedCh, pingResults)
	} else {
		results = scanner.SpeedTestViaXray(stopSpeedCh, pingResults)
	}

	elapsed := time.Since(startTime)

	interrupted := false
	select {
	case <-stopSpeedCh:
		interrupted = true
	default:
	}

	if len(results) == 0 {
		red := color.New(color.FgRed, color.Bold)
		if interrupted {
			red.Println("No clean IPs found before scan was stopped.")
		} else {
			red.Println("No clean IPs found.")
			fmt.Println()
			color.New(color.FgYellow).Println("Try running again at a different time.")
		}
		printScanStats(elapsed, interrupted)
		return
	}

	topResults := results
	if len(results) > 10 {
		topResults = results[:10]
	}

	if interrupted {
		color.New(color.FgYellow, color.Bold).Printf(
			"\nShowing %d clean IP(s) found before scan was stopped:\n", len(results))
	}

	utils.PrintResults(topResults)

	if err := utils.SaveResults(results, "clean_ips.txt"); err != nil {
		color.New(color.FgRed).Printf("Error saving file: %v\n", err)
	} else {
		color.New(color.FgGreen).Println("Results saved to clean_ips.txt")
		color.New(color.FgGreen).Printf("Total clean IPs found: %d\n", len(results))
	}

	if err := utils.SaveSimpleResults(topResults, pingResults, "clean_ips_list.txt"); err != nil {
		color.New(color.FgRed).Printf("Error saving simple list: %v\n", err)
	} else {
		color.New(color.FgGreen).Println("Simple IP list saved to clean_ips_list.txt")
	}

	printScanStats(elapsed, interrupted)
}