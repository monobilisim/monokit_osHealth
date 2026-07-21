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

func AsteriskCheck(logger zerolog.Logger) {
	var asteriskVersion AsteriskVersion
	var oldAsteriskVersion lib.Version

	_, err := exec.LookPath("asterisk")
	if err != nil {
		logger.Debug().Msg("Asterisk CLI not found, skipping version check")
		return
	}

	/*
	* Example output of asterisk -V:
	* Asterisk 22.3.0
	*
	 */
	out, err := exec.Command("asterisk", "-V").Output()
	if err != nil {
		logger.Error().Err(err).Msg("Error getting Asterisk version")
		return
	}

	version := string(out)

	if version == "" {
		logger.Error().Msg("asterisk -V returns empty")
		return
	}

	asteriskVersion.VersionFull = strings.TrimSpace(version)
	parts := strings.Fields(asteriskVersion.VersionFull)
	if len(parts) >= 2 {
		asteriskVersion.Version = parts[1]
	} else {
		logger.Error().Err(fmt.Errorf("%s: %s", "Unexpected output format from asterisk", version)).Msg("Asterisk version parsing error")
		return
	}

	logger.Debug().Str("version", version).Msg("Detected Asterisk version")

	err = lib.DB.Model(&lib.Version{}).Where("name = ?", "Asterisk").First(&oldAsteriskVersion).Error
	if err != nil {
		logger.Error().Err(err).Str("component", "osHealth").Str("application", "Asterisk").Str("operation", "query_version").Msg("Error querying Asterisk version from database")
		return
	}

	asteriskBody, _ := json.Marshal(asteriskVersion)

	if oldAsteriskVersion.Version == "" && asteriskVersion.Version != "" {
		logger.Debug().Msg(fmt.Sprintf("Storing initial Asterisk version: %s", asteriskVersion.Version))

		lib.DB.Model(&lib.Version{}).Where("name = ?", "Asterisk").Updates(lib.Version{
			Version:      asteriskVersion.Version,
			VersionMulti: string(asteriskBody),
			Status:       "installed",
		})
		return
	}

	if oldAsteriskVersion.Version != "" && oldAsteriskVersion.Version != asteriskVersion.Version {
		logger.Info().Str("old_version", oldAsteriskVersion.Version).
			Str("new_version", asteriskVersion.Version).
			Msg("Docker Engine version has been updated")

		news := lib.News{
			Title:       fmt.Sprintf("%s sunucusunun Asterisk sürümü güncellendi.", lib.GlobalConfig.Hostname),
			Description: fmt.Sprintf("%s sunucusunda Asterisk, %s sürümünden %s sürümüne yükseltildi.", lib.GlobalConfig.Hostname, oldAsteriskVersion.Version, asteriskVersion.Version),
		}

		lib.CreateRedmineNews(news)

		lib.DB.Model(&lib.Version{}).Where("name = ?", "Asterisk").Updates(lib.Version{
			Version:      asteriskVersion.Version,
			VersionMulti: string(asteriskBody),
			Status:       "installed",
		})
	}
}
