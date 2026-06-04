package command

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/embedded-ops-platform/agent/pkg/shellsafe"
)

// Task represents a command task received from the server.
type Task struct {
	TaskID string                 `json:"task_id"`
	Action string                 `json:"action"`
	Params map[string]interface{} `json:"params"`
}

// Result represents the result of a command execution.
type Result struct {
	TaskID   string
	Stdout   string
	Stderr   string
	ExitCode int
	Status   string
}

// Dispatcher handles command dispatch and execution.
type Dispatcher struct {
	executor *shellsafe.SafeExecutor
}

// NewDispatcher creates a new command dispatcher.
func NewDispatcher(allowReboot bool) *Dispatcher {
	return &Dispatcher{
		executor: shellsafe.NewSafeExecutor(allowReboot),
	}
}

// Dispatch executes a command task and returns the result.
func (d *Dispatcher) Dispatch(task *Task) *Result {
	switch task.Action {
	case "get_basic_info":
		return d.getBasicInfo()
	case "get_dmesg":
		return d.getDmesg()
	case "get_usb_devices":
		return d.getUSBDevices()
	case "get_audio_status":
		return d.getAudioStatus()
	case "set_audio_volume":
		return d.setAudioVolume(task.Params)
	case "get_gpu_status":
		return d.getGPUStatus()
	case "restart_app":
		return d.restartApp(task.Params)
	case "get_wifi_status":
		return d.getWiFiStatus()
	case "upload_log_bundle":
		return d.uploadLogBundle()
	default:
		return &Result{
			TaskID:   task.TaskID,
			Stderr:   fmt.Sprintf("unknown action: %s", task.Action),
			ExitCode: -1,
			Status:   "failed",
		}
	}
}

func (d *Dispatcher) getBasicInfo() *Result {
	info := make(map[string]string)
	result := d.executor.Execute("uname", "-a")
	info["uname"] = result.Stdout

	result2 := d.executor.Execute("cat", "/etc/os-release")
	info["os_release"] = result2.Stdout

	result3 := d.executor.Execute("cat", "/proc/cpuinfo")
	info["cpu_info"] = truncateOutput(result3.Stdout, 2000)

	data, _ := json.Marshal(info)
	return &Result{
		Stdout:   string(data),
		ExitCode: 0,
		Status:   "completed",
	}
}

func (d *Dispatcher) getDmesg() *Result {
	result := d.executor.Execute("dmesg")
	return &Result{
		Stdout:   truncateOutput(result.Stdout, 10000),
		ExitCode: result.ExitCode,
		Status:   mapStatus(result.ExitCode),
	}
}

func (d *Dispatcher) getUSBDevices() *Result {
	result := d.executor.Execute("lsusb")
	return &Result{
		Stdout:   result.Stdout,
		ExitCode: result.ExitCode,
		Status:   mapStatus(result.ExitCode),
	}
}

func (d *Dispatcher) getAudioStatus() *Result {
	var b strings.Builder

	// Sound cards
	result := d.executor.Execute("cat", "/proc/asound/cards")
	b.WriteString("=== Sound Cards ===\n")
	b.WriteString(result.Stdout + "\n")

	// ALSA devices
	result = d.executor.Execute("aplay", "-l")
	if result.ExitCode == 0 {
		b.WriteString("\n=== ALSA Playback Devices ===\n")
		b.WriteString(result.Stdout + "\n")
	}

	// Mixer info
	result = d.executor.Execute("amixer", "-c", "0")
	if result.ExitCode == 0 {
		b.WriteString("\n=== Mixer Controls ===\n")
		b.WriteString(result.Stdout + "\n")
	}

	return &Result{
		Stdout:   b.String(),
		ExitCode: 0,
		Status:   "completed",
	}
}

func (d *Dispatcher) setAudioVolume(params map[string]interface{}) *Result {
	card := getParamFloat(params, "card", 0)
	control := getParamString(params, "control", "HP")
	value := getParamFloat(params, "value", 50)

	// Validate
	if card < 0 || card > 8 {
		return &Result{Stderr: "card must be 0-8", ExitCode: -1, Status: "failed"}
	}
	if value < 0 || value > 100 {
		return &Result{Stderr: "value must be 0-100", ExitCode: -1, Status: "failed"}
	}
	validControls := map[string]bool{"HP": true, "Speaker": true, "Headphone": true, "Master": true, "PCM": true}
	if !validControls[control] {
		return &Result{Stderr: fmt.Sprintf("invalid control: %s", control), ExitCode: -1, Status: "failed"}
	}

	cardStr := fmt.Sprintf("%.0f", card)
	valueStr := fmt.Sprintf("%.0f", value)

	result := d.executor.Execute("amixer", "-c", cardStr, "sset", control, valueStr+"%")
	return &Result{
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
		Status:   mapStatus(result.ExitCode),
	}
}

