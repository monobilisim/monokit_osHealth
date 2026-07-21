//go:build osHealth

package main

import (
	"fmt"
	"sort"
	"time"

	lib "github.com/monobilisim/monokit2/lib"
	"github.com/rs/zerolog"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/process"
)

func CheckSystemLoad(logger zerolog.Logger) {
	var moduleName string = "sysload"

	logger.Info().Msg("Starting System Load monitoring...")

	loadAverage, err := load.Avg()

	if err != nil {
		logger.Error().Err(err).Msg("Failed to get load average")
	}

	// Get the number of physical CPU cores NOT LOGICAL
	cpuCores, err := cpu.Counts(false)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get CPU core count")
	}

	loadLimit := lib.OsHealthConfig.SystemLoadAlarm.LimitMultiplier * float64(cpuCores)

	if loadAverage.Load1 >= loadLimit {
		stringifiedLoadLimit := fmt.Sprintf("%.2f", loadLimit)
		stringifiedLoad := fmt.Sprintf("%.2f", loadAverage.Load1)
		stringifiedInterval := fmt.Sprintf("%d", lib.GlobalConfig.ZulipAlarm.Interval)

		alarmMessage := "[osHealth] - " + lib.GlobalConfig.Hostname + " - System load has been more than " + stringifiedLoadLimit + " (" + stringifiedLoad + ")" + " for last " + stringifiedInterval + " minutes"

		if lib.OsHealthConfig.SystemLoadAlarm.TopProcesses.Enabled {
			processes, err := process.Processes()
			var usages []ProcUsage

			if err != nil {
				logger.Error().Err(err).Msg("Failed to get processes")
			}

			if err == nil {

				for _, p := range processes {
					_, _ = p.CPUPercent()
				}
				time.Sleep(time.Second)

				for _, p := range processes {
					cpu, err := p.CPUPercent()
					if err != nil {
						continue
					}

					mem, err := p.MemoryPercent()
					if err != nil {
						continue
					}

					name, _ := p.Name()

					usages = append(usages, ProcUsage{
						Pid:  p.Pid,
						Name: name,
						CPU:  cpu,
						RAM:  mem,
					})
				}

				sort.Slice(usages, func(i, j int) bool {
					return usages[i].CPU > usages[j].CPU
				})

				alarmMessage += "\n\nTop CPU consuming processes:\n\n"
				alarmMessage += "| PID | NAME | CPU% | RAM% |\n"
				alarmMessage += "| --- | --- | --- | --- |\n"
				for i, u := range usages {
					if i >= lib.OsHealthConfig.SystemLoadAlarm.TopProcesses.Processes {
						break
					}
					alarmMessage += fmt.Sprintf("| %d | %s | %.2f | %.2f |\n", u.Pid, u.Name, u.CPU, u.RAM)
				}
			}
		}

		err := lib.SendZulipAlarm(alarmMessage, pluginName, moduleName, down)

		lastIssue, err := lib.GetLastRedmineIssue(pluginName, moduleName)

		if err != nil {
			lib.Logger.Error().Err(err).Msg("Failed to get last issue from database")
			return
		}

		var issue lib.Issue

		if lastIssue.Status == up {
			issue = lib.Issue{
				ProjectIdentifier: lib.GlobalConfig.ProjectIdentifier,
				Hostname:          lib.GlobalConfig.Hostname,
				Subject:           fmt.Sprintf("%s için sistem yükü %.2f üstüne çıktı", lib.GlobalConfig.Hostname, loadLimit),
				Notes:             fmt.Sprintf("Sorun devam ediyor, sistem yükü %.2f", loadAverage.Load1),
				StatusId:          lib.IssueStatus.Feedback,
				PriorityId:        lib.IssuePriority.Urgent,
				Service:           pluginName,
				Module:            moduleName,
				Status:            down,
			}
		} else {
			issue = lib.Issue{
				ProjectIdentifier: lib.GlobalConfig.ProjectIdentifier,
				Hostname:          lib.GlobalConfig.Hostname,
				Subject:           fmt.Sprintf("%s için sistem yükü %.2f üstüne çıktı", lib.GlobalConfig.Hostname, loadLimit),
				Description:       alarmMessage,
				StatusId:          lib.IssueStatus.Feedback,
				PriorityId:        lib.IssuePriority.Urgent,
				Service:           pluginName,
				Module:            moduleName,
				Status:            down,
			}
		}

		lib.CreateRedmineIssue(issue)
	} else {
		lastIssue, err := lib.GetLastRedmineIssue(pluginName, moduleName)

		if err != nil {
			lib.Logger.Error().Err(err).Msg("Failed to get last issue from database")
			return
		}

		lastAlarm, err := lib.GetLastZulipAlarm(pluginName, moduleName)

		if err != nil {
			lib.Logger.Error().Err(err).Msg("Failed to get last alarm from database")
			return
		}

		alarmMessage := fmt.Sprintf("[osHealth] - %s - System load is back to normal", lib.GlobalConfig.Hostname)

		if lastIssue.Status == down {
			issue := lib.Issue{
				ProjectIdentifier: lib.GlobalConfig.ProjectIdentifier,
				Hostname:          lib.GlobalConfig.Hostname,
				Subject:           fmt.Sprintf("%s için sistem yükü %.2f üstüne çıktı", lib.GlobalConfig.Hostname, loadLimit),
				Notes:             fmt.Sprintf("Sistem yükü normale döndü (%.2f)", loadAverage.Load1),
				PriorityId:        lib.IssuePriority.Urgent,
				StatusId:          lib.IssueStatus.Closed,
				Service:           pluginName,
				Module:            moduleName,
				Status:            up,
			}

			lib.Logger.Debug().Msgf("Creating Redmine issue: %+v", issue)

			lib.CreateRedmineIssue(issue)
		}

		if lastAlarm.Status == down {
			lib.SendZulipAlarm(alarmMessage, pluginName, moduleName, up)
		}
	}
}
