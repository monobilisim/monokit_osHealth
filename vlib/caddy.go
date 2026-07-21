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

func CaddyCheck(logger zerolog.Logger) {
	var caddyVersion CaddyVersion
	var oldCaddyVersion lib.Version

	// Check if Caddy binary is installed
	_, err := exec.LookPath("caddy")
	if err != nil {
		logger.Debug().Msg("Cocker CLI not found, skipping version check")
		return
	}

	/* Example output of `caddy version`:
	 * v2.9.1 h1:OEYiZ7DbCzAWVb6TNEkjRcSCRGHVoZsJinoDR/n9oaY=
	 * OR
	 * Caddy v2.9.1 h1:OEYiZ7DbCzAWVb6TNEkjRcSCRGHVoZsJinoDR/n9oaY=
	 */
	out, err := exec.Command("caddy", "version").Output()
	if err != nil {
		logger.Error().Err(err).Msg("Error executing caddy version command")
		return
	}

	versionOutput := strings.TrimSpace(string(out))
	caddyVersion.VersionFull = versionOutput
	fields := strings.Fields(versionOutput)
	if len(fields) == 0 {
		logger.Error().Err(fmt.Errorf("%s: %s", "Version string is empty", versionOutput)).Msg("Caddy version parsing error")
		return
	}

	first := fields[0]
	if strings.EqualFold(first, "caddy") {
		if len(fields) < 2 {
			logger.Error().Err(fmt.Errorf("%s: %s", "Unexpected output format form caddy", versionOutput)).Msg("Caddy version parsing error")
			return
		}
		first = fields[1]
	}
	caddyVersion.Version = first

	logger.Debug().Str("version", caddyVersion.Version).Msg("Detected Caddy version")

	err = lib.DB.Model(&lib.Version{}).Where("name = ?", "Caddy").First(&oldCaddyVersion).Error
	if err != nil {
		logger.Error().Err(err).Str("component", "osHealth").Str("application", "Caddy").Str("operation", "query_version").Msg("Error querying Caddy version from database")
		return
	}

	caddyJson, _ := json.Marshal(caddyVersion)

	if oldCaddyVersion.Version == "" && caddyVersion.Version != "" {
		logger.Debug().Msg(fmt.Sprintf("Storing initial Caddy version: %s", caddyVersion.Version))
		lib.DB.Model(&lib.Version{}).Where("name = ?", "Caddy").Updates(lib.Version{
			Version:      caddyVersion.Version,
			VersionMulti: string(caddyJson),
			Status:       "installed",
		})
		return
	}

	if oldCaddyVersion.Version != "" && oldCaddyVersion.Version != caddyVersion.Version {
		logger.Debug().Msg("Caddy has been updated.")
		logger.Debug().Str("old_version", oldCaddyVersion.Version).Str("new_version", caddyVersion.Version).Msg("Caddy has been updated")

		news := lib.News{
			Title:       fmt.Sprintf("%s sunucusunun Caddy sürümü güncellendi.", lib.GlobalConfig.Hostname),
			Description: fmt.Sprintf("%s sunucusunda Caddy, %s sürümünden %s sürümüne yükseltildi.", lib.GlobalConfig.Hostname, oldCaddyVersion.Version, caddyVersion.Version),
		}

		lib.CreateRedmineNews(news)

		lib.DB.Model(&lib.Version{}).Where("name = ?", "Caddy").Updates(lib.Version{
			Version:      caddyVersion.Version,
			VersionMulti: string(caddyJson),
			Status:       "installed",
		})
	}

	// if oldVersion != "" && oldVersion == version {
	// 	log.Debug().Msg("Caddy version unchanged.")
	// 	addToNotUpdated(AppVersion{Name: "Caddy", OldVersion: oldVersion})
	// } else if oldVersion != "" && oldVersion != version {
	// 	log.Debug().Msg("Caddy has been updated.")
	// 	log.Debug().Str("old_version", oldVersion).Str("new_version", version).Msg("Caddy has been updated")
	// 	addToUpdated(AppVersion{Name: "Caddy", OldVersion: oldVersion, NewVersion: version})
	// 	CreateNews("Caddy", oldVersion, version, false)
	// } else {
	// 	log.Debug().Msg("Storing initial Caddy version: " + version)
	// 	addToNotUpdated(AppVersion{Name: "Caddy", OldVersion: version})
	// }

	// StoreVersion("caddy", version)
	// return version, nil
}
