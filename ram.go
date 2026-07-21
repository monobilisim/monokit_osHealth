//go:build osHealth

package main

import (
	"fmt"
	"sort"
	"time"

	lib "github.com/monobilisim/monokit2/lib"
	"github.com/rs/zerolog"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

func CheckSystemRAM(logger zerolog.Logger) {
	var moduleName string = "memory"

	logger.Info().Msg("Starting RAM monitoring...")

	vm, err := mem.VirtualMemory()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get virtual memory stats")
		return
	}

	// vm.UsedPercent  this does not work while actual usage is 13% it says 90% because of cached and buffered memory
	actualUsedPercent := float64(vm.Total-vm.Available) / float64(vm.Total) * 100
	logger.Debug().Msgf("RAM usage %.2f", actualUsedPercent)

	if actualUsedPercent >= float64(lib.OsHealthConfig.RamUsageAlarm.Limit) {
		alarmMessage := fmt.Sprintf("[osHealth] - %s - RAM usage has been more than %d%% (%.2f%%) for last %d minutes", lib.GlobalConfig.Hostname, lib.OsHealthConfig.RamUsageAlarm.Limit, actualUsedPercent, lib.GlobalConfig.ZulipAlarm.Interval)

		if lib.OsHealthConfig.RamUsageAlarm.TopProcesses.Enabled {
			processes, err := process.Processes()
			var usages []ProcUsage

			if err != nil {
				logger.Error().Err(err).Msg("Failed to get processes")
			} else {

				for _, p := range processes {
					_, _ = p.MemoryPercent()
				}
				time.Sleep(time.Second)

				for _, p := range processes {
					memPercent, err := p.MemoryPercent()
					if err != nil {
						continue
					}

					cpuPercent, _ := p.CPUPercent()
					name, _ := p.Name()

					usages = append(usages, ProcUsage{
						Pid:  p.Pid,
						Name: name,
						CPU:  cpuPercent,
						RAM:  memPercent,
					})
				}

				sort.Slice(usages, func(i, j int) bool {
					return usages[i].RAM > usages[j].RAM
				})

				alarmMessage += "\n\nTop RAM consuming processes:\n\n"
				alarmMessage += "| PID | NAME | CPU% | RAM% |\n"
				alarmMessage += "| --- | --- | --- | --- |\n"
				for i, u := range usages {
					if i >= lib.OsHealthConfig.RamUsageAlarm.TopProcesses.Processes {
						break
					}
					alarmMessage += fmt.Sprintf("| %d | %s | %.2f | %.2f |\n", u.Pid, u.Name, u.CPU, u.RAM)
				}
			}
		}

		lib.SendZulipAlarm(alarmMessage, pluginName, moduleName, down)
	} else {
		lastAlarm, err := lib.GetLastZulipAlarm(pluginName, moduleName)

		if err != nil {
			lib.Logger.Error().Err(err).Msg("Failed to get last RAM alarm from database")
			return
		}

		if lastAlarm.Status == down {
			alarmMessage := fmt.Sprintf("[osHealth] - %s - RAM usage is back to normal", lib.GlobalConfig.Hostname)

			lib.SendZulipAlarm(alarmMessage, pluginName, moduleName, up)
		}
	}
}
