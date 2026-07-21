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

func FrankenPHPCheck(logger zerolog.Logger) {
	var frankenphpVersion FrankenPHPVersion
	var oldFrankenPHPVersion lib.Version

	_, err := exec.LookPath("frankenphp")
	if err != nil {
		logger.Debug().Msg("FrankenPHP binary not found, skipping version check")
		return
	}

	out, err := exec.Command("frankenphp", "-v").Output()
	if err != nil {
		logger.Error().Err(err).Msg("Error getting FrankenPHP version")
		return
	}

	/* Example output of frankenphp version:
	 *  FrankenPHP v1.11.0 PHP 8.4.16 Caddy v2.10.2 h1:g/gTYjGMD0dec+UgMw8SnfmJ3I9+M2TdvoRL/Ovu6U8=
	 */
	frankenphpVersion.VersionFull = strings.TrimSpace(string(out))

	fields := strings.Fields(frankenphpVersion.VersionFull)

	if len(fields) < 2 {
		logger.Error().Err(fmt.Errorf("%s: %s", "Unexpected output format from frankenphp", frankenphpVersion.VersionFull)).Msg("FrankenPHP version parsing error")
		return
	}

	frankenphpVersion.FrankenPHP.Version = strings.TrimSpace(fields[1])
	frankenphpVersion.FrankenPHP.VersionFull = fmt.Sprintf("%s %s", strings.TrimSpace(fields[0]), strings.TrimSpace(fields[1]))
	frankenphpVersion.PHP.Version = strings.TrimSpace(fields[3])
	frankenphpVersion.PHP.VersionFull = fmt.Sprintf("%s %s", strings.TrimSpace(fields[2]), strings.TrimSpace(fields[3]))
	frankenphpVersion.Caddy.Version = strings.TrimSpace(fields[5])
	frankenphpVersion.Caddy.VersionFull = fmt.Sprintf("%s %s %s", strings.TrimSpace(fields[4]), strings.TrimSpace(fields[5]), strings.TrimSpace(fields[6]))

	err = lib.DB.Model(&lib.Version{}).Where("name = ?", "FrankenPHP").First(&oldFrankenPHPVersion).Error
	if err != nil {
		logger.Error().Err(err).Str("component", "osHealth").Str("application", "FrankenPHP").Str("operation", "query_version").Msg("Error querying FrankenPHP version from database")
		return
	}

	frankenphpBody, _ := json.Marshal(frankenphpVersion)

	if oldFrankenPHPVersion.Version == "" && frankenphpVersion.FrankenPHP.Version != "" {
		logger.Info().Str("application", "FrankenPHP").Str("version", frankenphpVersion.FrankenPHP.Version).Msg("FrankenPHP version detected and recorded")
		lib.DB.Model(&lib.Version{}).Where("name = ?", "FrankenPHP").Updates(
			lib.Version{
				Version:      frankenphpVersion.FrankenPHP.Version,
				VersionMulti: string(frankenphpBody),
				Status:       "installed",
			})
		return
	}

	if oldFrankenPHPVersion.Version != "" && oldFrankenPHPVersion.Version != frankenphpVersion.FrankenPHP.Version {
		logger.Info().Str("application", "FrankenPHP").Str("old_version", oldFrankenPHPVersion.Version).Str("new_version", frankenphpVersion.FrankenPHP.Version).Msg("FrankenPHP version changed")

		news := lib.News{
			Title:       fmt.Sprintf("%s sunucusunun FrankenPHP sürümü güncellendi.", lib.GlobalConfig.Hostname),
			Description: fmt.Sprintf("%s sunucusunda FrankenPHP, %s sürümünden %s sürümüne yükseltildi.", lib.GlobalConfig.Hostname, oldFrankenPHPVersion.Version, frankenphpVersion.FrankenPHP.Version),
		}

		lib.CreateRedmineNews(news)

		lib.DB.Model(&lib.Version{}).Where("name = ?", "FrankenPHP").Updates(
			lib.Version{
				Version:      frankenphpVersion.FrankenPHP.Version,
				VersionMulti: string(frankenphpBody),
				Status:       "installed",
			})
		return
	}

}
