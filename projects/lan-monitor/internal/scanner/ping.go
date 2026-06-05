package scanner

import (
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// pingCheck checks if an IP is reachable via ICMP
func pingCheck(ip string, timeout time.Duration) bool {
	// Try Go native ping first
	if pingNative(ip, timeout) {
		return true
	}

	// Fallback to system ping command
	return pingCommand(ip, timeout)
}

func pingNative(ip string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("ip4:icmp", ip, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func pingCommand(ip string, timeout time.Duration) bool {
	cmd := exec.Command("ping", "-c", "1", "-W", "1", ip)
	err := cmd.Run()
	return err == nil
}

// pingScan performs a concurrent ping scan of a subnet
func pingScan(subnet string, timeout time.Duration) []string {
	ips, err := generateIPList(subnet)
	if err != nil {
		return nil
	}

	var results []string
	var mu sync.Mutex
	sem := make(chan struct{}, 100)
	var wg sync.WaitGroup

	for _, ip := range ips {
		wg.Add(1)
		sem <- struct{}{}
		go func(ip string) {
			defer wg.Done()
			defer func() { <-sem }()
			if pingCheck(ip, timeout) {
				mu.Lock()
				results = append(results, ip)
				mu.Unlock()
			}
		}(ip)
	}
	wg.Wait()
	return results
}

// generateIPList generates all IPs in a CIDR subnet
func generateIPList(subnet string) ([]string, error) {
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		// Try parsing as single IP
		ip := net.ParseIP(subnet)
		if ip != nil {
			return []string{subnet}, nil
		}
		return nil, err
	}

	var ips []string
	ip := ipNet.IP.Mask(ipNet.Mask)
	for ipNet.Contains(ip) {
		ipStr := ip.String()
		// Skip network and broadcast addresses
		ips = append(ips, ipStr)
		ip = nextIP(ip)
	}
	// Remove network address and broadcast
	if len(ips) > 2 {
		ips = ips[1 : len(ips)-1]
	}
	return ips, nil
}

func nextIP(ip net.IP) net.IP {
	next := make(net.IP, len(ip))
	copy(next, ip)
	for i := len(next) - 1; i >= 0; i-- {
		next[i]++
		if next[i] != 0 {
			break
		}
	}
	return next
}

// resolveHostname tries to resolve a hostname for an IP
func resolveHostname(ip string) string {
	names, err := net.LookupAddr(ip)
	if err != nil || len(names) == 0 {
		return ""
	}
	hostname := strings.TrimSuffix(names[0], ".")
	return hostname
}
