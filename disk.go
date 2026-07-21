//go:build osHealth

package main

import (
	"fmt"
	"math"
	"slices"
	"strconv"

	lib "github.com/monobilisim/monokit2/lib"
	"github.com/rs/zerolog"
	"github.com/shirou/gopsutil/v4/disk"
)

// removed "zfs" filesystem type because it is handled in zfs.go
var supportedFilesystems = []string{"ext4", "ext3", "ext2", "xfs", "btrfs", "fat32", "vfat"}

func CheckSystemDisk(logger zerolog.Logger) {
	var moduleName string = "disk"

	logger.Info().Msg("Starting Disk Usage monitoring...")

	if !lib.OsHealthConfig.DiskUsageAlarm.Enabled {
		logger.Debug().Msg("Disk usage alarm is disabled")
		return
	}

	diskPartitions, err := disk.Partitions(true)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get disk partitions")
		return
	}

	var exceededDiskInfos []DiskInfo

	for _, partition := range diskPartitions {
		if !slices.Contains(supportedFilesystems, partition.Fstype) {
			continue
		}

		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			logger.Error().Err(err).Str("mountpoint", partition.Mountpoint).Msg("Failed to get disk usage")
			continue
		}

		logger.Debug().
			Str("mountpoint", partition.Mountpoint).
			Float64("usage_percent", math.Round(usage.UsedPercent)).
			Msg("Disk usage information")

		if usage.UsedPercent > float64(lib.OsHealthConfig.DiskUsageAlarm.Limit) {
			diskInfo := DiskInfo{
				Device:     partition.Device,
				Mountpoint: partition.Mountpoint,
				Used:       formatBytes(usage.Used),
				Total:      formatBytes(usage.Total),
				UsedPct:    usage.UsedPercent,
				Fstype:     partition.Fstype,
			}
			exceededDiskInfos = append(exceededDiskInfos, diskInfo)
		}
	}

	if len(exceededDiskInfos) > 0 {
		alarmMessage := "[osHealth] - " + lib.GlobalConfig.Hostname + " - Disk usage exceeded " + strconv.Itoa(lib.OsHealthConfig.DiskUsageAlarm.Limit) + "% on the following partitions:\n\n"
		alarmMessage += "| Device | Mount | Usage | Total |\n"
		alarmMessage += "| --- | --- | --- | --- |\n"

		for _, diskInfo := range exceededDiskInfos {
			alarmMessage += fmt.Sprintf("| %s | %s | %.1f%% | (%s/%s) |\n", diskInfo.Device, diskInfo.Mountpoint, diskInfo.UsedPct, diskInfo.Used, diskInfo.Total)
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
				Subject:           fmt.Sprintf("%s için disk doluluk seviyesi %d%% üstüne çıktı", lib.GlobalConfig.Hostname, lib.OsHealthConfig.DiskUsageAlarm.Limit),
				Notes:             fmt.Sprintf("Sorun devam ediyor."),
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
				Subject:           fmt.Sprintf("%s için disk doluluk seviyesi %d%% üstüne çıktı", lib.GlobalConfig.Hostname, lib.OsHealthConfig.DiskUsageAlarm.Limit),
				Description:       alarmMessage,
				StatusId:          lib.IssueStatus.Feedback,
				PriorityId:        lib.IssuePriority.Urgent,
				Service:           pluginName,
				Module:            moduleName,
				Status:            down,
			}
		}

		err = lib.CreateRedmineIssue(issue)
	} else {
		lastIssue, err := lib.GetLastRedmineIssue(pluginName, moduleName)

		if err != nil {
			lib.Logger.Error().Err(err).Msg("Failed to get last issue from database")
			return
		}

		lastAlarm, err := lib.GetLastZulipAlarm(pluginName, moduleName)

		if err != nil {
			logger.Error().Err(err).Msg("Failed to get last alarm from database")
			return
		}

		if lastIssue.Status == down {
			issue := lib.Issue{
				ProjectIdentifier: lib.GlobalConfig.ProjectIdentifier,
				Hostname:          lib.GlobalConfig.Hostname,
				Subject:           fmt.Sprintf("%s için disk doluluk seviyesi %d%% üstüne çıktı", lib.GlobalConfig.Hostname, lib.OsHealthConfig.DiskUsageAlarm.Limit),
				Notes:             "Disk doluluğu limitin altına indi",
				PriorityId:        lib.IssuePriority.Urgent,
				StatusId:          lib.IssueStatus.Closed,
				Service:           pluginName,
				Module:            moduleName,
				Status:            up,
			}

			lib.Logger.Debug().Msgf("Creating Redmine issue: %+v", issue)

			err = lib.CreateRedmineIssue(issue)
		}

		if lastAlarm.Status == down {
			alarmMessage := "[osHealth] - " + lib.GlobalConfig.Hostname + " - All disk partitions are now under the limit of " + strconv.Itoa(lib.OsHealthConfig.DiskUsageAlarm.Limit) + "%"

			lib.SendZulipAlarm(alarmMessage, pluginName, moduleName, up)
		}
	}
}

// formatBytes converts bytes to human readable format
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
