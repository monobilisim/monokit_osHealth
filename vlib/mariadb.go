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

func MariaDBCheck(logger zerolog.Logger) {
	var mariadbVersion MariaDBVersion
	var oldMariaDBVersion lib.Version

	// Ensure /usr/sbin is in PATH to locate mariadbd or mariadbd for backward compatibility
	currentPath := os.Getenv("PATH")
	newPath := fmt.Sprintf("/usr/sbin:%s", currentPath)
	os.Setenv("PATH", newPath)

	_, err := exec.LookPath("mariadbd")
	if err != nil {
		logger.Debug().Msg("mariadbd binary not found; MariaDB may not be installed.")
		return
	}

	out, err := exec.Command("mariadbd", "--version").Output()
	if err != nil {
		logger.Error().Err(err).Msg("Error getting MariaDB version")
		return
	}

	// mariadbd  Ver 10.11.11-MariaDB for Linux on x86_64 (MariaDB Server)
	// mysqld  Ver 5.7.24 for Linux on x86_64 (MySQL Community Server (GPL))
	mariadbVersion.VersionFull = strings.TrimSpace(string(out))

	if mariadbVersion.VersionFull == "" {
		logger.Error().Str("output", mariadbVersion.VersionFull).Msg("mariadbd --version returns empty")
		return
	}

	if strings.Contains(strings.ToLower(mariadbVersion.VersionFull), "mysql") {
		logger.Debug().Msg("Detected MySQL installation, skipping MariaDB version check")
		return
	}

	parts := strings.Fields(mariadbVersion.VersionFull)
	// get the 10.11.11 part from 10.11.11-MariaDB
	mariadbVersion.Version = strings.TrimSuffix(strings.ToLower(parts[2]), "-mariadb")

	err = lib.DB.Model(&lib.Version{}).Where("name = ?", "MariaDB").First(&oldMariaDBVersion).Error
	if err != nil {
		logger.Error().Err(err).Str("component", "osHealth").Str("application", "MariaDB").Str("operation", "query_version").Msg("Error querying MariaDB version from database")
		return
	}

	mariadbBody, _ := json.Marshal(mariadbVersion)

	if oldMariaDBVersion.Version == "" && mariadbVersion.Version != "" {
		logger.Info().Str("application", "MariaDB").Str("version", mariadbVersion.Version).Msg("MariaDB version detected and recorded")
		lib.DB.Model(&lib.Version{}).Where("name = ?", "MariaDB").Updates(
			lib.Version{
				Version:      mariadbVersion.Version,
				VersionMulti: string(mariadbBody),
				Status:       "installed"},
		)
		return
	}

	if oldMariaDBVersion.Version != "" && mariadbVersion.Version != oldMariaDBVersion.Version {
		logger.Info().Str("application", "MariaDB").Str("old_version", oldMariaDBVersion.Version).Str("new_version", mariadbVersion.Version).Msg("MariaDB version changed")

		news := lib.News{
			Title:       fmt.Sprintf("%s sunucusunun MariaDB sürümü güncellendi.", lib.GlobalConfig.Hostname),
			Description: fmt.Sprintf("%s sunucusunda MariaDB, %s sürümünden %s sürümüne yükseltildi.", lib.GlobalConfig.Hostname, oldMariaDBVersion.Version, mariadbVersion.Version),
		}

		lib.CreateRedmineNews(news)

		lib.DB.Model(&lib.Version{}).Where("name = ?", "MariaDB").Updates(
			lib.Version{
				Version:      mariadbVersion.Version,
				VersionMulti: string(mariadbBody),
				Status:       "installed"},
		)
		return
	}

}
