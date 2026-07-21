//go:build osHealth

package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	lib "github.com/monobilisim/monokit2/lib"
	"github.com/rs/zerolog"
)

func CheckSystemDiskZFS(logger zerolog.Logger) {
	var moduleName string
	var issueSubject string

	_, err := exec.LookPath("zpool")
	if err != nil {
		logger.Error().Err(err).Msg("zpool command not found")
		return
	}

	// monokit2-devel	ONLINE	0%
	out, err := exec.Command("zpool", "list", "-H", "-o", "name,health,capacity").Output()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to execute zpool command")
		return
	}

	unhealthyPools := []ZFSPoolHealth{}
	limitExceededPools := []ZFSPoolCapacity{}

	// pool1 ONLINE 0%
	// pool2 DEGRADED 0%
	lines := string(out)
	for _, line := range strings.Split(lines, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// monokit2-devel ONLINE 0% => []string{"monokit2-devel", "ONLINE", "0%"}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		poolName := fields[0]
		health := fields[1]
		capacityStr := fields[2]

		if health != "ONLINE" {
			unhealthyPool := ZFSPoolHealth{
				Name:   poolName,
				Health: health,
			}

			unhealthyPools = append(unhealthyPools, unhealthyPool)
		}

		capacityStr = strings.TrimSuffix(capacityStr, "%")
		capacity, err := strconv.Atoi(capacityStr)
		if err != nil {
			logger.Error().Err(err).Str("capacity", capacityStr).Msg("Failed to parse capacity")
			continue
		}

		logger.Debug().
			Str("pool", poolName).
			Int("capacity_percent", capacity).
			Msg("ZFS pool usage information")

		if capacity >= lib.OsHealthConfig.DiskUsageAlarm.Limit {
			limitExceededPool := ZFSPoolCapacity{
				Name:     poolName,
				Capacity: capacity,
			}

			limitExceededPools = append(limitExceededPools, limitExceededPool)
		}
	}

	moduleName = "zfsHealth"
	issueSubject = fmt.Sprintf("%s için ZFS pool(lar) sağlıklı değil", lib.GlobalConfig.Hostname)
	// one or more pools are not healthy
	if len(unhealthyPools) > 0 {
		tableHeaders := []string{"NAME", "HEALTH"}
		tableValues := [][]string{}
		for _, pool := range unhealthyPools {
			logger.Warn().Str("pool", pool.Name).Str("health", pool.Health).Msg("ZFS pool is not healthy")
			tableValues = append(tableValues, []string{pool.Name, pool.Health})
		}

		table := lib.CreateMarkdownTable(tableHeaders, tableValues)

		alarmMessage := fmt.Sprintf("[%s] - %s - One or more ZFS pools are not healthy:\n\n", pluginName, lib.GlobalConfig.Hostname)
		alarmMessage += table

		// Zulip alarm
		lib.SendZulipAlarm(alarmMessage, pluginName, moduleName, down)

		// Redmine issue
		lastIssue, err := lib.GetLastRedmineIssue(pluginName, moduleName)

		if err != nil {
			lib.Logger.Error().Err(err).Msg("Failed to get last issue from database")
			return
		}

		var issue lib.Issue

		if lastIssue.Status == up {
			issue = lib.Issue{
				Subject:    issueSubject,
				Notes:      fmt.Sprintf("Sorun devam ediyor.\n\n%s", table),
				StatusId:   lib.IssueStatus.Feedback,
				PriorityId: lib.IssuePriority.Urgent,
				Service:    pluginName,
				Module:     moduleName,
				Status:     down,
			}
		} else {
			issue = lib.Issue{
				Subject:     issueSubject,
				Description: fmt.Sprintf("%s", table),
				StatusId:    lib.IssueStatus.Feedback,
				PriorityId:  lib.IssuePriority.Urgent,
				Service:     pluginName,
				Module:      moduleName,
				Status:      down,
			}
		}

		lib.CreateRedmineIssue(issue)
	}

	// all pools are healthy now
	if len(unhealthyPools) == 0 {
		// Zulip alarm
		lastAlarm, err := lib.GetLastZulipAlarm(pluginName, moduleName)

		if err != nil {
			logger.Error().Err(err).Msg("Failed to get last alarm from database")
			return
		}

		if lastAlarm.Status == down {
			alarmMessage := fmt.Sprintf("[%s] - %s - All ZFS pools are now healthy", pluginName, lib.GlobalConfig.Hostname)

			lib.SendZulipAlarm(alarmMessage, pluginName, moduleName, up)
		}

		// Redmine issue
		lastIssue, err := lib.GetLastRedmineIssue(pluginName, moduleName)

		if err != nil {
			lib.Logger.Error().Err(err).Msg("Failed to get last issue from database")
			return
		}

		var issue lib.Issue

		if lastIssue.Status == down {
			issue = lib.Issue{
				Subject:    issueSubject,
				Notes:      fmt.Sprintf("%s için tüm ZFS poolları sağlıklı durumda.", lib.GlobalConfig.Hostname),
				StatusId:   lib.IssueStatus.Resolved,
				PriorityId: lib.IssuePriority.Urgent,
				Service:    pluginName,
				Module:     moduleName,
				Status:     up,
			}

			lib.CreateRedmineIssue(issue)
		}
	}

	moduleName = "zfsCapacity"
	issueSubject = fmt.Sprintf("%s için ZFS dataset doluluk seviyesi %d%% üstüne çıktı", lib.GlobalConfig.Hostname, lib.OsHealthConfig.DiskUsageAlarm.Limit)
	if len(limitExceededPools) > 0 {
		tableHeaders := []string{"NAME", "CAPACITY"}
		tableValues := [][]string{}
		for _, pool := range limitExceededPools {
			logger.Warn().Str("pool", pool.Name).Int("capacity", pool.Capacity).Msg("ZFS pool capacity exceeded limit")
			tableValues = append(tableValues, []string{pool.Name, fmt.Sprintf("%d%%", pool.Capacity)})
		}

		table := lib.CreateMarkdownTable(tableHeaders, tableValues)

		alarmMessage := fmt.Sprintf("[%s] - %s - One or more ZFS pools have exceeded capacity limit of %d%%:\n\n", pluginName, lib.GlobalConfig.Hostname, lib.OsHealthConfig.DiskUsageAlarm.Limit)
		alarmMessage += table

		// Zulip alarm
		lib.SendZulipAlarm(alarmMessage, pluginName, moduleName, down)

		// Redmine issue
		lastIssue, err := lib.GetLastRedmineIssue(pluginName, moduleName)

		if err != nil {
			lib.Logger.Error().Err(err).Msg("Failed to get last issue from database")
			return
		}

		var issue lib.Issue

		if lastIssue.Status == up {
			issue = lib.Issue{
				Subject:    issueSubject,
				Notes:      fmt.Sprintf("Sorun devam ediyor.\n\n%s", table),
				StatusId:   lib.IssueStatus.Feedback,
				PriorityId: lib.IssuePriority.Urgent,
				Service:    pluginName,
				Module:     moduleName,
				Status:     down,
			}
		} else {
			issue = lib.Issue{
				Subject:     issueSubject,
				Description: fmt.Sprintf("%s", table),
				StatusId:    lib.IssueStatus.Feedback,
				PriorityId:  lib.IssuePriority.Urgent,
				Service:     pluginName,
				Module:      moduleName,
				Status:      down,
			}
		}

		lib.CreateRedmineIssue(issue)
	}

	if len(limitExceededPools) == 0 {
		// Zulip alarm
		lastAlarm, err := lib.GetLastZulipAlarm(pluginName, moduleName)

		if err != nil {
			logger.Error().Err(err).Msg("Failed to get last alarm from database")
			return
		}

		if lastAlarm.Status == down {
			alarmMessage := fmt.Sprintf("[%s] - %s - All ZFS pools are now under the capacity limit of %d%%", pluginName, lib.GlobalConfig.Hostname, lib.OsHealthConfig.DiskUsageAlarm.Limit)

			lib.SendZulipAlarm(alarmMessage, pluginName, moduleName, up)
		}

		// Redmine issue
		lastIssue, err := lib.GetLastRedmineIssue(pluginName, moduleName)

		if err != nil {
			lib.Logger.Error().Err(err).Msg("Failed to get last issue from database")
			return
		}

		var issue lib.Issue

		if lastIssue.Status == down {
			issue = lib.Issue{
				Subject:    issueSubject,
				Notes:      fmt.Sprintf("%s için bütün ZFS datasetleri %d%% altına indi, kapatılıyor.", lib.GlobalConfig.Hostname, lib.OsHealthConfig.DiskUsageAlarm.Limit),
				StatusId:   lib.IssueStatus.Resolved,
				PriorityId: lib.IssuePriority.Urgent,
				Service:    pluginName,
				Module:     moduleName,
				Status:     up,
			}

			lib.CreateRedmineIssue(issue)
		}
	}
}
