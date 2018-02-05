package main

/**
 * Laptop Battery Info
 */

import (
	"strconv"
	"strings"
	"time"
)

type BatteryInfo struct {
	ColorizedGauge    string
	ColorizedTimeLeft string
	Gauge             string
	TimeLeft          time.Duration
	IsCharging        bool
	Percent           int
}

func NewBatteryInfo() (*BatteryInfo, error) {
	// Load battery info
	output, _, err := execAndGetOutput("ibam-battery-prompt", nil, "-p")

	if err == nil {
		// Parse the output
		lines := strings.Split(output, "\n")

		info := &BatteryInfo{}

		if len(lines) > 0 {
			info.ColorizedGauge = strings.TrimSpace(lines[0])
			info.Gauge = stripANSI(info.ColorizedGauge)
		}

		if len(lines) > 1 {
			info.ColorizedTimeLeft = strings.TrimSpace(lines[1])

			// get the time into something we can parse as a duration
			timeLeft := stripANSI(info.ColorizedTimeLeft)
			timeLeft = strings.Replace(timeLeft, ":", "h", 1) + "m"

			info.TimeLeft, _ = time.ParseDuration(timeLeft)
		}

		if len(lines) > 2 {
			info.IsCharging, _ = strconv.ParseBool(strings.TrimSpace(lines[2]))
		}

		if len(lines) >= 4 {
			info.Percent, _ = strconv.Atoi(strings.TrimSpace(lines[4]))
		}

		return info, nil
	} else {
		return nil, err
	}
}
