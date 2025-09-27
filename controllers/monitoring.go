package controllers

import (
	"fmt"
	"net/http"
	"runtime"
	"syscall"

	"github.com/The-Skyscape/devtools/pkg/application"
	"github.com/The-Skyscape/devtools/pkg/containers"
	"github.com/The-Skyscape/devtools/pkg/database"
)

// Monitoring is a factory function that returns the controller prefix and instance.
// Creates a system monitor that samples CPU, memory, and disk usage at regular intervals.
// The prefix "monitoring" makes methods available in templates as {{monitoring.MethodName}}.
func Monitoring() (string, *MonitoringController) {
	return "monitoring", &MonitoringController{
		collector: containers.NewCollector(false, 100), // Keep 100 samples
	}
}

// MonitoringController provides real-time system monitoring capabilities.
// It tracks CPU usage, memory consumption, disk usage (specifically the data directory),
// and system load averages. The monitor runs in a background goroutine and maintains
// a sliding window of samples for trend analysis.
type MonitoringController struct {
	application.Controller
	collector *containers.Collector
}

// Setup initializes the monitoring controller during application startup.
// Registers HTMX partial routes for auto-refreshing dashboard components
// and starts the background system monitor that collects metrics every 2 seconds.
// Routes registered:
// - GET /health - Health check endpoint
// - GET /partials/stats - System statistics partial (CPU, memory, disk)
// - GET /partials/coder-status - VS Code server status partial
func (c *MonitoringController) Setup(app *application.App) {
	c.Controller.Setup(app)

	auth := app.Use("auth").(*AuthController)

	http.Handle("GET /health", app.ProtectFunc(c.healthCheck, auth.Optional))

	// Partial routes for HTMX auto-refresh
	http.Handle("GET /partials/stats", app.Serve("stats-partial.html", auth.Required))
	http.Handle("GET /partials/coder-status", app.Serve("coder-status-partial.html", auth.Required))

	// Start system monitoring
	go c.collector.Start()
}

// Handle prepares the controller for request-specific operations.
// Called for each HTTP request to set the request context.
// Returns a pointer to the controller for use in template rendering.
func (c MonitoringController) Handle(req *http.Request) application.Handler {
	c.Request = req
	return &c
}

// ============================================================================
// Template Helper Methods - Accessible in views as {{monitoring.MethodName}}
// ============================================================================

// GetSystemStats returns comprehensive system statistics including CPU, memory,
// disk usage, and load averages. Returns the most recent sample from the monitor.
// Template usage: {{with monitoring.GetSystemStats}}...{{end}}
func (c *MonitoringController) GetSystemStats() *containers.SystemStats {
	if c.collector == nil {
		return nil
	}
	stats, _ := c.collector.GetCurrent()
	return stats
}

// GetSystemInfo returns static system information like hostname, OS, architecture,
// CPU count, and total memory. This information doesn't change during runtime.
// Template usage: {{monitoring.GetSystemInfo.Hostname}}
func (c *MonitoringController) GetSystemInfo() map[string]any {
	return map[string]any{
		"OS":           runtime.GOOS,
		"Arch":         runtime.GOARCH,
		"NumCPU":       runtime.NumCPU(),
		"GoVersion":    runtime.Version(),
		"NumGoroutine": runtime.NumGoroutine(),
	}
}

// GetCPUUsage returns the current CPU usage as a percentage (0-100).
// Returns 0 if monitoring data is unavailable.
// Template usage: {{monitoring.GetCPUUsage}}%
func (c *MonitoringController) GetCPUUsage() float64 {
	stats := c.GetSystemStats()
	if stats == nil {
		return 0
	}
	return stats.CPU.UsagePercent
}

// GetMemoryUsage returns the current memory usage as a percentage (0-100).
// Calculated as (used memory / total memory) * 100.
// Template usage: {{monitoring.GetMemoryUsage}}%
func (c *MonitoringController) GetMemoryUsage() float64 {
	stats := c.GetSystemStats()
	if stats == nil {
		return 0
	}
	return stats.Memory.UsedPercent
}

// GetLoadAverage returns the 1-minute load average as a formatted string.
// Load average indicates system load relative to CPU cores.
// Values > CPU count suggest high load.
// Template usage: Load: {{monitoring.GetLoadAverage}}
func (c *MonitoringController) GetLoadAverage() string {
	stats := c.GetSystemStats()
	if stats == nil || stats.LoadAverage.Load1 == 0 {
		return "0.00"
	}
	return fmt.Sprintf("%.2f", stats.LoadAverage.Load1)
}

// FormatBytes converts bytes to human-readable format (KB, MB, GB, TB).
// Uses binary prefixes (1024-based) for accurate representation.
// Template usage: {{monitoring.FormatBytes .Memory.Total}}
func (c *MonitoringController) FormatBytes(bytes uint64) string {
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

// GetDataDirStats returns disk usage statistics for the persistent data directory.
// This tracks only data that persists between container restarts (repos, database, etc.),
// NOT the system disk. Shows used/total space and percentage utilization.
// Template usage: {{with monitoring.GetDataDirStats}}...{{end}}
func (c *MonitoringController) GetDataDirStats() map[string]any {
	dataDir := database.DataDir()

	var stat syscall.Statfs_t
	err := syscall.Statfs(dataDir, &stat)
	if err != nil {
		// Return empty stats on error
		return map[string]any{}
	}

	// Calculate sizes in bytes
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	used := total - free

	usedPercent := 0.0
	if total > 0 {
		usedPercent = float64(used) / float64(total) * 100.0
	}

	return map[string]any{
		"Path":        dataDir,
		"Total":       total,
		"Used":        used,
		"Free":        free,
		"UsedPercent": usedPercent,
	}
}

func (c *MonitoringController) healthCheck(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "online")
}
