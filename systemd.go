//go:build osHealth && linux

package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/monobilisim/monokit2/lib"
	"github.com/rs/zerolog"
)

type SystemdUnit = lib.SystemdUnit

func CheckSystemInit(logger zerolog.Logger) {
	var moduleName string = "systemd"

	logger.Info().Msg("systemctl command found, checking services...")

	services, err := GetServiceStatus()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get unit statuses")
		return
	}

	for _, service := range services {
		matched := false
		for _, pattern := range lib.OsHealthConfig.ServiceHealthAlarm.Services {
			if match, _ := filepath.Match(pattern, strings.TrimSuffix(service.Name, ".service")); match {
				matched = true
				break
			}
		}

		if !matched {
			continue
		}

		var existingService SystemdUnit
		// using Take, First or Scan logs out "no record found"
		tx := lib.DB.Model(&SystemdUnit{}).Where("name = ? AND project_identifier = ?", service.Name, lib.GlobalConfig.ProjectIdentifier).Find(&existingService)
		if tx.Error != nil {
			logger.Error().Err(err).Msg("Failed to check if service exists in database")
		}

		if tx.RowsAffected == 0 {
			err := lib.DB.Create(&SystemdUnit{
				ProjectIdentifier: lib.GlobalConfig.ProjectIdentifier,
				Hostname:          lib.GlobalConfig.Hostname,
				Name:              service.Name,
				LoadState:         service.LoadState,
				ActiveState:       service.ActiveState,
				SubState:          service.SubState,
				Description:       service.Description,
				Uptime:            service.Uptime,
			}).Error

			if err != nil {
				logger.Error().Err(err).Msgf("Failed to insert %s into database", service.Name)
			}
			continue
		}

		var savedService SystemdUnit
		err = lib.DB.Model(&SystemdUnit{}).Where("name = ? AND project_identifier = ?", service.Name, lib.GlobalConfig.ProjectIdentifier).First(&savedService).Error
		if err != nil {
			logger.Error().Err(err).Msgf("Failed to get current service %s from database", service.Name)
			continue
		}

		// more than 1 service can be down at same time. while Zulip webhooks working like fire and forget, Redmine tickets does not.
		serviceModule := fmt.Sprintf("%s-unit", strings.TrimSuffix(service.Name, ".service"))

		// if service started in last 60 seconds that means it has restarted
		if savedService.Uptime > service.Uptime && service.Uptime > 0 && savedService.Uptime-service.Uptime < 60 {
			logger.Debug().Msgf("Service %s has restarted. Previous uptime: %d seconds, Current uptime: %d seconds", service.Name, savedService.Uptime, service.Uptime)
			alarmMessage := fmt.Sprintf("Service %s has restarted. Previous uptime: %d seconds, Current uptime: %d seconds", service.Name, savedService.Uptime, service.Uptime)

			lib.SendZulipAlarm(alarmMessage, pluginName, moduleName, down)

			if err == nil {
				err = lib.DB.Model(&lib.SystemdUnit{}).Where("name = ? AND project_identifier = ?", service.Name, lib.GlobalConfig.ProjectIdentifier).Updates(service).Error

				if err != nil {
					logger.Error().Err(err).Msgf("Failed to update service %s in database", service.Name)
				}
			}
		}

		issueSubject := fmt.Sprintf("%s için %s servisi çalışmıyor", lib.GlobalConfig.Hostname, service.Name)

		if service.ActiveState != "active" && savedService.ActiveState == "active" {
			logger.Debug().Msgf("Service %s is down. Current state: %s", service.Name, service.ActiveState)
			alarmMessage := fmt.Sprintf("[osHealth] - %s - Service %s is down. Current state: %s", lib.GlobalConfig.Hostname, service.Name, service.ActiveState)

			err := lib.SendZulipAlarm(alarmMessage, pluginName, moduleName, down)
			if err == nil {
				err = lib.DB.Model(&lib.SystemdUnit{}).Where("name = ? AND project_identifier = ?", service.Name, lib.GlobalConfig.ProjectIdentifier).Updates(service).Error

				if err != nil {
					logger.Error().Err(err).Msgf("Failed to update service %s in database", service.Name)
				}
			}

			serviceLogs, err := GetServiceLogs(service.Name, 200)
			if err != nil {
				serviceLogs = fmt.Sprintf("Could not get logs for the service %s from systemd.", service.Name)
			}

			lastIssue, err := lib.GetLastRedmineIssue(pluginName, serviceModule)

			var issue lib.Issue

			if lastIssue.Status == up {
				issue = lib.Issue{
					ProjectIdentifier: lib.GlobalConfig.ProjectIdentifier,
					Hostname:          lib.GlobalConfig.Hostname,
					Subject:           issueSubject,
					Notes:             fmt.Sprintf("Sorun devam ediyor"),
					StatusId:          lib.IssueStatus.Feedback,
					PriorityId:        lib.IssuePriority.Urgent,
					Service:           pluginName,
					Module:            serviceModule,
					Status:            down,
				}
			} else {
				issue = lib.Issue{
					ProjectIdentifier: lib.GlobalConfig.ProjectIdentifier,
					Hostname:          lib.GlobalConfig.Hostname,
					Subject:           issueSubject,
					Description:       serviceLogs,
					StatusId:          lib.IssueStatus.Feedback,
					PriorityId:        lib.IssuePriority.Urgent,
					Service:           pluginName,
					Module:            serviceModule,
					Status:            down,
				}
			}

			lib.CreateRedmineIssue(issue)
		}

		if service.ActiveState == "active" && savedService.ActiveState != "active" {

			logger.Debug().Msgf("Service %s is active again. Current state: %s", service.Name, service.ActiveState)
			lastAlarm, err := lib.GetLastZulipAlarm(pluginName, moduleName)

			if err != nil {
				logger.Error().Err(err).Msg("Failed to get last alarm from database")
				return
			}

			if lastAlarm.Status == down {
				alarmMessage := fmt.Sprintf("[osHealth] - %s - Service %s is now active", lib.GlobalConfig.Hostname, service.Name)

				err := lib.SendZulipAlarm(alarmMessage, pluginName, moduleName, up)
				if err == nil {
					err = lib.DB.Model(&lib.SystemdUnit{}).Where("name = ? AND project_identifier = ?", service.Name, lib.GlobalConfig.ProjectIdentifier).Updates(service).Error

					if err != nil {
						logger.Error().Err(err).Msgf("Failed to update service %s in database", service.Name)
					}
				}
			}

			lastIssue, err := lib.GetLastRedmineIssue(pluginName, serviceModule)

			if lastIssue.Status == down {
				issue := lib.Issue{
					ProjectIdentifier: lib.GlobalConfig.ProjectIdentifier,
					Hostname:          lib.GlobalConfig.Hostname,
					Subject:           issueSubject,
					Notes:             fmt.Sprintf("Sorun çözüldü servis durumu %s", service.ActiveState),
					StatusId:          lib.IssueStatus.Closed,
					PriorityId:        lib.IssuePriority.Urgent,
					Service:           pluginName,
					Module:            serviceModule,
					Status:            up,
				}

				lib.CreateRedmineIssue(issue)
			}
		}
	}
}

