package scanner

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"lan-monitor/internal/config"
	"lan-monitor/internal/database"
	"lan-monitor/internal/models"
	"lan-monitor/internal/websocket"
)

type ScanResult struct {
	IP        string `json:"ip"`
	MAC       string `json:"mac"`
	Hostname  string `json:"hostname"`
	Online    bool   `json:"online"`
	OpenPorts []int  `json:"open_ports,omitempty"`
	Method    string `json:"method"`
}

type ScannerService struct {
	cfg     *config.ScanConfig
	hub     *websocket.Hub
	mu      sync.RWMutex
	running bool
}

func NewScannerService(cfg *config.ScanConfig, hub *websocket.Hub) *ScannerService {
	return &ScannerService{cfg: cfg, hub: hub}
}

func GetLocalSubnets() ([]string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var subnets []string
	seen := make(map[string]bool)
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.To4() == nil {
				continue
			}
			ones, _ := ipNet.Mask.Size()
			scanOnes := ones
			if ones < 24 {
				scanOnes = 24
			}
			network := &net.IPNet{
				IP:   ipNet.IP.Mask(net.CIDRMask(scanOnes, 32)),
				Mask: net.CIDRMask(scanOnes, 32),
			}
			subnet := network.String()
			if !seen[subnet] {
				seen[subnet] = true
				subnets = append(subnets, subnet)
			}
		}
	}
	if len(subnets) == 0 {
		subnets = []string{"192.168.1.0/24", "192.168.0.0/24", "10.0.0.0/24"}
	}
	return subnets, nil
}

func (s *ScannerService) GetLocalSubnets() ([]string, error) {
	return GetLocalSubnets()
}

func (s *ScannerService) GetSubnetsToScan() []string {
	var result []string
	for _, sn := range s.cfg.Subnets {
		if sn == "auto" {
			local, err := s.GetLocalSubnets()
			if err == nil {
				result = append(result, local...)
			}
		} else {
			result = append(result, sn)
		}
	}
	return result
}

func (s *ScannerService) RunScan() (*ScanTaskResult, []ScanResult) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil, nil
	}
	s.running = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	subnets := s.GetSubnetsToScan()
	timeout := time.Duration(s.cfg.TimeoutMs) * time.Millisecond
	var allResults []ScanResult

	for _, subnet := range subnets {
		ips, err := generateIPList(subnet)
		if err != nil {
			continue
		}
		now := time.Now()
		result, _ := database.DB.Exec(
			"INSERT INTO scan_tasks (subnet, scan_type, status, started_at, total_ips) VALUES (?, ?, 'running', ?, ?)",
			subnet, strings.Join(s.cfg.Methods, "+"), now, len(ips))
		taskID, _ := result.LastInsertId()

		jobCh := make(chan string, len(ips))
		resultCh := make(chan ScanResult, len(ips))
		workers := s.cfg.WorkerCount
		if workers > len(ips) {
			workers = len(ips)
		}
		if workers < 1 {
			workers = 1
		}

		var wg sync.WaitGroup
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for ip := range jobCh {
					resultCh <- s.scanIP(ip, timeout)
				}
			}()
		}
		for _, ip := range ips {
			jobCh <- ip
		}
		close(jobCh)
		wg.Wait()
		close(resultCh)

		onlineCount := 0
		for r := range resultCh {
			if r.Online {
				onlineCount++
			}
			allResults = append(allResults, r)
		}
		finished := time.Now()
		database.DB.Exec(
			"UPDATE scan_tasks SET status = 'completed', finished_at = ?, online_count = ? WHERE id = ?",
			finished, onlineCount, taskID)
	}

	s.processScanResults(allResults)
	return nil, allResults
}

type ScanTaskResult struct {
	ID          uint
	Subnet      string
	OnlineCount int
	TotalIPs    int
}

func (s *ScannerService) scanIP(ip string, timeout time.Duration) ScanResult {
	result := ScanResult{IP: ip, Online: false}
	for _, method := range s.cfg.Methods {
		switch method {
		case "arp":
			if mac, ok := arpLookup(ip, timeout); ok {
				result.MAC = mac
				result.Online = true
				result.Method = "arp"
				continue
			}
		case "ping":
			if pingCheck(ip, timeout) {
				result.Online = true
				if result.Method == "" {
					result.Method = "ping"
				}
			}
		case "tcp":
			ports := tcpProbe(ip, s.cfg.TCPPorts, timeout)
			if len(ports) > 0 {
				result.Online = true
				result.OpenPorts = ports
				if result.Method == "" {
					result.Method = "tcp"
				}
			}
		}
	}
	if result.Online {
		result.Hostname = resolveHostname(ip)
		if result.MAC == "" {
			if mac, ok := arpLookup(ip, timeout); ok {
				result.MAC = mac
			}
		}
	}
	return result
}

