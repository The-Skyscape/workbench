package internal

import (
	"fmt"
	"runtime"
	"syscall"

	"github.com/The-Skyscape/devtools/pkg/database"
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

// DataDirStats represents disk usage statistics for the data directory
type DataDirStats struct {
	Path        string
	Total       uint64
	Used        uint64
	Free        uint64
	UsedPercent float64
}

// GetDataDirStats returns disk usage statistics for the persistent data directory
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


