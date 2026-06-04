package machineid

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

// GetMachineID returns a unique machine identifier.
// On Linux, it reads /etc/machine-id. Falls back to reading the DMI product UUID
// or generating a persistent ID.
func GetMachineID() string {
	// Try /etc/machine-id first (systemd systems)
	id, err := readFileTrimmed("/etc/machine-id")
	if err == nil && id != "" {
		return id
	}

	// Try DMI product UUID
	id, err = readFileTrimmed("/sys/class/dmi/id/product_uuid")
	if err == nil && id != "" {
		return strings.ToLower(strings.ReplaceAll(id, "-", ""))
	}

	// Try /proc/sys/kernel/random/boot_id
	id, err = readFileTrimmed("/proc/sys/kernel/random/boot_id")
	if err == nil && id != "" {
		return strings.ToLower(strings.ReplaceAll(id, "-", ""))
	}

	// Fallback: generate a random ID
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GetMACAddresses returns all non-loopback MAC addresses.
func GetMACAddresses() []string {
	macs := []string{}
	data, err := os.ReadFile("/sys/class/net")
	if err != nil {
		return macs
	}

	interfaces := strings.Fields(string(data))
	for _, iface := range interfaces {
		if iface == "lo" {
			continue
		}
		mac, err := readFileTrimmed(fmt.Sprintf("/sys/class/net/%s/address", iface))
		if err == nil && mac != "" && mac != "00:00:00:00:00:00" {
			macs = append(macs, mac)
		}
	}
	return macs
}

func readFileTrimmed(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}