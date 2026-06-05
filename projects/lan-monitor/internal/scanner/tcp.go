package scanner

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// tcpProbe checks if specific TCP ports are open on an IP
func tcpProbe(ip string, ports []int, timeout time.Duration) []int {
	var openPorts []int
	var mu sync.Mutex
	sem := make(chan struct{}, 20)
	var wg sync.WaitGroup

	for _, port := range ports {
		wg.Add(1)
		sem <- struct{}{}
		go func(p int) {
			defer wg.Done()
			defer func() { <-sem }()
			addr := fmt.Sprintf("%s:%d", ip, p)
			conn, err := net.DialTimeout("tcp", addr, timeout)
			if err == nil {
				conn.Close()
				mu.Lock()
				openPorts = append(openPorts, p)
				mu.Unlock()
			}
		}(port)
	}
	wg.Wait()
	return openPorts
}

// Well-known MAC OUI to vendor mapping (common vendors)
var vendorDB = map[string]string{
	"00:50:56": "VMware",
	"00:0C:29": "VMware",
	"00:05:69": "VMware",
	"08:00:27": "Oracle VirtualBox",
	"52:54:00": "QEMU/KVM",
	"00:1B:21": "Intel",
	"00:1C:C0": "Intel",
	"00:1E:C9": "Intel",
	"00:1F:3B": "Intel",
	"00:21:5A": "Intel",
	"00:21:5B": "Intel",
	"00:21:5C": "Intel",
	"00:21:5D": "Intel",
	"00:22:4D": "Intel",
	"00:26:C6": "Intel",
	"00:27:0E": "Intel",
	"3C:A0:67": "Intel",
	"F0:1F:AF": "Intel",
	"00:1A:A0": "Dell",
	"00:1D:09": "Dell",
	"00:23:AE": "Dell",
	"00:25:64": "Dell",
	"B8:AC:6F": "Dell",
	"00:1B:63": "Apple",
	"00:1E:C2": "Apple",
	"00:1F:F3": "Apple",
	"00:24:36": "Apple",
	"00:26:08": "Apple",
	"00:26:B0": "Apple",
	"04:1E:64": "Apple",
	"28:CF:E9": "Apple",
	"40:30:04": "Apple",
	"60:F4:45": "Apple",
	"8C:85:90": "Apple",
	"AC:BC:32": "Apple",
	"B0:65:BD": "Apple",
	"DC:A9:04": "Apple",
	"F0:18:98": "Apple",
	"00:1B:FC": "Cisco",
	"00:1C:F6": "Cisco",
	"00:1E:BE": "Cisco",
	"00:23:EB": "Cisco",
	"00:26:0B": "Cisco",
	"58:97:BD": "Cisco",
	"68:EF:BD": "Cisco",
	"70:81:05": "Cisco",
	"F4:4E:05": "Cisco",
	"00:1A:79": "Samsung",
	"00:1E:DF": "Samsung",
	"00:23:D4": "Samsung",
	"04:FE:31": "Samsung",
	"18:67:B0": "Samsung",
	"1C:62:B8": "Samsung",
	"38:01:46": "Samsung",
	"60:D0:A9": "Samsung",
	"80:C5:E6": "Samsung",
	"84:38:35": "Samsung",
	"88:BD:45": "Samsung",
	"8C:79:67": "Samsung",
	"B4:CD:27": "Samsung",
	"C4:57:6E": "Samsung",
	"CC:05:77": "Samsung",
	"F4:F5:D8": "Samsung",
	"00:08:22": "Nokia",
	"00:1B:AF": "Nokia",
	"1C:7B:21": "Sony",
	"00:04:4F": "Raspberry Pi",
	"E4:5F:01": "Raspberry Pi",
	"28:CD:C1": "Raspberry Pi",
	"00:15:6D": "Foxconn",
	"00:18:AE": "Foxconn",
	"00:1C:25": "Foxconn",
	"00:1E:EC": "Foxconn",
	"00:0C:6E": "Asus",
	"00:1A:8D": "Asus",
	"00:1B:8E": "Asus",
	"00:1D:60": "Asus",
	"00:1E:8C": "Asus",
	"00:23:54": "Asus",
	"00:24:8C": "Asus",
	"00:25:22": "Asus",
	"08:60:6E": "Asus",
	"0C:9D:92": "Asus",
	"18:D6:C7": "Asus",
	"1C:87:2C": "Asus",
	"2C:FD:A1": "Asus",
	"38:2C:4A": "Asus",
	"40:16:7E": "Asus",
	"54:A0:50": "Asus",
	"AC:9E:17": "Asus",
	"BC:EE:7B": "Asus",
	"D8:50:E6": "Asus",
	"E0:CB:4E": "Asus",
	"F0:79:59": "Asus",
	"00:1E:58": "HP",
	"00:1F:29": "HP",
	"00:23:7D": "HP",
	"00:26:55": "HP",
	"2C:27:D7": "HP",
	"3C:D9:2B": "HP",
	"40:A8:F0": "HP",
	"64:51:06": "HP",
	"80:C1:6E": "HP",
	"98:4F:EE": "Huawei",
	"00:18:82": "Huawei",
	"00:1E:33": "Huawei",
	"00:25:9E": "Huawei",
	"28:6E:D4": "Huawei",
	"48:DB:50": "Huawei",
	"6C:0B:84": "Huawei",
	"00:18:E7": "Xiaomi",
	"00:1E:2A": "Xiaomi",
	"04:7C:16": "Xiaomi",
	"0C:1D:AF": "Xiaomi",
	"10:44:00": "Xiaomi",
	"18:9E:FC": "Xiaomi",
	"20:82:C0": "Xiaomi",
	"28:6C:07": "Xiaomi",
	"30:B4:9E": "Xiaomi",
	"34:CE:00": "Xiaomi",
	"38:DE:AD": "Xiaomi",
	"40:31:3C": "Xiaomi",
	"44:A5:6E": "Xiaomi",
	"4C:63:76": "Xiaomi",
	"50:64:2B": "Xiaomi",
	"54:E4:3A": "Xiaomi",
	"58:1F:AA": "Xiaomi",
	"60:AB:67": "Xiaomi",
	"64:09:80": "Xiaomi",
	"64:CC:2E": "Xiaomi",
	"68:DF:DD": "Xiaomi",
	"78:02:F8": "Xiaomi",
	"7C:1D:D9": "Xiaomi",
	"80:50:3B": "Xiaomi",
	"88:C9:D0": "Xiaomi",
	"8C:53:C3": "Xiaomi",
	"90:48:6C": "Xiaomi",
	"94:65:2D": "Xiaomi",
	"98:07:2C": "Xiaomi",
	"9C:99:A0": "Xiaomi",
	"A4:6C:F1": "Xiaomi",
	"B0:E1:7E": "Xiaomi",
	"C4:6A:B7": "Xiaomi",
	"C8:8B:47": "Xiaomi",
	"CC:9E:A2": "Xiaomi",
	"D4:97:0B": "Xiaomi",
	"D8:0D:17": "Xiaomi",
	"E0:27:1C": "Xiaomi",
	"F4:60:E2": "Xiaomi",
	"F8:A4:5F": "Xiaomi",
	"FC:64:BA": "Xiaomi",
	"B8:27:EB": "Rockchip",
	"00:04:A3": "Microchip",
	"DC:A6:32": "Rockchip",
}

// lookupVendor returns the vendor name based on MAC OUI prefix
func lookupVendor(mac string) string {
	if len(mac) < 8 {
		return "Unknown"
	}
	prefix := mac[:8]
	if vendor, ok := vendorDB[prefix]; ok {
		return vendor
	}
	// Also try 6-char prefix
	if len(mac) >= 8 {
		prefix6 := mac[:8]
		_ = prefix6
	}
	return "Unknown"
}
