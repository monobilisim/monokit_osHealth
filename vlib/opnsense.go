//go:build osHealth

package vlib

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	lib "github.com/monobilisim/monokit2/lib"
	"github.com/rs/zerolog"
)

func OPNsenseCheck(logger zerolog.Logger) {
	var opnsenseVersion OPNsenseVersion
	var oldOPNsenseVersion lib.Version

	_, err := exec.LookPath("opnsense-version")
	if err != nil {
		logger.Debug().Msg("opnsense-version binary not found, skipping version check")
		return
	}

	out, err := exec.Command("opnsense-version").Output()
	if err != nil {
		logger.Error().Err(err).Msg("Error getting OPNsense version")
		return
	}

	// OPNsense 19.1.b_264 (amd64/LibreSSL)
	// OPNsense 21.1.8_1 (amd64)
	opnsenseVersion.VersionFull = strings.TrimSpace(string(out))

	if opnsenseVersion.VersionFull == "" {
		logger.Error().Str("output", opnsenseVersion.VersionFull).Msg("opnsense-version returns empty")
		return
	}

	opnsenseVersion.Version = strings.Split(opnsenseVersion.VersionFull, " ")[1]

	err = lib.DB.Model(&lib.Version{}).Where("name = ?", "OPNsense").First(&oldOPNsenseVersion).Error
	if err != nil {
		logger.Error().Err(err).Str("component", "osHealth").Str("application", "OPNsense").Str("operation", "query_version").Msg("Error querying OPNsense version from database")
		return
	}

	opnsenseBody, _ := json.Marshal(opnsenseVersion)

	if oldOPNsenseVersion.Version == "" && opnsenseVersion.Version != "" {
		logger.Info().Str("application", "OPNsense").Str("version", opnsenseVersion.Version).Msg("OPNsense version detected and recorded")
		lib.DB.Model(&lib.Version{}).Where("name = ?", "OPNsense").Updates(
			lib.Version{
				Version:      opnsenseVersion.Version,
				VersionMulti: string(opnsenseBody),
				Status:       "installed"},
		)
		return
	}

	if oldOPNsenseVersion.Version != "" && opnsenseVersion.Version != oldOPNsenseVersion.Version {
		logger.Info().Str("application", "OPNsense").Str("old_version", oldOPNsenseVersion.Version).Str("new_version", opnsenseVersion.Version).Msg("OPNsense version changed")

		news := lib.News{
			Title:       fmt.Sprintf("%s sunucusunun OPNsense sürümü güncellendi.", lib.GlobalConfig.Hostname),
			Description: fmt.Sprintf("%s sunucusunda OPNsense, %s sürümünden %s sürümüne yükseltildi.", lib.GlobalConfig.Hostname, oldOPNsenseVersion.Version, opnsenseVersion.Version),
		}

		lib.CreateRedmineNews(news)

		lib.DB.Model(&lib.Version{}).Where("name = ?", "OPNsense").Updates(
			lib.Version{
				Version:      opnsenseVersion.Version,
				VersionMulti: string(opnsenseBody),
				Status:       "installed"},
		)
		return
	}
}