func (d *Dispatcher) getGPUStatus() *Result {
	var b strings.Builder

	// DRI devices
	result := d.executor.Execute("ls", "/dev/dri")
	b.WriteString("=== DRI Devices ===\n")
	b.WriteString(result.Stdout + "\n")

	// Mali devices
	result = d.executor.Execute("ls", "/dev/mali*")
	b.WriteString("\n=== Mali Devices ===\n")
	b.WriteString(result.Stdout + "\n")

	// GPU frequency
	result = d.executor.Execute("cat", "/sys/class/devfreq/*/cur_freq")
	b.WriteString("\n=== GPU Frequencies ===\n")
	b.WriteString(result.Stdout + "\n")

	return &Result{
		Stdout:   b.String(),
		ExitCode: 0,
		Status:   "completed",
	}
}

func (d *Dispatcher) restartApp(params map[string]interface{}) *Result {
	serviceName := getParamString(params, "service_name", "")
	if serviceName == "" {
		return &Result{Stderr: "service_name is required", ExitCode: -1, Status: "failed"}
	}

	validServices := map[string]bool{"agent": true, "networking": true, "audio": true, "display": true}
	if !validServices[serviceName] {
		return &Result{Stderr: fmt.Sprintf("invalid service: %s", serviceName), ExitCode: -1, Status: "failed"}
	}

	switch serviceName {
	case "agent":
		result := d.executor.Execute("systemctl", "restart", "embedded-agent")
		return &Result{Stdout: result.Stdout, ExitCode: result.ExitCode, Status: mapStatus(result.ExitCode)}
	case "networking":
		result := d.executor.Execute("systemctl", "restart", "networking")
		return &Result{Stdout: result.Stdout, ExitCode: result.ExitCode, Status: mapStatus(result.ExitCode)}
	case "audio":
		result := d.executor.Execute("systemctl", "restart", "alsa-restore")
		return &Result{Stdout: result.Stdout, ExitCode: result.ExitCode, Status: mapStatus(result.ExitCode)}
	case "display":
		result := d.executor.Execute("systemctl", "restart", "display-manager")
		return &Result{Stdout: result.Stdout, ExitCode: result.ExitCode, Status: mapStatus(result.ExitCode)}
	default:
		return &Result{Stderr: "unknown service", ExitCode: -1, Status: "failed"}
	}
}

func (d *Dispatcher) getWiFiStatus() *Result {
	var b strings.Builder

	result := d.executor.Execute("iw", "dev")
	b.WriteString("=== Wireless Devices ===\n")
	b.WriteString(result.Stdout + "\n")

	result = d.executor.Execute("ip", "link")
	b.WriteString("\n=== Network Interfaces ===\n")
	b.WriteString(result.Stdout + "\n")

	result = d.executor.Execute("lsmod")
	b.WriteString("\n=== Kernel Modules ===\n")
	for _, line := range strings.Split(result.Stdout, "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "wifi") || strings.Contains(lower, "wlan") || strings.Contains(lower, "80211") {
			b.WriteString(line + "\n")
		}
	}

	return &Result{
		Stdout:   b.String(),
		ExitCode: 0,
		Status:   "completed",
	}
}

func (d *Dispatcher) uploadLogBundle() *Result {
	// Collect system logs
	result := d.executor.Execute("journalctl", "--no-pager", "-n", "200")
	return &Result{
		Stdout:   truncateOutput(result.Stdout, 5000),
		ExitCode: result.ExitCode,
		Status:   mapStatus(result.ExitCode),
	}
}

func getParamFloat(params map[string]interface{}, key string, defaultVal float64) float64 {
	if params == nil {
		return defaultVal
	}
	if v, ok := params[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case int:
			return float64(val)
		}
	}
	return defaultVal
}

func getParamString(params map[string]interface{}, key, defaultVal string) string {
	if params == nil {
		return defaultVal
	}
	if v, ok := params[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
}

func mapStatus(exitCode int) string {
	if exitCode == 0 {
		return "completed"
	}
	return "failed"
}

func truncateOutput(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "\n... [truncated]"
	}
	return s
}