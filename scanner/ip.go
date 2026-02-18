package scanner

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
)

func buildProgressBar(filled, width int) string {
	bar := "["
	for j := 0; j < width; j++ {
		if j < filled {
			bar += "="
		} else if j == filled {
			bar += ">"
		} else {
			bar += " "
		}
	}
	return bar + "]"
}

func randIPEndWith(num byte) byte {
	if num == 0 {
		return 0
	}
	return byte(rand.Intn(int(num)))
}

type IPRanges struct {
	ips     []*net.IPAddr
	mask    string
	firstIP net.IP
	ipNet   *net.IPNet
}

func newIPRanges() *IPRanges {
	return &IPRanges{
		ips: make([]*net.IPAddr, 0),
	}
}

func (r *IPRanges) fixIP(ip string) string {
	if i := strings.IndexByte(ip, '/'); i < 0 {
		if isIPv4(ip) {
			r.mask = "/32"
		} else {
			r.mask = "/128"
		}
		ip += r.mask
	} else {
		r.mask = ip[i:]
	}
	return ip
}

func (r *IPRanges) parseCIDR(ip string) {
	var err error
	if r.firstIP, r.ipNet, err = net.ParseCIDR(r.fixIP(ip)); err != nil {
		fmt.Printf("ParseCIDR error: %v\n", err)
		return
	}
}

func (r *IPRanges) appendIPv4(d byte) {
	r.appendIP(net.IPv4(r.firstIP[12], r.firstIP[13], r.firstIP[14], d))
}

func (r *IPRanges) appendIP(ip net.IP) {
	r.ips = append(r.ips, &net.IPAddr{IP: ip})
}

func (r *IPRanges) getIPRange() (minIP, hosts byte) {
	minIP = r.firstIP[15] & r.ipNet.Mask[3]
	m := net.IPv4Mask(255, 255, 255, 255)
	for i, v := range r.ipNet.Mask {
		m[i] ^= v
	}
	total, _ := strconv.ParseInt(m.String(), 16, 32)
	if total > 255 {
		hosts = 255
		return
	}
	hosts = byte(total)
	return
}

func (r *IPRanges) chooseIPv4() {
	if r.mask == "/32" {
		r.appendIP(r.firstIP)
	} else {
		minIP, hosts := r.getIPRange()
		maxIterations := 0
		for r.ipNet.Contains(r.firstIP) {
			for i := 0; i < 3; i++ {
				r.appendIPv4(minIP + randIPEndWith(hosts))
			}
			r.firstIP[14]++
			if r.firstIP[14] == 0 {
				r.firstIP[13]++
				if r.firstIP[13] == 0 {
					r.firstIP[12]++
				}
			}
			maxIterations++
			if maxIterations > 10000 {
				break
			}
		}
	}
}

func (r *IPRanges) chooseIPv6() {
	if r.mask == "/128" {
		r.appendIP(r.firstIP)
	} else {
		var tempIP uint8
		maxIterations := 0
		for r.ipNet.Contains(r.firstIP) {
			r.firstIP[15] = randIPEndWith(255)
			r.firstIP[14] = randIPEndWith(255)
			targetIP := make([]byte, len(r.firstIP))
			copy(targetIP, r.firstIP)
			r.appendIP(targetIP)
			for i := 13; i >= 0; i-- {
				tempIP = r.firstIP[i]
				r.firstIP[i] += randIPEndWith(255)
				if r.firstIP[i] >= tempIP {
					break
				}
			}
			maxIterations++
			if maxIterations > 10000 {
				break
			}
		}
	}
}

func isIPv4(ip string) bool {
	return strings.Contains(ip, ".")
}

func GenerateIPs(ranges []string, countPerRange int) []*net.IPAddr {
	ipRanges := newIPRanges()
	for _, ipRange := range ranges {
		ipRange = strings.TrimSpace(ipRange)
		if ipRange == "" {
			continue
		}
		ipRanges.parseCIDR(ipRange)
		if isIPv4(ipRange) {
			ipRanges.chooseIPv4()
		} else {
			ipRanges.chooseIPv6()
		}
	}
	return ipRanges.ips
}