func GetServiceStatus() ([]SystemdUnit, error) {
	conn, err := dbus.New()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to dbus: %v", err)
	}
	defer conn.Close()

	units, err := conn.ListUnits()
	if err != nil {
		return nil, fmt.Errorf("failed to list units: %v", err)
	}

	var statuses []SystemdUnit

	for _, unit := range units {
		if !strings.HasSuffix(unit.Name, ".service") {
			continue
		}

		props, err := conn.GetUnitProperties(unit.Name)
		if err != nil {
			continue
		}

		status := SystemdUnit{
			Name:        unit.Name,
			LoadState:   unit.LoadState,
			ActiveState: unit.ActiveState,
			SubState:    unit.SubState,
			Description: unit.Description,
		}

		// Only calculate uptime if the service is active
		if ts, ok := props["ActiveEnterTimestamp"].(uint64); ok && unit.ActiveState == "active" && ts > 0 {
			startTime := time.Unix(0, int64(ts)*1000)
			status.Uptime = int64(time.Since(startTime).Seconds())
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

func GetServiceLogs(service string, lines int) (string, error) {
	if service == "" {
		return "", fmt.Errorf("service name is empty")
	}
	if lines <= 0 {
		return "", fmt.Errorf("lines must be > 0")
	}

	cmd := exec.Command(
		"journalctl",
		"-u", service,
		"-n", fmt.Sprintf("%d", lines),
		"--no-pager",
	)

	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf(
			"journalctl failed: %w: %s",
			err,
			errOut.String(),
		)
	}

	return out.String(), nil
}
