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

func PostgreSQLCheck(logger zerolog.Logger) {
	var postgresqlVersion PostgreSQLVersion
	var oldPostgreSQLVersion lib.Version

	_, err := exec.LookPath("psql")
	if err != nil {
		logger.Debug().Msg("PostgreSQL binary not found, skipping version check")
		return
	}

	out, err := exec.Command("psql", "--version").Output()
	if err != nil {
		logger.Error().Err(err).Msg("Error getting PostgreSQL version")
		return
	}

	// psql (PostgreSQL) 16.11
	// psql (PostgreSQL) 13.3 (Ubuntu 13.3-1.pgdg20.04+1)
	postgresqlVersion.VersionFull = strings.TrimSpace(string(out))

	if postgresqlVersion.VersionFull == "" {
		logger.Error().Str("output", postgresqlVersion.VersionFull).Msg("psql --version returns empty")
		return
	}

	parts := strings.Fields(postgresqlVersion.VersionFull)
	postgresqlVersion.Version = parts[2]

	err = lib.DB.Model(&lib.Version{}).Where("name = ?", "PostgreSQL").First(&oldPostgreSQLVersion).Error
	if err != nil {
		logger.Error().Err(err).Str("component", "osHealth").Str("application", "PostgreSQL").Str("operation", "query_version").Msg("Error querying PostgreSQL version from database")
		return
	}

	postgresqlBody, _ := json.Marshal(postgresqlVersion)

	if oldPostgreSQLVersion.Version == "" && postgresqlVersion.Version != "" {
		logger.Info().Str("application", "PostgreSQL").Str("version", postgresqlVersion.Version).Msg("PostgreSQL version detected and recorded")
		lib.DB.Model(&lib.Version{}).Where("name = ?", "PostgreSQL").Updates(
			lib.Version{
				Version:      postgresqlVersion.Version,
				VersionMulti: string(postgresqlBody),
				Status:       "installed"},
		)
		return
	}

	if oldPostgreSQLVersion.Version != "" && postgresqlVersion.Version != oldPostgreSQLVersion.Version {
		logger.Info().Str("application", "PostgreSQL").Str("old_version", oldPostgreSQLVersion.Version).Str("new_version", postgresqlVersion.Version).Msg("PostgreSQL version changed")

		news := lib.News{
			Title:       fmt.Sprintf("%s sunucusunun PostgreSQL sürümü güncellendi.", lib.GlobalConfig.Hostname),
			Description: fmt.Sprintf("%s sunucusunda PostgreSQL, %s sürümünden %s sürümüne yükseltildi.", lib.GlobalConfig.Hostname, oldPostgreSQLVersion.Version, postgresqlVersion.Version),
		}

		lib.CreateRedmineNews(news)

		lib.DB.Model(&lib.Version{}).Where("name = ?", "PostgreSQL").Updates(
			lib.Version{
				Version:      postgresqlVersion.Version,
				VersionMulti: string(postgresqlBody),
				Status:       "installed"},
		)
		return
	}
}
