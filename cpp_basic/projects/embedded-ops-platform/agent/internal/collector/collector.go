package collector

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/embedded-ops-platform/agent/pkg/shellsafe"
)

// BasicInfo holds the basic system information collected by the agent.
type BasicInfo struct {
	Hostname      string   `json:"hostname"`
	MachineID     string   `json:"machine_id"`
	MACAddresses  []string `json:"mac_addresses"`
	Arch          string   `json:"arch"`
	OSName        string   `json:"os_name"`
	KernelVersion string   `json:"kernel_version"`
	BoardType     string   `json:"board_type"`
	AgentVersion  string   `json:"agent_version"`
}

// SystemMetrics holds the current system metrics.
type SystemMetrics struct {
	Timestamp    int64   `json:"timestamp"`
	CPUUsage     float64 `json:"cpu_usage"`
	MemoryUsage  float64 `json:"memory_usage"`
	DiskUsage    float64 `json:"disk_usage"`
	LoadAvg1m    float64 `json:"load_avg_1m"`
	Temperature  float64 `json:"temperature"`
	GPULoad      float64 `json:"gpu_load"`
	NetworkRx    int64   `json:"network_rx_bytes"`
	NetworkTx    int64   `json:"network_tx_bytes"`
}

// CollectBasicInfo collects basic system information.
func CollectBasicInfo(machineID string, macAddresses []string) *BasicInfo {
	exec := shellsafe.NewSafeExecutor(false)
	info := &BasicInfo{
		MachineID:    machineID,
		MACAddresses: macAddresses,
		AgentVersion: "1.0.0",
	}

	// Hostname
	if hostname, err := os.Hostname(); err == nil {
		info.Hostname = hostname
	}

	// Architecture
	if result := exec.Execute("uname", "-m"); result.ExitCode == 0 {
		info.Arch = result.Stdout
	}

	// OS name
	if result := exec.Execute("cat", "/etc/os-release"); result.ExitCode == 0 {
		for _, line := range strings.Split(result.Stdout, "\n") {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				info.OSName = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
				break
			}
		}
	}

	// Kernel version
	if result := exec.Execute("uname", "-r"); result.ExitCode == 0 {
		info.KernelVersion = result.Stdout
	}

	// Board type detection
	info.BoardType = detectBoardType(exec)

	return info
}

// CollectMetrics collects current system metrics.
func CollectMetrics() *SystemMetrics {
	exec := shellsafe.NewSafeExecutor(false)
	metrics := &SystemMetrics{
		Timestamp: time.Now().Unix(),
	}

	// CPU usage from /proc/stat
	metrics.CPUUsage = getCPUUsage(exec)

	// Memory usage from /proc/meminfo
	metrics.MemoryUsage = getMemoryUsage(exec)

	// Disk usage from df
	metrics.DiskUsage = getDiskUsage(exec)

	// Load average from /proc/loadavg
	metrics.LoadAvg1m = getLoadAverage(exec)

	// Temperature
	metrics.Temperature = getTemperature(exec)

	// Network bytes
	metrics.NetworkRx, metrics.NetworkTx = getNetworkStats(exec)

	return metrics
}

func detectBoardType(exec *shellsafe.SafeExecutor) string {
	// Try to detect via device tree model
	if result := exec.Execute("cat", "/proc/device-tree/model"); result.ExitCode == 0 && result.Stdout != "" {
		model := strings.TrimSpace(result.Stdout)
		if strings.Contains(model, "RK3588") {
			return "RK3588"
		}
		if strings.Contains(model, "RK3576") {
			return "RK3576"
		}
		if strings.Contains(model, "RK3568") || strings.Contains(model, "RK3566") {
			return "RK3568"
		}
		if strings.Contains(model, "Jetson") || strings.Contains(model, "Orin") {
			return "Jetson"
		}
		return model
	}

	// Fallback: check /proc/cpuinfo
	if result := exec.Execute("cat", "/proc/cpuinfo"); result.ExitCode == 0 {
		for _, line := range strings.Split(result.Stdout, "\n") {
			lower := strings.ToLower(line)
			if strings.Contains(lower, "rk3588") {
				return "RK3588"
			}
			if strings.Contains(lower, "rk3576") {
				return "RK3576"
			}
			if strings.Contains(lower, "rk3568") {
				return "RK3568"
			}
		}
	}

	return "generic"
}

