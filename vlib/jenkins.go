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

func JenkinsCheck(logger zerolog.Logger) {
	var jenkinsVersion JenkinsVersion
	var oldJenkinsVersion lib.Version

	_, err := exec.LookPath("jenkins")
	if err != nil {
		logger.Debug().Msg("Jenkins CLI not found, skipping version check")
		return
	}

	out, err := exec.Command("jenkins", "--version").Output()
	if err != nil {
		logger.Error().Err(err).Msg(fmt.Sprintf("Error getting Jenkins version: %s"))
		return
	}

	jenkinsVersion.VersionFull = strings.TrimSpace(string(out))
	jenkinsVersion.Version = jenkinsVersion.VersionFull

	if jenkinsVersion.Version == "" {
		logger.Error().Str("output", jenkinsVersion.Version).Msg(fmt.Sprintf("jenkins --version returns empty: %s", jenkinsVersion.Version))
		return
	}

	err = lib.DB.Model(&lib.Version{}).Where("name = ?", "Jenkins").First(&oldJenkinsVersion).Error
	if err != nil {
		logger.Error().Err(err).Str("component", "osHealth").Str("application", "Jenkins").Str("operation", "query_version").Msg("Error querying Jenkins version from database")
		return
	}

	jenkinsBody, _ := json.Marshal(jenkinsVersion)

	if oldJenkinsVersion.Version == "" && jenkinsVersion.Version != "" {
		logger.Info().Str("application", "Jenkins").Str("version", jenkinsVersion.Version).Msg("Jenkins version detected and recorded")
		lib.DB.Model(&lib.Version{}).Where("name = ?", "Jenkins").Updates(
			lib.Version{
				Version:      jenkinsVersion.Version,
				VersionMulti: string(jenkinsBody),
				Status:       "installed"},
		)
		return
	}

	if oldJenkinsVersion.Version != "" && jenkinsVersion.Version != oldJenkinsVersion.Version {
		logger.Info().Str("application", "Jenkins").Str("old_version", oldJenkinsVersion.Version).Str("new_version", jenkinsVersion.Version).Msg("Jenkins version changed")

		news := lib.News{
			Title:       fmt.Sprintf("%s sunucusunun Jenkins sürümü güncellendi.", lib.GlobalConfig.Hostname),
			Description: fmt.Sprintf("%s sunucusunda Jenkins, %s sürümünden %s sürümüne yükseltildi.", lib.GlobalConfig.Hostname, oldJenkinsVersion.Version, jenkinsVersion.Version),
		}

		lib.CreateRedmineNews(news)

		lib.DB.Model(&lib.Version{}).Where("name = ?", "Jenkins").Updates(
			lib.Version{
				Version:      jenkinsVersion.Version,
				VersionMulti: string(jenkinsBody),
				Status:       "installed"},
		)
		return
	}

}
