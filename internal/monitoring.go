package internal

import (
	"fmt"
	"runtime"
	"time"

	"github.com/The-Skyscape/devtools/pkg/monitoring"
)

// SystemMonitor provides system monitoring capabilities
type SystemMonitor struct {
	collector *monitoring.Collector
	started   bool
}

// NewSystemMonitor creates a new system monitor
func NewSystemMonitor() *SystemMonitor {
	return &SystemMonitor{
		collector: monitoring.NewCollector(false, 100), // Keep 100 samples
	}
}

// Start begins collecting system statistics
func (m *SystemMonitor) Start() {
	if !m.started {
		m.collector.Start()
		m.started = true
	}
}

// GetCurrentStats returns the current system statistics
func (m *SystemMonitor) GetCurrentStats() *monitoring.SystemStats {
	if !m.started {
		m.Start()
	}
	stats, _ := m.collector.GetCurrent()
	return stats
}

// GetHistory returns historical statistics
func (m *SystemMonitor) GetHistory(limit int) []monitoring.SystemStats {
	if !m.started {
		m.Start()
	}
	history := m.collector.GetHistory()
	if limit > 0 && len(history) > limit {
		return history[len(history)-limit:]
	}
	return history
}

// GetSystemInfo returns static system information
func GetSystemInfo() SystemInfo {
	return SystemInfo{
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		NumCPU:       runtime.NumCPU(),
		GoVersion:    runtime.Version(),
		NumGoroutine: runtime.NumGoroutine(),
	}
}

// SystemInfo represents static system information
type SystemInfo struct {
	OS           string
	Arch         string
	NumCPU       int
	GoVersion    string
	NumGoroutine int
}

// FormatBytes converts bytes to human-readable format
func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	if exp >= len(units) {
		exp = len(units) - 1
	}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// GetUptime returns system uptime as a formatted string
func GetUptime() string {
	stats, _ := monitoring.NewCollector(false, 1).GetCurrent()
	if stats == nil {
		return "unknown"
	}

	uptime := time.Since(stats.Timestamp)
	days := int(uptime.Hours() / 24)
	hours := int(uptime.Hours()) % 24
	minutes := int(uptime.Minutes()) % 60

	if days > 0 {
		return string(rune(days)) + "d " + string(rune(hours)) + "h " + string(rune(minutes)) + "m"
	} else if hours > 0 {
		return string(rune(hours)) + "h " + string(rune(minutes)) + "m"
	}
	return string(rune(minutes)) + "m"
}

// GetHealthStatus returns a simple health status based on resource usage
func GetHealthStatus(stats *monitoring.SystemStats) string {
	if stats == nil {
		return "unknown"
	}

	// Check critical thresholds
	if stats.CPU.UsagePercent > 90 || stats.Memory.UsedPercent > 90 || stats.Disk.UsedPercent > 90 {
		return "critical"
	}

	// Check warning thresholds
	if stats.CPU.UsagePercent > 70 || stats.Memory.UsedPercent > 70 || stats.Disk.UsedPercent > 80 {
		return "warning"
	}

	return "healthy"
}

// GetHealthBadgeClass returns the badge class for health status
func GetHealthBadgeClass(status string) string {
	switch status {
	case "critical":
		return "badge badge-error"
	case "warning":
		return "badge badge-warning"
	case "healthy":
		return "badge badge-success"
	default:
		return "badge badge-ghost"
	}
}
