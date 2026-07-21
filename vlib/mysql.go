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

func MySQLCheck(logger zerolog.Logger) {
	var mysqlVersion MySQLVersion
	var oldMySQLVersion lib.Version

	// Ensure /usr/sbin is in PATH to locate mysqld or mariadbd for backward compatibility
	currentPath := os.Getenv("PATH")
	newPath := fmt.Sprintf("/usr/sbin:%s", currentPath)
	os.Setenv("PATH", newPath)

	_, err := exec.LookPath("mysqld")
	if err != nil {
		logger.Debug().Msg("mysqld binary not found; MySQL may not be installed.")
		return
	}

	out, err := exec.Command("mysqld", "--version").Output()
	if err != nil {
		logger.Error().Err(err).Msg("Error getting MySQL version")
		return
	}

	// mariadbd  Ver 10.11.11-MariaDB for Linux on x86_64 (MariaDB Server)
	// mysqld  Ver 5.7.24 for Linux on x86_64 (MySQL Community Server (GPL))
	mysqlVersion.VersionFull = strings.TrimSpace(string(out))

	if mysqlVersion.VersionFull == "" {
		logger.Error().Str("output", mysqlVersion.VersionFull).Msg("mysqld --version returns empty")
		return
	}

	if strings.Contains(strings.ToLower(mysqlVersion.VersionFull), "maria") {
		logger.Debug().Msg("Detected MariaDB installation, skipping MySQL version check")
		return
	}

	parts := strings.Fields(mysqlVersion.VersionFull)
	mysqlVersion.Version = parts[2]

	err = lib.DB.Model(&lib.Version{}).Where("name = ?", "MySQL").First(&oldMySQLVersion).Error
	if err != nil {
		logger.Error().Err(err).Str("component", "osHealth").Str("application", "MySQL").Str("operation", "query_version").Msg("Error querying MySQL version from database")
		return
	}

	mysqlBody, _ := json.Marshal(mysqlVersion)

	if oldMySQLVersion.Version == "" && mysqlVersion.Version != "" {
		logger.Info().Str("application", "MySQL").Str("version", mysqlVersion.Version).Msg("MySQL version detected and recorded")
		lib.DB.Model(&lib.Version{}).Where("name = ?", "MySQL").Updates(
			lib.Version{
				Version:      mysqlVersion.Version,
				VersionMulti: string(mysqlBody),
				Status:       "installed"},
		)
		return
	}

	if oldMySQLVersion.Version != "" && mysqlVersion.Version != oldMySQLVersion.Version {
		logger.Info().Str("application", "MySQL").Str("old_version", oldMySQLVersion.Version).Str("new_version", mysqlVersion.Version).Msg("MySQL version changed")

		news := lib.News{
			Title:       fmt.Sprintf("%s sunucusunun MySQL sürümü güncellendi.", lib.GlobalConfig.Hostname),
			Description: fmt.Sprintf("%s sunucusunda MySQL, %s sürümünden %s sürümüne yükseltildi.", lib.GlobalConfig.Hostname, oldMySQLVersion.Version, mysqlVersion.Version),
		}

		lib.CreateRedmineNews(news)

		lib.DB.Model(&lib.Version{}).Where("name = ?", "MySQL").Updates(
			lib.Version{
				Version:      mysqlVersion.Version,
				VersionMulti: string(mysqlBody),
				Status:       "installed"},
		)
		return
	}

}
