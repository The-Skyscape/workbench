package internal

import (
	"fmt"
	"runtime"
	"syscall"

	"github.com/The-Skyscape/devtools/pkg/containers"
	"github.com/The-Skyscape/devtools/pkg/database"
)

// SystemMonitor wraps the devtools monitoring collector to provide
// system statistics for the workbench dashboard. It collects CPU, memory,
// disk, and load average metrics at 2-second intervals, maintaining
// a rolling window of 100 samples (3.3 minutes of history).
type SystemMonitor struct {
	collector *containers.Collector
	started   bool
}

// NewSystemMonitor creates a system monitor instance configured for workbench.
// The monitor is created but not started - call Start() to begin collection.
// Keeps 100 samples in memory for trend visualization.
func NewSystemMonitor() *SystemMonitor {
	return &SystemMonitor{
		collector: containers.NewCollector(false, 100), // Keep 100 samples
	}
}

// Start begins background collection of system statistics.
// Safe to call multiple times - subsequent calls are no-ops.
// Statistics are collected every 2 seconds in a goroutine.
func (m *SystemMonitor) Start() {
	if !m.started {
		m.collector.Start()
		m.started = true
	}
}

// GetCurrentStats returns the most recent system statistics sample.
// Automatically starts the monitor if not already running.
// Returns nil if no data is available yet (rare, only immediately after start).
func (m *SystemMonitor) GetCurrentStats() *containers.SystemStats {
	if !m.started {
		m.Start()
	}
	stats, _ := m.collector.GetCurrent()
	return stats
}

// GetHistory returns historical statistics samples for trend analysis.
// Parameters:
//   - limit: Maximum number of samples to return (0 = all samples)
//
// Returns newest samples first. Used for generating charts and graphs.
func (m *SystemMonitor) GetHistory(limit int) []containers.SystemStats {
	if !m.started {
		m.Start()
	}
	history := m.collector.GetHistory()
	if limit > 0 && len(history) > limit {
		return history[len(history)-limit:]
	}
	return history
}

// GetSystemInfo returns static system information that doesn't change during runtime.
// Includes OS, architecture, CPU count, Go version, and active goroutines.
// Used in dashboard header to show environment details.
func GetSystemInfo() SystemInfo {
	return SystemInfo{
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		NumCPU:       runtime.NumCPU(),
		GoVersion:    runtime.Version(),
		NumGoroutine: runtime.NumGoroutine(),
	}
}

// SystemInfo contains static system properties displayed in the dashboard.
// These values are determined at startup and remain constant.
type SystemInfo struct {
	OS           string
	Arch         string
	NumCPU       int
	GoVersion    string
	NumGoroutine int
}

// FormatBytes converts bytes to human-readable format using binary prefixes.
// Uses 1024-based units (KiB, MiB, GiB) for accurate representation.
// Examples:
//   - 1024 bytes → "1.0 KB"
//   - 1048576 bytes → "1.0 MB"
//   - 1073741824 bytes → "1.0 GB"
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

// DataDirStats contains disk usage information for the persistent data directory.
// This tracks only the data that survives container restarts (repositories, database),
// NOT the ephemeral system disk. Critical for monitoring persistent storage limits.
type DataDirStats struct {
	Path        string
	Total       uint64
	Used        uint64
	Free        uint64
	UsedPercent float64
}

// GetDataDirStats calculates disk usage for the persistent data directory.
// Uses syscall.Statfs to get filesystem statistics directly from the kernel.
// The data directory path is determined by the devtools database package.
//
// Returns:
//   - Total, used, and free space in bytes
//   - Usage percentage for progress bars
//   - Empty stats if filesystem query fails
//
// This monitors ~/.skyscape/ or similar persistent volume mount.
func GetDataDirStats() *DataDirStats {
	dataDir := database.DataDir()
	
	var stat syscall.Statfs_t
	err := syscall.Statfs(dataDir, &stat)
	if err != nil {
		// Return empty stats on error
		return &DataDirStats{
			Path: dataDir,
		}
	}
	
	// Calculate sizes in bytes
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	used := total - free
	
	usedPercent := 0.0
	if total > 0 {
		usedPercent = float64(used) / float64(total) * 100.0
	}
	
	return &DataDirStats{
		Path:        dataDir,
		Total:       total,
		Used:        used,
		Free:        free,
		UsedPercent: usedPercent,
	}
}


