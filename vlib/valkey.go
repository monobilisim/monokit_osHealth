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

func ValkeyCheck(logger zerolog.Logger) {
	var valkeyVersion ValkeyVersion
	var oldValkeyVersion lib.Version

	_, err := exec.LookPath("valkey-server")
	if err != nil {
		logger.Debug().Msg("Valkey server binary not found, skipping version check")
		return
	}

	// Output example of redis-server --version:
	// Valkey server v=8.0.6 sha=00000000:0 malloc=jemalloc-5.3.0 bits=64 build=cc4ea19b99ae73a7
	// Redis server v=7.0.15 sha=00000000:0 malloc=jemalloc-5.3.0 bits=64 build=5281cccdf7ef82d6
	out, err := exec.Command("valkey-server", "--version").Output()
	if err != nil {
		logger.Error().Err(err).Msg("Error getting Valkey version")
		return
	}

	valkeyVersion.VersionFull = strings.TrimSpace(string(out))

	if valkeyVersion.VersionFull == "" {
		logger.Error().Str("output", valkeyVersion.VersionFull).Msg("valkey-server --version returns empty")
		return
	}

	if strings.Contains(strings.ToLower(valkeyVersion.VersionFull), "redis") {
		logger.Debug().Msg("Detected Redis installation, skipping the version check")
		return
	}

	fields := strings.Fields(valkeyVersion.VersionFull)
	for _, field := range fields {
		if strings.HasPrefix(field, "v=") {
			valkeyVersion.Version = strings.TrimSpace(strings.TrimPrefix(field, "v="))
			break
		}
	}

	if valkeyVersion.Version == "" {
		logger.Error().Str("output", valkeyVersion.VersionFull).Msg("Could not parse Valkey version")
		return
	}

	err = lib.DB.Model(&lib.Version{}).Where("name = ?", "Valkey").First(&oldValkeyVersion).Error
	if err != nil {
		logger.Error().Err(err).Str("component", "osHealth").Str("application", "Valkey").Str("operation", "query_version").Msg("Error querying Valkey version from database")
		return
	}

	valkeyBody, _ := json.Marshal(valkeyVersion)

	if oldValkeyVersion.Version == "" && valkeyVersion.Version != "" {
		logger.Info().Str("application", "Valkey").Str("version", valkeyVersion.Version).Msg("Valkey version detected and recorded")
		lib.DB.Model(&lib.Version{}).Where("name = ?", "Valkey").Updates(
			lib.Version{
				Version:      valkeyVersion.Version,
				VersionMulti: string(valkeyBody),
				Status:       "installed"},
		)
		return
	}

	if oldValkeyVersion.Version != "" && valkeyVersion.Version != oldValkeyVersion.Version {
		logger.Info().Str("application", "Valkey").Str("old_version", oldValkeyVersion.Version).Str("new_version", valkeyVersion.Version).Msg("Valkey version changed")

		news := lib.News{
			Title:       fmt.Sprintf("%s sunucusunun Valkey sürümü güncellendi.", lib.GlobalConfig.Hostname),
			Description: fmt.Sprintf("%s sunucusunda Valkey, %s sürümünden %s sürümüne yükseltildi.", lib.GlobalConfig.Hostname, oldValkeyVersion.Version, valkeyVersion.Version),
		}

		lib.CreateRedmineNews(news)

		lib.DB.Model(&lib.Version{}).Where("name = ?", "Valkey").Updates(
			lib.Version{
				Version:      valkeyVersion.Version,
				VersionMulti: string(valkeyBody),
				Status:       "installed"},
		)
	}
}
