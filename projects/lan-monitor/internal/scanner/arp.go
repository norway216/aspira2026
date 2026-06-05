package scanner

import (
	"bufio"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var (
	arpCache    = make(map[string]string)
	arpCacheMu  sync.RWMutex
	arpCacheAge = make(map[string]time.Time)
)

func arpLookup(ip string, timeout time.Duration) (string, bool) {
	arpCacheMu.RLock()
	if mac, ok := arpCache[ip]; ok {
		if time.Since(arpCacheAge[ip]) < 5*time.Minute {
			arpCacheMu.RUnlock()
			return mac, true
		}
	}
	arpCacheMu.RUnlock()

	mac := arpLookupNative(ip)
	if mac != "" {
		arpCacheMu.Lock()
		arpCache[ip] = mac
		arpCacheAge[ip] = time.Now()
		arpCacheMu.Unlock()
		return mac, true
	}

	mac = arpLookupCommand(ip)
	if mac != "" {
		arpCacheMu.Lock()
		arpCache[ip] = mac
		arpCacheAge[ip] = time.Now()
		arpCacheMu.Unlock()
		return mac, true
	}

	return "", false
}

func arpLookupNative(ip string) string {
	data, err := os.ReadFile("/proc/net/arp")
	if err != nil {
		return ""
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Scan()
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 4 && fields[0] == ip {
			mac := fields[3]
			if mac != "00:00:00:00:00:00" {
				return mac
			}
		}
	}
	return ""
}

func arpLookupCommand(ip string) string {
	out, err := exec.Command("arp", "-n", ip).Output()
	if err != nil {
		return ""
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == ip {
			return fields[2]
		}
	}
	return ""
}
