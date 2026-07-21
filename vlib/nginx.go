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

func NginxCheck(logger zerolog.Logger) {
	var nginxVersion NginxVersion
	var oldNginxVersion lib.Version

	if _, err := exec.LookPath("nginx"); err != nil {
		logger.Debug().Msg("Nginx binary not found, skipping version check")
		return
	}

	// nginx version: nginx/1.28.0
	// nginx version: openresty/1.21.4.1
	out, err := exec.Command("nginx", "-v").CombinedOutput()
	if err != nil {
		logger.Debug().Err(err).Msg("Nginx -v returned non-zero; attempting to parse output")
		return
	}

	nginxVersion.VersionFull = strings.TrimSpace(string(out))

	if nginxVersion.VersionFull == "" {
		logger.Error().Str("output", nginxVersion.VersionFull).Msg("nginx -v returns empty")
		return
	}

	parts := strings.Split(nginxVersion.VersionFull, "version:")
	if len(parts) < 2 {
		logger.Error().Str("output", nginxVersion.VersionFull).Msg("Unexpected nginx -v output format")
		return
	}

	// extracts 1.28.0 from nginx/1.28.0
	nginxVersion.Version = strings.TrimSpace(strings.Split(parts[1], "/")[1])

	err = lib.DB.Model(&lib.Version{}).Where("name = ?", "Nginx").First(&oldNginxVersion).Error
	if err != nil {
		logger.Error().Err(err).Str("component", "osHealth").Str("application", "Nginx").Str("operation", "query_version").Msg("Error querying Nginx version from database")
		return
	}

	nginxBody, _ := json.Marshal(nginxVersion)

	if oldNginxVersion.Version == "" && nginxVersion.Version != "" {
		logger.Info().Str("application", "Nginx").Str("version", nginxVersion.Version).Msg("Nginx version detected and recorded")
		lib.DB.Model(&lib.Version{}).Where("name = ?", "Nginx").Updates(
			lib.Version{
				Version:      nginxVersion.Version,
				VersionMulti: string(nginxBody),
				Status:       "installed"},
		)
		return
	}

	if oldNginxVersion.Version != "" && nginxVersion.Version != oldNginxVersion.Version {
		logger.Info().Str("application", "Nginx").Str("old_version", oldNginxVersion.Version).Str("new_version", nginxVersion.Version).Msg("Nginx version changed")

		news := lib.News{
			Title:       fmt.Sprintf("%s sunucusunun Nginx sürümü güncellendi.", lib.GlobalConfig.Hostname),
			Description: fmt.Sprintf("%s sunucusunda Nginx, %s sürümünden %s sürümüne yükseltildi.", lib.GlobalConfig.Hostname, oldNginxVersion.Version, nginxVersion.Version),
		}

		lib.CreateRedmineNews(news)

		lib.DB.Model(&lib.Version{}).Where("name = ?", "Nginx").Updates(
			lib.Version{
				Version:      nginxVersion.Version,
				VersionMulti: string(nginxBody),
				Status:       "installed"},
		)
		return
	}
}
