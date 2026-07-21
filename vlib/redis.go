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

func RedisCheck(logger zerolog.Logger) {
	var redisVersion RedisVersion
	var oldRedisVersion lib.Version

	_, err := exec.LookPath("redis-server")
	if err != nil {
		logger.Debug().Msg("Redis server binary not found, skipping version check")
		return
	}

	// Output example of redis-server --version:
	// Valkey server v=8.0.6 sha=00000000:0 malloc=jemalloc-5.3.0 bits=64 build=cc4ea19b99ae73a7
	// Redis server v=7.0.15 sha=00000000:0 malloc=jemalloc-5.3.0 bits=64 build=5281cccdf7ef82d6
	out, err := exec.Command("redis-server", "--version").Output()
	if err != nil {
		logger.Error().Err(err).Msg("Error getting Redis version")
		return
	}

	redisVersion.VersionFull = strings.TrimSpace(string(out))

	if redisVersion.VersionFull == "" {
		logger.Error().Str("output", redisVersion.VersionFull).Msg("redis-server --version returns empty")
		return
	}

	if strings.Contains(strings.ToLower(redisVersion.VersionFull), "valkey") {
		logger.Debug().Msg("Detected Valkey Redis installation, skipping Redis version check")
		return
	}

	fields := strings.Fields(redisVersion.VersionFull)
	for _, field := range fields {
		if strings.HasPrefix(field, "v=") {
			redisVersion.Version = strings.TrimSpace(strings.TrimPrefix(field, "v="))
			break
		}
	}

	if redisVersion.Version == "" {
		logger.Error().Str("output", redisVersion.VersionFull).Msg("Could not parse Redis version")
		return
	}

	err = lib.DB.Model(&lib.Version{}).Where("name = ?", "Redis").First(&oldRedisVersion).Error
	if err != nil {
		logger.Error().Err(err).Str("component", "osHealth").Str("application", "Redis").Str("operation", "query_version").Msg("Error querying Redis version from database")
		return
	}

	redisBody, _ := json.Marshal(redisVersion)

	if oldRedisVersion.Version == "" && redisVersion.Version != "" {
		logger.Info().Str("application", "Redis").Str("version", redisVersion.Version).Msg("Redis version detected and recorded")
		lib.DB.Model(&lib.Version{}).Where("name = ?", "Redis").Updates(
			lib.Version{
				Version:      redisVersion.Version,
				VersionMulti: string(redisBody),
				Status:       "installed"},
		)
		return
	}

	if oldRedisVersion.Version != "" && redisVersion.Version != oldRedisVersion.Version {
		logger.Info().Str("application", "Redis").Str("old_version", oldRedisVersion.Version).Str("new_version", redisVersion.Version).Msg("Redis version changed")

		news := lib.News{
			Title:       fmt.Sprintf("%s sunucusunun Redis sürümü güncellendi.", lib.GlobalConfig.Hostname),
			Description: fmt.Sprintf("%s sunucusunda Redis, %s sürümünden %s sürümüne yükseltildi.", lib.GlobalConfig.Hostname, oldRedisVersion.Version, redisVersion.Version),
		}

		lib.CreateRedmineNews(news)

		lib.DB.Model(&lib.Version{}).Where("name = ?", "Redis").Updates(
			lib.Version{
				Version:      redisVersion.Version,
				VersionMulti: string(redisBody),
				Status:       "installed"},
		)
	}
}