func getCPUUsage(exec *shellsafe.SafeExecutor) float64 {
	result := exec.Execute("cat", "/proc/stat")
	if result.ExitCode != 0 {
		return 0
	}

	lines := strings.Split(result.Stdout, "\n")
	if len(lines) < 1 {
		return 0
	}

	fields := strings.Fields(lines[0])
	if len(fields) < 5 {
		return 0
	}

	total := 0.0
	idle := 0.0
	for i, f := range fields {
		if i == 0 {
			continue
		}
		val := 0.0
		fmt.Sscanf(f, "%f", &val)
		total += val
		if i == 4 { // idle
			idle = val
		}
	}

	if total == 0 {
		return 0
	}
	return ((total - idle) / total) * 100
}

func getMemoryUsage(exec *shellsafe.SafeExecutor) float64 {
	result := exec.Execute("cat", "/proc/meminfo")
	if result.ExitCode != 0 {
		return 0
	}

	total := 0.0
	available := 0.0

	for _, line := range strings.Split(result.Stdout, "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			fmt.Sscanf(line, "MemTotal: %f", &total)
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			fmt.Sscanf(line, "MemAvailable: %f", &available)
		}
	}

	if total == 0 {
		return 0
	}
	return ((total - available) / total) * 100
}

func getDiskUsage(exec *shellsafe.SafeExecutor) float64 {
	result := exec.Execute("df", "/")
	if result.ExitCode != 0 {
		return 0
	}

	lines := strings.Split(result.Stdout, "\n")
	if len(lines) < 2 {
		return 0
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 5 {
		return 0
	}

	pct := strings.TrimSuffix(fields[4], "%")
	val := 0.0
	fmt.Sscanf(pct, "%f", &val)
	return val
}

func getLoadAverage(exec *shellsafe.SafeExecutor) float64 {
	result := exec.Execute("cat", "/proc/loadavg")
	if result.ExitCode != 0 {
		return 0
	}

	fields := strings.Fields(result.Stdout)
	if len(fields) < 1 {
		return 0
	}

	val := 0.0
	fmt.Sscanf(fields[0], "%f", &val)
	return val
}

func getTemperature(exec *shellsafe.SafeExecutor) float64 {
	// Try thermal zones
	basePath := "/sys/class/thermal"
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return 0
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "thermal_zone") {
			typePath := fmt.Sprintf("%s/%s/type", basePath, name)
			tempPath := fmt.Sprintf("%s/%s/temp", basePath, name)

			typeData, err := os.ReadFile(typePath)
			if err != nil {
				continue
			}

			zoneType := strings.TrimSpace(string(typeData))
			// Prefer CPU-thermal or soc-thermal
			if strings.Contains(zoneType, "cpu-thermal") || strings.Contains(zoneType, "soc-thermal") {
				tempData, err := os.ReadFile(tempPath)
				if err != nil {
					continue
				}
				temp := 0.0
				fmt.Sscanf(strings.TrimSpace(string(tempData)), "%f", &temp)
				return temp / 1000.0
			}
		}
	}

	// Fallback: return first valid temperature
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "thermal_zone") {
			tempPath := fmt.Sprintf("%s/%s/temp", basePath, name)
			tempData, err := os.ReadFile(tempPath)
			if err != nil {
				continue
			}
			temp := 0.0
			fmt.Sscanf(strings.TrimSpace(string(tempData)), "%f", &temp)
			if temp > 0 {
				return temp / 1000.0
			}
		}
	}

	return 0
}

func getNetworkStats(exec *shellsafe.SafeExecutor) (rx, tx int64) {
	result := exec.Execute("cat", "/proc/net/dev")
	if result.ExitCode != 0 {
		return 0, 0
	}

	lines := strings.Split(result.Stdout, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Inter-") || strings.HasPrefix(line, " face") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}

		iface := strings.TrimSuffix(fields[0], ":")
		if iface == "lo" {
			continue
		}

		// Sum across all non-loopback interfaces
		r := int64(0)
		t := int64(0)
		fmt.Sscanf(fields[1], "%d", &r)
		fmt.Sscanf(fields[9], "%d", &t)
		rx += r
		tx += t
	}

	return rx, tx
}