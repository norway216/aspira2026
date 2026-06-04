package collector

import (
	"fmt"
	"os"
	"strings"

	"github.com/embedded-ops-platform/agent/pkg/shellsafe"
)

// EmbeddedInfo holds embedded-specific device information.
type EmbeddedInfo struct {
	GPUInfo   *GPUInfo   `json:"gpu_info,omitempty"`
	USBDevices []USBDevice `json:"usb_devices,omitempty"`
	WiFiInfo  *WiFiInfo  `json:"wifi_info,omitempty"`
	AudioInfo *AudioInfo `json:"audio_info,omitempty"`
}

// GPUInfo holds GPU-related information.
type GPUInfo struct {
	HasDRI      bool   `json:"has_dri"`
	HasMali     bool   `json:"has_mali"`
	CurrentFreq string `json:"current_freq,omitempty"`
	Load        string `json:"load,omitempty"`
	Renderer    string `json:"renderer,omitempty"`
}

// USBDevice holds USB device information.
type USBDevice struct {
	Bus  string `json:"bus"`
	Dev  string `json:"dev"`
	Desc string `json:"desc"`
}

// WiFiInfo holds WiFi module information.
type WiFiInfo struct {
	Interfaces []string `json:"interfaces"`
	Modules    []string `json:"modules"`
	Connected  bool     `json:"connected"`
}

// AudioInfo holds audio device information.
type AudioInfo struct {
	Cards     []string `json:"cards"`
	HasAlsa   bool     `json:"has_alsa"`
	Sinks     []string `json:"sinks,omitempty"`
}

// GetGPUInfo collects GPU-related device information.
func GetGPUInfo() *GPUInfo {
	exec := shellsafe.NewSafeExecutor(false)
	info := &GPUInfo{}

	// Check DRI devices
	if _, err := os.Stat("/dev/dri"); err == nil {
		entries, _ := os.ReadDir("/dev/dri")
		info.HasDRI = len(entries) > 0
	}

	// Check Mali devices
	if _, err := os.Stat("/dev/mali0"); err == nil {
		info.HasMali = true
	} else if _, err := os.Stat("/dev/mali"); err == nil {
		info.HasMali = true
	}

	// Get GPU frequency
	if result := exec.Execute("cat", "/sys/class/devfreq/*/cur_freq 2>/dev/null"); result.ExitCode == 0 && result.Stdout != "" {
		info.CurrentFreq = result.Stdout
	}

	// Get GPU load
	for _, path := range []string{"/sys/class/devfreq/*/load"} {
		if result := exec.Execute("cat", path); result.ExitCode == 0 && result.Stdout != "" {
			info.Load = result.Stdout
			break
		}
	}

	return info
}

// GetUSBDevices collects USB device information.
func GetUSBDevices() []USBDevice {
	exec := shellsafe.NewSafeExecutor(false)
	result := exec.Execute("lsusb")
	if result.ExitCode != 0 {
		return nil
	}

	var devices []USBDevice
	for _, line := range strings.Split(result.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 6 && fields[0] == "Bus" {
			devices = append(devices, USBDevice{
				Bus:  strings.TrimSuffix(fields[1], ":"),
				Dev:  strings.TrimSuffix(fields[3], ":"),
				Desc: strings.Join(fields[5:], " "),
			})
		}
	}
	return devices
}

// GetWiFiInfo collects WiFi module information.
func GetWiFiInfo() *WiFiInfo {
	exec := shellsafe.NewSafeExecutor(false)
	info := &WiFiInfo{}

	// WiFi interfaces
	if result := exec.Execute("iw", "dev"); result.ExitCode == 0 {
		for _, line := range strings.Split(result.Stdout, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Interface") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					info.Interfaces = append(info.Interfaces, parts[1])
				}
			}
		}
	}

	// WiFi modules
	if result := exec.Execute("lsmod"); result.ExitCode == 0 {
		for _, line := range strings.Split(result.Stdout, "\n") {
			line = strings.ToLower(line)
			if strings.Contains(line, "wifi") || strings.Contains(line, "wlan") || strings.Contains(line, "cfg80211") {
				fields := strings.Fields(line)
				if len(fields) > 0 {
					info.Modules = append(info.Modules, fields[0])
				}
			}
		}
	}

	// Check if any WiFi interface is up
	for _, iface := range info.Interfaces {
		result := exec.Execute("ip", "link", "show", iface)
		if result.ExitCode == 0 && strings.Contains(result.Stdout, "UP") {
			info.Connected = true
			break
		}
	}

	return info
}

// GetAudioInfo collects audio device information.
func GetAudioInfo() *AudioInfo {
	exec := shellsafe.NewSafeExecutor(false)
	info := &AudioInfo{}

	// Get sound cards from /proc/asound/cards
	if result := exec.Execute("cat", "/proc/asound/cards"); result.ExitCode == 0 {
		for _, line := range strings.Split(result.Stdout, "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "---") {
				info.Cards = append(info.Cards, line)
			}
		}
		info.HasAlsa = len(info.Cards) > 0
	}

	// PulseAudio sinks (if PulseAudio is running)
	if result := exec.Execute("pactl", "list", "sinks", "short"); result.ExitCode == 0 {
		sinks := strings.Split(result.Stdout, "\n")
		for _, sink := range sinks {
			if strings.TrimSpace(sink) != "" {
				info.Sinks = append(info.Sinks, strings.TrimSpace(sink))
			}
		}
	}

	return info
}

// GetGPURenderer gets the OpenGL renderer string.
func GetGPURenderer() string {
	exec := shellsafe.NewSafeExecutor(false)
	if result := exec.Execute("glxinfo"); result.ExitCode == 0 {
		for _, line := range strings.Split(result.Stdout, "\n") {
			if strings.Contains(line, "OpenGL renderer") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	}
	return ""
}

// CheckLLVMPipe checks if the system is using llvmpipe software rendering.
func CheckLLVMPipe() bool {
	renderer := GetGPURenderer()
	return strings.Contains(strings.ToLower(renderer), "llvmpipe")
}

// PrintEmbeddedInfo prints embedded device information (for debugging).
func PrintEmbeddedInfo(info *EmbeddedInfo) string {
	var b strings.Builder
	b.WriteString("=== Embedded Device Info ===\n")

	if info.GPUInfo != nil {
		b.WriteString(fmt.Sprintf("GPU DRI: %v, Mali: %v, Freq: %s\n",
			info.GPUInfo.HasDRI, info.GPUInfo.HasMali, info.GPUInfo.CurrentFreq))
	}

	if len(info.USBDevices) > 0 {
		b.WriteString(fmt.Sprintf("USB Devices: %d\n", len(info.USBDevices)))
		for _, usb := range info.USBDevices {
			b.WriteString(fmt.Sprintf("  Bus %s Dev %s: %s\n", usb.Bus, usb.Dev, usb.Desc))
		}
	}

	if info.WiFiInfo != nil {
		b.WriteString(fmt.Sprintf("WiFi: interfaces=%v, connected=%v\n",
			info.WiFiInfo.Interfaces, info.WiFiInfo.Connected))
	}

	if info.AudioInfo != nil {
		b.WriteString(fmt.Sprintf("Audio: cards=%v, sinks=%v\n", info.AudioInfo.Cards, info.AudioInfo.Sinks))
	}

	return b.String()
}