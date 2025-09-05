package controllers

import (
	"fmt"
	"net/http"
	"workbench/internal"

	"github.com/The-Skyscape/devtools/pkg/application"
)

// Monitoring is a factory function that returns the controller prefix and instance.
// Creates a system monitor that samples CPU, memory, and disk usage at regular intervals.
// The prefix "monitoring" makes methods available in templates as {{monitoring.MethodName}}.
func Monitoring() (string, *MonitoringController) {
	return "monitoring", &MonitoringController{
		monitor: internal.NewSystemMonitor(),
	}
}

// MonitoringController provides real-time system monitoring capabilities.
// It tracks CPU usage, memory consumption, disk usage (specifically the data directory),
// and system load averages. The monitor runs in a background goroutine and maintains
// a sliding window of samples for trend analysis.
type MonitoringController struct {
	application.BaseController
	monitor *internal.SystemMonitor
}

// Setup initializes the monitoring controller during application startup.
// Registers HTMX partial routes for auto-refreshing dashboard components
// and starts the background system monitor that collects metrics every 2 seconds.
// Routes registered:
// - GET /partials/stats - System statistics partial (CPU, memory, disk)
// - GET /partials/coder-status - VS Code server status partial
func (c *MonitoringController) Setup(app *application.App) {
	c.BaseController.Setup(app)

	auth := app.Use("auth").(*AuthController)

	// Partial routes for HTMX auto-refresh
	http.Handle("GET /partials/stats", app.Serve("stats-partial.html", auth.Required))
	http.Handle("GET /partials/coder-status", app.Serve("coder-status-partial.html", auth.Required))

	// Start system monitoring
	c.monitor.Start()
}

// Handle prepares the controller for request-specific operations.
// Called for each HTTP request to set the request context.
// Returns a pointer to the controller for use in template rendering.
func (c MonitoringController) Handle(req *http.Request) application.Controller {
	c.Request = req
	return &c
}

// ============================================================================
// Template Helper Methods - Accessible in views as {{monitoring.MethodName}}
// ============================================================================

// GetSystemStats returns comprehensive system statistics including CPU, memory,
// disk usage, and load averages. Returns the most recent sample from the monitor.
// Template usage: {{with monitoring.GetSystemStats}}...{{end}}
func (c *MonitoringController) GetSystemStats() interface{} {
	if c.monitor == nil {
		return nil
	}
	return c.monitor.GetCurrentStats()
}

// GetSystemInfo returns static system information like hostname, OS, architecture,
// CPU count, and total memory. This information doesn't change during runtime.
// Template usage: {{monitoring.GetSystemInfo.Hostname}}
func (c *MonitoringController) GetSystemInfo() internal.SystemInfo {
	return internal.GetSystemInfo()
}


// GetCPUUsage returns the current CPU usage as a percentage (0-100).
// Returns 0 if monitoring data is unavailable.
// Template usage: {{monitoring.GetCPUUsage}}%
func (c *MonitoringController) GetCPUUsage() float64 {
	stats := c.monitor.GetCurrentStats()
	if stats == nil {
		return 0
	}
	return stats.CPU.UsagePercent
}

// GetMemoryUsage returns the current memory usage as a percentage (0-100).
// Calculated as (used memory / total memory) * 100.
// Template usage: {{monitoring.GetMemoryUsage}}%
func (c *MonitoringController) GetMemoryUsage() float64 {
	stats := c.monitor.GetCurrentStats()
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
	stats := c.monitor.GetCurrentStats()
	if stats == nil || stats.LoadAverage.Load1 == 0 {
		return "0.00"
	}
	return fmt.Sprintf("%.2f", stats.LoadAverage.Load1)
}


// FormatBytes converts bytes to human-readable format (KB, MB, GB, TB).
// Uses binary prefixes (1024-based) for accurate representation.
// Template usage: {{monitoring.FormatBytes .Memory.Total}}
func (c *MonitoringController) FormatBytes(bytes uint64) string {
	return internal.FormatBytes(bytes)
}

// GetDataDirStats returns disk usage statistics for the persistent data directory.
// This tracks only data that persists between container restarts (repos, database, etc.),
// NOT the system disk. Shows used/total space and percentage utilization.
// Template usage: {{with monitoring.GetDataDirStats}}...{{end}}
func (c *MonitoringController) GetDataDirStats() *internal.DataDirStats {
	return internal.GetDataDirStats()
}
