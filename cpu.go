package main

/**
 * CPU Information
 */

import (
	linuxproc "github.com/c9s/goprocinfo/linux"
)

type CPUInfo struct {
	NumProcessors int
	Load1Min float64
	Load5Min float64
	Load1MinPercentage float64
	Load5MinPercentage float64
}

func NewCPUInfo() *CPUInfo {
	info := &CPUInfo{}

	// Read /proc/stat for overall CPU information
	stats, statErr := linuxproc.ReadStat("/proc/stat")

	if statErr == nil {
		// How many processors do we have?
		info.NumProcessors = len(stats.CPUStats)
	}

	// Read load average
	loadavg, loadErr := linuxproc.ReadLoadAvg("/proc/loadavg")

	if loadErr == nil {
		info.Load1Min = loadavg.Last1Min
		info.Load5Min = loadavg.Last5Min
	}

	// Calculate percentages
	if info.NumProcessors > 0 {
		info.Load1MinPercentage = info.Load1Min / float64(info.NumProcessors)
		info.Load5MinPercentage = info.Load5Min / float64(info.NumProcessors)
	}

	return info
}
