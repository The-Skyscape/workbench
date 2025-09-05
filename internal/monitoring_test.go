package internal

import (
	"testing"

	"github.com/The-Skyscape/devtools/pkg/testutils"
)

func TestSystemMonitor(t *testing.T) {
	monitor := NewSystemMonitor()
	
	// Test that monitor starts properly
	monitor.Start()
	testutils.AssertTrue(t, monitor.started)
	
	// Test getting current stats
	stats := monitor.GetCurrentStats()
	testutils.AssertNotNil(t, stats)
	
	// Test that stats have reasonable values
	testutils.AssertTrue(t, stats.CPU.UsagePercent >= 0)
	testutils.AssertTrue(t, stats.CPU.UsagePercent <= 100)
	testutils.AssertTrue(t, stats.Memory.Total > 0)
	testutils.AssertTrue(t, stats.Disk.Total > 0)
}

func TestGetSystemInfo(t *testing.T) {
	info := GetSystemInfo()
	
	// Verify system info fields are populated
	testutils.AssertNotEqual(t, "", info.OS)
	testutils.AssertNotEqual(t, "", info.Arch)
	testutils.AssertTrue(t, info.NumCPU > 0)
	testutils.AssertNotEqual(t, "", info.GoVersion)
	testutils.AssertTrue(t, info.NumGoroutine > 0)
}

func TestFormatBytes(t *testing.T) {
	testCases := []struct {
		input    uint64
		expected string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
		{1125899906842624, "1.0 PB"},
	}
	
	for _, tc := range testCases {
		result := FormatBytes(tc.input)
		testutils.AssertEqual(t, tc.expected, result)
	}
}

func TestSystemMonitorHistory(t *testing.T) {
	monitor := NewSystemMonitor()
	monitor.Start()
	
	// Get history with limit
	history := monitor.GetHistory(5)
	testutils.AssertTrue(t, len(history) <= 5)
	
	// Get all history
	allHistory := monitor.GetHistory(0)
	testutils.AssertNotNil(t, allHistory)
}