package controllers

import (
	"fmt"
	"net/http"
	"workbench/internal"

	"github.com/The-Skyscape/devtools/pkg/application"
)

// Monitoring is a factory function with the prefix and instance
func Monitoring() (string, *MonitoringController) {
	return "monitoring", &MonitoringController{
		monitor: internal.NewSystemMonitor(),
	}
}

// MonitoringController handles system monitoring
type MonitoringController struct {
	application.BaseController
	monitor *internal.SystemMonitor
}

// Setup is called when the application is started
func (c *MonitoringController) Setup(app *application.App) {
	c.BaseController.Setup(app)

	auth := app.Use("auth").(*AuthController)

	// Partial routes for HTMX auto-refresh
	http.Handle("GET /partials/stats", app.Serve("stats-partial.html", auth.Required))
	http.Handle("GET /partials/coder-status", app.Serve("coder-status-partial.html", auth.Required))

	// Start system monitoring
	c.monitor.Start()
}

// Handle is called when each request is handled
func (c MonitoringController) Handle(req *http.Request) application.Controller {
	c.Request = req
	return &c
}

// Monitoring Methods for Templates

// GetSystemStats returns current system statistics
func (c *MonitoringController) GetSystemStats() interface{} {
	if c.monitor == nil {
		return nil
	}
	return c.monitor.GetCurrentStats()
}

// GetSystemInfo returns static system information
func (c *MonitoringController) GetSystemInfo() internal.SystemInfo {
	return internal.GetSystemInfo()
}


// GetCPUUsage returns current CPU usage percentage
func (c *MonitoringController) GetCPUUsage() float64 {
	stats := c.monitor.GetCurrentStats()
	if stats == nil {
		return 0
	}
	return stats.CPU.UsagePercent
}

// GetMemoryUsage returns current memory usage percentage
func (c *MonitoringController) GetMemoryUsage() float64 {
	stats := c.monitor.GetCurrentStats()
	if stats == nil {
		return 0
	}
	return stats.Memory.UsedPercent
}


// GetLoadAverage returns system load average
func (c *MonitoringController) GetLoadAverage() string {
	stats := c.monitor.GetCurrentStats()
	if stats == nil || stats.LoadAverage.Load1 == 0 {
		return "0.00"
	}
	return fmt.Sprintf("%.2f", stats.LoadAverage.Load1)
}


// FormatBytes converts bytes to human-readable format for templates
func (c *MonitoringController) FormatBytes(bytes uint64) string {
	return internal.FormatBytes(bytes)
}

// GetDataDirStats returns disk usage stats for the persistent data directory
func (c *MonitoringController) GetDataDirStats() *internal.DataDirStats {
	return internal.GetDataDirStats()
}
