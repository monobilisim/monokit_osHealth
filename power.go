package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	lib "github.com/monobilisim/monokit2/lib"
	"github.com/rs/zerolog"
	"github.com/shirou/gopsutil/v4/host"
)

func CheckSystemPowerHealth(logger zerolog.Logger) {
	logger.Info().Msg("Starting power health monitoring...")

	moduleName := "power"

	status, err := getPowerStatus()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get power status")
		return
	}

	if status.Action != "none" {
		alarmMessage := fmt.Sprintf("[osHealth] - %s - System scheduled for %s at %s. Uptime: %s", lib.GlobalConfig.Hostname, status.Action, status.ScheduledAt, status.Uptime)

		lastAlarm, err := lib.GetLastZulipAlarm(pluginName, moduleName)
		if err != nil {
			logger.Error().Err(err).Msg(fmt.Sprintf("Could not get last alarm from module %s", moduleName))
		}

		var alarmStatus string
		// bypass n last same status alarm rule
		if lastAlarm.Status == down {
			alarmStatus = up
		} else {
			alarmStatus = down
		}

		// if last alarm was sent in less than 10 minutes skip
		if time.Since(lastAlarm.CreatedAt) < time.Duration(10)*time.Minute {
			logger.Info().Msgf("Last power alarm sent less than 10 minutes ago, skipping...")
			return
		}

		err = lib.SendZulipAlarm(alarmMessage, pluginName, moduleName, alarmStatus)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to send Zulip alarm")
		}
	}
}

func getPowerStatus() (*PowerStatus, error) {
	status := &PowerStatus{
		Action: "none",
	}

	uptime, err := host.Uptime()
	if err != nil {
		return nil, err
	}

	duration := time.Duration(uptime) * time.Second
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60

	if hours > 24 {
		days := hours / 24
		hours = hours % 24
		status.Uptime = fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else {
		status.Uptime = fmt.Sprintf("%dh %dm", hours, minutes)
	}

	switch runtime.GOOS {
	case "linux":
		if err := checkLinuxPowerStatus(status); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("Unsupported OS")
	}

	return status, nil
}

func checkLinuxPowerStatus(status *PowerStatus) error {
	scheduledFile := "/run/systemd/shutdown/scheduled"
	data, err := os.ReadFile(scheduledFile)
	if err == nil {
		lines := strings.Split(string(data), "\n")
		var usec int64
		var mode string
		for _, line := range lines {
			if strings.HasPrefix(line, "USEC=") {
				usecStr := strings.TrimPrefix(line, "USEC=")
				usec, _ = strconv.ParseInt(usecStr, 10, 64)
			}
			if strings.HasPrefix(line, "MODE=") {
				mode = strings.TrimPrefix(line, "MODE=")
			}
		}

		if usec > 0 {
			status.ScheduledAt = time.Unix(usec/1000000, 0).Format(time.RFC3339)
			status.Action = mode
			if status.Action == "" {
				status.Action = "shutdown"
			}
			return nil
		}
	}
	return nil
}

func checkWindowsPowerStatus(status *PowerStatus) error {
	cmd := exec.Command("schtasks", "/query", "/fo", "LIST", "/v")
	output, err := cmd.Output()
	if err == nil {
		outputStr := string(output)
		if strings.Contains(strings.ToLower(outputStr), "shutdown") {
			status.Action = "shutdown"
		} else if strings.Contains(strings.ToLower(outputStr), "reboot") {
			status.Action = "restart"
		}
	}
	return nil
}
