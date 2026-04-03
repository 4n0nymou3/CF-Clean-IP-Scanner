package scanner

import (
	"fmt"
	"math/rand"
	"net"
	"strings"
)

type IPRanges struct {
	ips  []*net.IPAddr
	seen map[string]bool
}

func newIPRanges() *IPRanges {
	return &IPRanges{
		ips:  make([]*net.IPAddr, 0),
		seen: make(map[string]bool),
	}
}

func (r *IPRanges) appendIP(ip net.IP) {
	r.ips = append(r.ips, &net.IPAddr{IP: ip})
}

func (r *IPRanges) expandCIDR(cidr string) {
	cidr = strings.TrimSpace(cidr)
	if cidr == "" {
		return
	}

	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		fmt.Printf("ParseCIDR error for %s: %v\n", cidr, err)
		return
	}

	networkKey := ipNet.String()
	if r.seen[networkKey] {
		return
	}
	r.seen[networkKey] = true

	ip := cloneIP(ipNet.IP)
	for ipNet.Contains(ip) {
		clone := make(net.IP, len(ip))
		copy(clone, ip)
		r.appendIP(clone)
		incrementIP(ip)
	}
}

func cloneIP(ip net.IP) net.IP {
	clone := make(net.IP, len(ip))
	copy(clone, ip)
	return clone
}

func incrementIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}

func isIPv4(ip string) bool {
	return strings.Contains(ip, ".")
}

func GenerateIPs(ranges []string) []*net.IPAddr {
	ipRanges := newIPRanges()
	for _, r := range ranges {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		if !strings.Contains(r, "/") {
			if isIPv4(r) {
				r += "/32"
			} else {
				r += "/128"
			}
		}
		ipRanges.expandCIDR(r)
	}

	rand.Shuffle(len(ipRanges.ips), func(i, j int) {
		ipRanges.ips[i], ipRanges.ips[j] = ipRanges.ips[j], ipRanges.ips[i]
	})

	return ipRanges.ips
}