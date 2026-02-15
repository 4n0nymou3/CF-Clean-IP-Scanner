package scanner

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GenerateIPs(ranges []string, countPerRange int) []string {
	var ips []string
	
	for _, cidr := range ranges {
		generated := generateIPsFromCIDR(cidr, countPerRange)
		ips = append(ips, generated...)
	}
	
	return ips
}

func generateIPsFromCIDR(cidr string, count int) []string {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil
	}
	
	var ips []string
	
	ones, bits := ipNet.Mask.Size()
	if bits-ones > 8 {
		for i := 0; i < count; i++ {
			randomIP := generateRandomIP(ip, ipNet)
			ips = append(ips, randomIP.String())
		}
	} else {
		for ip := ip.Mask(ipNet.Mask); ipNet.Contains(ip); inc(ip) {
			ips = append(ips, ip.String())
			if len(ips) >= count {
				break
			}
		}
	}
	
	return ips
}

func generateRandomIP(baseIP net.IP, ipNet *net.IPNet) net.IP {
	ip := make(net.IP, len(baseIP))
	copy(ip, baseIP)
	
	for i := 0; i < len(ip); i++ {
		if ipNet.Mask[i] == 0 {
			ip[i] = byte(rand.Intn(256))
		} else if ipNet.Mask[i] != 255 {
			hostBits := 255 ^ ipNet.Mask[i]
			ip[i] = (baseIP[i] & ipNet.Mask[i]) | byte(rand.Intn(int(hostBits)+1))
		}
	}
	
	return ip
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func parseIP(ipStr string) (byte, byte, byte, byte, error) {
	parts := strings.Split(ipStr, ".")
	if len(parts) != 4 {
		return 0, 0, 0, 0, fmt.Errorf("invalid IP")
	}
	
	a, _ := strconv.Atoi(parts[0])
	b, _ := strconv.Atoi(parts[1])
	c, _ := strconv.Atoi(parts[2])
	d, _ := strconv.Atoi(parts[3])
	
	return byte(a), byte(b), byte(c), byte(d), nil
}