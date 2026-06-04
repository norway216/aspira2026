package logs

import (
	"fmt"
	"strconv"

	"github.com/embedded-ops-platform/agent/pkg/shellsafe"
)

// Collector collects log data from the system.
type Collector struct {
	executor *shellsafe.SafeExecutor
}

// NewCollector creates a new log collector.
func NewCollector() *Collector {
	return &Collector{
		executor: shellsafe.NewSafeExecutor(false),
	}
}

// GetDmesg gets kernel log messages.
func (c *Collector) GetDmesg(lines int) string {
	result := c.executor.Execute("dmesg")
	output := result.Stdout
	if lines > 0 && len(output) > lines*80 {
		start := len(output) - lines*80
		if start < 0 {
			start = 0
		}
		output = output[start:]
	}
	return output
}

// GetJournalctl gets systemd journal logs.
func (c *Collector) GetJournalctl(lines int) string {
	result := c.executor.Execute("journalctl", "--no-pager", "-n", strconv.Itoa(lines))
	return result.Stdout
}

// GetAppLog reads an application log file.
func (c *Collector) GetAppLog(path string, maxLines int) string {
	if maxLines <= 0 {
		maxLines = 200
	}
	result := c.executor.Execute("tail", "-n", fmt.Sprintf("%d", maxLines), path)
	return result.Stdout
}