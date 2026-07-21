//go:build osHealth

package vlib

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/monobilisim/monokit2/lib"
	"github.com/rs/zerolog"
)

func ZimbraCheck(logger zerolog.Logger) {
	var zimbraVersion ZimbraVersion
	var oldZimbraVersion lib.Version
	var zimbraPath string
	var zimbraUser string

	if _, err := os.Stat("/opt/zimbra"); !os.IsNotExist(err) {
		zimbraPath = "/opt/zimbra"
		zimbraUser = "zimbra"
	}

	// Check if zimbraPath is empty
	if zimbraPath == "" {
		logger.Debug().Msg("Zimbra installation not found, skipping version check")
		return
	}

	// Get the version of Zimbra
	cmd := exec.Command("/bin/su", zimbraUser, "-c", zimbraPath+"/bin/zmcontrol -v")
	out, err := cmd.Output()
	if err != nil {
		logger.Error().Err(err).Msg("Error getting Zimbra version output")
		return
	}

	zimbraVersion.VersionFull = strings.TrimSpace(string(out))

	// Release 8.8.15_GA_3869.UBUNTU18.64 UBUNTU18_64 FOSS edition.
	// Release 10.0.7_GA_0005.RHEL8_64 RHEL8_64 NETWORK edition.
	// Release 8.8.15_GA_3869.UBUNTU18.64 UBUNTU18_64 FOSS edition.
	versionParts := strings.Fields(string(zimbraVersion.VersionFull))
	if len(versionParts) < 2 {
		logger.Error().Msg(fmt.Sprintf("Unexpected output format from zmcontrol -v: %s", zimbraVersion.VersionFull))
		return
	}
	// Extract version like "8.8.15" or "10.0.7"
	zimbraVersion.Version = strings.Split(versionParts[1], "_GA_")[0]

	err = lib.DB.Model(&lib.Version{}).Where("name = ?", "Zimbra").First(&oldZimbraVersion).Error
	if err != nil {
		logger.Error().Err(err).Str("component", "osHealth").Str("application", "Zimbra").Str("operation", "query_version").Msg("Error querying Zimbra version from database")
		return
	}

	zimbraJson, _ := json.Marshal(zimbraVersion)

	if oldZimbraVersion.Version == "" && zimbraVersion.Version != "" {
		logger.Info().Str("application", "Zimbra").Str("version", zimbraVersion.Version).Msg("Zimbra version detected and recorded")
		lib.DB.Model(&lib.Version{}).Where("name = ?", "Zimbra").Updates(
			lib.Version{
				Version:      zimbraVersion.Version,
				VersionMulti: string(zimbraJson),
				Status:       "installed",
			},
		)
		return
	}

	if oldZimbraVersion.Version != "" && zimbraVersion.Version != oldZimbraVersion.Version {
		logger.Info().Str("application", "Zimbra").Str("old_version", oldZimbraVersion.Version).Str("new_version", zimbraVersion.Version).Msg("Zimbra version changed")

		news := lib.News{
			Title:       fmt.Sprintf("%s sunucusunun Zimbra sürümü güncellendi.", lib.GlobalConfig.Hostname),
			Description: fmt.Sprintf("%s sunucusunda Zimbra, %s sürümünden %s sürümüne yükseltildi.", lib.GlobalConfig.Hostname, oldZimbraVersion.Version, zimbraVersion.Version),
		}

		lib.CreateRedmineNews(news)

		lib.DB.Model(&lib.Version{}).Where("name = ?", "Zimbra").Updates(
			lib.Version{
				Version:      zimbraVersion.Version,
				VersionMulti: string(zimbraJson),
				Status:       "installed"},
		)
	}

}
