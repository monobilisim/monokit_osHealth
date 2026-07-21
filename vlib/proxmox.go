//go:build osHealth

package vlib

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/monobilisim/monokit2/lib"
	"github.com/rs/zerolog"
)

func ProxmoxVECheck(logger zerolog.Logger) {
	var pveVersion PVEVersion
	var oldPVEVersion lib.Version

	pveVersion.Type = "PVE"

	_, err := exec.LookPath("pveversion")
	if err != nil {
		logger.Debug().Msg("pveversion binaries not found, skipping version check")
		return
	}

	out, err := exec.Command("pveversion").Output()
	if err != nil {
		logger.Error().Msg("Error getting pveversion output")
		return
	}

	// pve-manager/6.4-13/1c2b3f0e (running kernel: 5.4.78-2-pve)
	pveVersion.VersionFull = strings.TrimSpace(string(out))

	parts := strings.Split(pveVersion.VersionFull, "/")

	pveVersion.Version = parts[1]

	err = lib.DB.Model(&lib.Version{}).Where("name = ?", pveVersion.Type).First(&oldPVEVersion).Error
	if err != nil {
		logger.Error().Err(err).Str("component", "osHealth").Str("application", pveVersion.Type).Str("operation", "query_version").Msg("Error querying PVE version from database")
		return
	}

	pveJson, _ := json.Marshal(pveVersion)

	if oldPVEVersion.Version == "" && pveVersion.Version != "" {
		logger.Info().Str("application", pveVersion.Type).Str("version", pveVersion.Version).Msg("PVE version detected and recorded")
		lib.DB.Model(&lib.Version{}).Where("name = ?", pveVersion.Type).Updates(
			lib.Version{
				Version:      pveVersion.Version,
				VersionMulti: string(pveJson),
				Status:       "installed",
			},
		)
	}

	if oldPVEVersion.Version != "" && pveVersion.Version != oldPVEVersion.Version {
		logger.Info().Str("application", pveVersion.Type).Str("old_version", oldPVEVersion.Version).Str("new_version", pveVersion.Version).Msg("PVE version changed")

		news := lib.News{
			Title:       fmt.Sprintf("%s sunucusunun Proxmox Virtual Environment sürümü güncellendi.", lib.GlobalConfig.Hostname),
			Description: fmt.Sprintf("%s sunucusunda Proxmox Virtual Environment, %s sürümünden %s sürümüne yükseltildi.", lib.GlobalConfig.Hostname, oldPVEVersion.Version, pveVersion.Version),
		}

		lib.CreateRedmineNews(news)

		lib.DB.Model(&lib.Version{}).Where("name = ?", pveVersion.Type).Updates(
			lib.Version{
				Version:      pveVersion.Version,
				VersionMulti: string(pveJson),
				Status:       "installed"},
		)
	}
}