func (s *ScannerService) processScanResults(results []ScanResult) {
	now := time.Now()
	var statusChanges []websocket.DeviceStatusUpdate

	for _, r := range results {
		var devID uint
		var devIP, devMAC, devHostname, devVendor, devStatus string
		var devConsecutiveFailures, devOfflineCount, devFailCount int
		var devLastOfflineAt, devLastOnlineAt *time.Time
		var devOnlineDurationSeconds int64

		if r.MAC != "" {
			database.DB.QueryRow(
				"SELECT id, ip, mac, hostname, vendor, status, consecutive_failures, last_offline_at, last_online_at, online_duration_seconds, offline_count, fail_count FROM devices WHERE mac = ?",
				r.MAC).Scan(&devID, &devIP, &devMAC, &devHostname, &devVendor, &devStatus,
				&devConsecutiveFailures, &devLastOfflineAt, &devLastOnlineAt,
				&devOnlineDurationSeconds, &devOfflineCount, &devFailCount)
		}
		if devID == 0 && r.IP != "" {
			database.DB.QueryRow(
				"SELECT id, ip, mac, hostname, vendor, status, consecutive_failures, last_offline_at, last_online_at, online_duration_seconds, offline_count, fail_count FROM devices WHERE ip = ? AND mac = ''",
				r.IP).Scan(&devID, &devIP, &devMAC, &devHostname, &devVendor, &devStatus,
				&devConsecutiveFailures, &devLastOfflineAt, &devLastOnlineAt,
				&devOnlineDurationSeconds, &devOfflineCount, &devFailCount)
		}

		isNew := devID == 0
		oldStatus := devStatus

		if isNew {
			vendor := lookupVendor(r.MAC)
			openPorts := ""
			if len(r.OpenPorts) > 0 {
				openPorts = intsToStr(r.OpenPorts)
			}
			result, _ := database.DB.Exec(
				"INSERT INTO devices (ip, mac, hostname, vendor, status, first_seen, last_seen, last_online_at, open_ports) VALUES (?, ?, ?, ?, 'online', ?, ?, ?, ?)",
				r.IP, r.MAC, r.Hostname, vendor, now, now, now, openPorts)
			id, _ := result.LastInsertId()
			devID = uint(id)
			devMAC = r.MAC

			database.DB.Exec(
				"INSERT INTO device_events (device_id, device_mac, event_type, new_status, event_time, description) VALUES (?, ?, 'first_seen', 'online', ?, ?)",
				devID, r.MAC, now, fmt.Sprintf("New device: %s (%s)", r.IP, r.MAC))

			if config.AppConfig.Alert.NewDeviceEnabled {
				result2, _ := database.DB.Exec(
					"INSERT INTO alerts (device_id, device_mac, alert_type, level, title, content, status) VALUES (?, ?, 'new_device', 'info', ?, ?, 'open')",
					devID, r.MAC, fmt.Sprintf("New device: %s", r.IP),
					fmt.Sprintf("IP: %s, MAC: %s, Vendor: %s", r.IP, r.MAC, vendor))
				alertID, _ := result2.LastInsertId()
				aID := uint(alertID)
				s.hub.BroadcastAlert(models.Alert{
					ID: aID, DeviceID: &devID, DeviceMAC: r.MAC,
					AlertType: "new_device", Level: "info",
					Title: fmt.Sprintf("New device: %s", r.IP),
					Content: fmt.Sprintf("IP: %s, MAC: %s, Vendor: %s", r.IP, r.MAC, vendor),
					Status: "open", CreatedAt: now,
				})
			}
		} else if r.Online {
			sets := []string{"last_seen = ?"}
			args := []interface{}{now}
			if r.IP != devIP {
				sets = append(sets, "ip = ?")
				args = append(args, r.IP)
			}
			if r.MAC != "" && devMAC == "" {
				sets = append(sets, "mac = ?")
				args = append(args, r.MAC)
				devMAC = r.MAC
			}
			if r.Hostname != "" {
				sets = append(sets, "hostname = ?")
				args = append(args, r.Hostname)
			}
			if devVendor == "" && r.MAC != "" {
				sets = append(sets, "vendor = ?")
				args = append(args, lookupVendor(r.MAC))
			}
			sets = append(sets, "fail_count = 0", "consecutive_failures = 0")
			if devStatus != "online" {
				if devStatus == "offline" {
					sets = append(sets, "last_online_at = ?", "status = 'online'", "offline_count = ?")
					args = append(args, now, devOfflineCount+1)
					database.DB.Exec(
						"INSERT INTO device_events (device_id, device_mac, event_type, old_status, new_status, event_time, description) VALUES (?, ?, 'online', ?, 'online', ?, ?)",
						devID, devMAC, oldStatus, now, fmt.Sprintf("Device %s back online", r.IP))
					database.DB.Exec(
						"INSERT INTO online_sessions (device_id, online_at) VALUES (?, ?)", devID, now)
				} else {
					sets = append(sets, "status = 'online'", "last_online_at = ?")
					args = append(args, now)
				}
			}
			args = append(args, devID)
			database.DB.Exec("UPDATE devices SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...)
		}

		if !r.Online {
			newConsecutive := devConsecutiveFailures + 1
			database.DB.Exec(
				"UPDATE devices SET last_seen = ?, consecutive_failures = ?, fail_count = ? WHERE id = ?",
				now, newConsecutive, devFailCount+1, devID)

			if newConsecutive >= s.cfg.OfflineThreshold && devStatus == "online" {
				if devLastOnlineAt != nil {
					duration := now.Sub(*devLastOnlineAt)
					database.DB.Exec(
						"UPDATE devices SET status = 'offline', last_offline_at = ?, online_duration_seconds = online_duration_seconds + ? WHERE id = ?",
						now, int64(duration.Seconds()), devID)
				} else {
					database.DB.Exec(
						"UPDATE devices SET status = 'offline', last_offline_at = ? WHERE id = ?", now, devID)
				}
				database.DB.Exec(
					"INSERT INTO device_events (device_id, device_mac, event_type, old_status, new_status, event_time, description) VALUES (?, ?, 'offline', 'online', 'offline', ?, ?)",
					devID, devMAC, now,
					fmt.Sprintf("Device %s offline (%d consecutive failures)", r.IP, newConsecutive))
				database.DB.Exec(
					"UPDATE online_sessions SET offline_at = ?, duration_sec = CAST(strftime('%s', ?) - strftime('%s', online_at) AS INTEGER) WHERE device_id = ? AND offline_at IS NULL",
					now, now, devID)

				if config.AppConfig.Alert.OfflineEnabled {
					result, _ := database.DB.Exec(
						"INSERT INTO alerts (device_id, device_mac, alert_type, level, title, content, status) VALUES (?, ?, 'device_offline', 'warning', ?, ?, 'open')",
						devID, devMAC,
						fmt.Sprintf("Device offline: %s", r.IP),
						fmt.Sprintf("Device %s (%s) offline after %d failures", r.IP, devMAC, newConsecutive))
					alertID, _ := result.LastInsertId()
					aID := uint(alertID)
					s.hub.BroadcastAlert(models.Alert{
						ID: aID, DeviceID: &devID, DeviceMAC: devMAC,
						AlertType: "device_offline", Level: "warning",
						Title: fmt.Sprintf("Device offline: %s", r.IP),
						Content: fmt.Sprintf("Device %s (%s) offline after %d failures", r.IP, devMAC, newConsecutive),
						Status: "open", CreatedAt: now,
					})
				}
			}
		}

		if oldStatus != getDevStatus(devID) || isNew {
			newStat := "online"
			if !r.Online {
				newStat = "offline"
			}
			statusChanges = append(statusChanges, websocket.DeviceStatusUpdate{
				DeviceID: devID, IP: r.IP, MAC: devMAC,
				Hostname: r.Hostname, Status: newStat, Timestamp: now.Unix(),
			})
		}
	}

	if len(statusChanges) > 0 {
		s.hub.BroadcastDeviceStatus(statusChanges)
	}
}

func getDevStatus(deviceID uint) string {
	var status string
	database.DB.QueryRow("SELECT status FROM devices WHERE id = ?", deviceID).Scan(&status)
	return status
}

func (s *ScannerService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func intsToStr(ports []int) string {
	strs := make([]string, len(ports))
	for i, p := range ports {
		strs[i] = fmt.Sprintf("%d", p)
	}
	return strings.Join(strs, ",")
}