func ProxmoxMGCheck(logger zerolog.Logger) {
	var pveVersion PVEVersion
	var oldPVEVersion lib.Version

	pveVersion.Type = "PMG"

	_, err := exec.LookPath("pmgversion")
	if err != nil {
		logger.Debug().Msg("pmgversion binaries not found, skipping version check")
		return
	}

	out, err := exec.Command("pmgversion").Output()
	if err != nil {
		logger.Error().Msg("Error getting pmgversion output")
		return
	}

	// pmg/6.4-13/1c2b3f0e (running kernel: 5.4.78-2-pve)
	pveVersion.VersionFull = strings.TrimSpace(string(out))

	parts := strings.Split(pveVersion.VersionFull, "/")

	pveVersion.Version = parts[1]

	err = lib.DB.Model(&lib.Version{}).Where("name = ?", pveVersion.Type).First(&oldPVEVersion).Error
	if err != nil {
		logger.Error().Err(err).Str("component", "osHealth").Str("application", pveVersion.Type).Str("operation", "query_version").Msg("Error querying PMG version from database")
		return
	}

	pveJson, _ := json.Marshal(pveVersion)

	if oldPVEVersion.Version == "" && pveVersion.Version != "" {
		logger.Info().Str("application", pveVersion.Type).Str("version", pveVersion.Version).Msg("PMG version detected and recorded")
		lib.DB.Model(&lib.Version{}).Where("name = ?", pveVersion.Type).Updates(
			lib.Version{
				Version:      pveVersion.Version,
				VersionMulti: string(pveJson),
				Status:       "installed",
			},
		)
	}

	if oldPVEVersion.Version != "" && pveVersion.Version != oldPVEVersion.Version {
		logger.Info().Str("application", pveVersion.Type).Str("old_version", oldPVEVersion.Version).Str("new_version", pveVersion.Version).Msg("PMG version changed")

		news := lib.News{
			Title:       fmt.Sprintf("%s sunucusunun Proxmox Mail Gateway sürümü güncellendi.", lib.GlobalConfig.Hostname),
			Description: fmt.Sprintf("%s sunucusunda Proxmox Mail Gateway, %s sürümünden %s sürümüne yükseltildi.", lib.GlobalConfig.Hostname, oldPVEVersion.Version, pveVersion.Version),
		}

		lib.CreateRedmineNews(news)

		lib.DB.Model(&lib.Version{}).Where("name = ?", pveVersion.Type).Updates(
			lib.Version{
				Version:      pveVersion.Version,
				VersionMulti: string(pveJson),
				Status:       "installed"},
		)
	}
}

func ProxmoxBSCheck(logger zerolog.Logger) {
	var pveVersion PVEVersion
	var oldPVEVersion lib.Version

	pveVersion.Type = "PBS"

	_, err := exec.LookPath("proxmox-backup-manager")
	if err != nil {
		logger.Debug().Msg("proxmox-backup-manager binaries not found, skipping version check")
		return
	}

	out, err := exec.Command("proxmox-backup-manager", "version").Output()
	if err != nil {
		logger.Error().Msg("Error getting proxmox-backup-manager version output")
		return
	}

	// proxmox-backup-server 3.3.2-1 running version: 3.3.2
	pveVersion.VersionFull = strings.TrimSpace(string(out))

	pveVersion.Version = strings.Fields(pveVersion.VersionFull)[4]

	err = lib.DB.Model(&lib.Version{}).Where("name = ?", pveVersion.Type).First(&oldPVEVersion).Error
	if err != nil {
		logger.Error().Err(err).Str("component", "osHealth").Str("application", pveVersion.Type).Str("operation", "query_version").Msg("Error querying PBS version from database")
		return
	}

	pveJson, _ := json.Marshal(pveVersion)

	if oldPVEVersion.Version == "" && pveVersion.Version != "" {
		logger.Info().Str("application", pveVersion.Type).Str("version", pveVersion.Version).Msg("PBS version detected and recorded")
		lib.DB.Model(&lib.Version{}).Where("name = ?", pveVersion.Type).Updates(
			lib.Version{
				Version:      pveVersion.Version,
				VersionMulti: string(pveJson),
				Status:       "installed",
			},
		)
	}

	if oldPVEVersion.Version != "" && pveVersion.Version != oldPVEVersion.Version {
		logger.Info().Str("application", pveVersion.Type).Str("old_version", oldPVEVersion.Version).Str("new_version", pveVersion.Version).Msg("PBS version changed")

		news := lib.News{
			Title:       fmt.Sprintf("%s sunucusunun Proxmox Backup Server sürümü güncellendi.", lib.GlobalConfig.Hostname),
			Description: fmt.Sprintf("%s sunucusunda Proxmox Backup Server, %s sürümünden %s sürümüne yükseltildi.", lib.GlobalConfig.Hostname, oldPVEVersion.Version, pveVersion.Version),
		}

		lib.CreateRedmineNews(news)

		lib.DB.Model(&lib.Version{}).Where("name = ?", pveVersion.Type).Updates(
			lib.Version{
				Version:      pveVersion.Version,
				VersionMulti: string(pveJson),
				Status:       "installed"},
		)
	}
}
