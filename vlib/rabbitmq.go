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

func RabbitMQCheck(logger zerolog.Logger) {
	var rabbitmqVersion RabbitMQVersion
	var oldRabbitMQVersion lib.Version

	_, err := exec.LookPath("rabbitmq-diagnostics")
	if err != nil {
		logger.Debug().Msg("RabbitMQ binaries not found, skipping version check")
		return
	}

	out, err := exec.Command("rabbitmq-diagnostics", "status", "--formatter", "json").Output()
	if err != nil {
		logger.Error().Err(err).Msg("Error getting RabbitMQ version via rabbitmq-diagnostics")
		return
	}

	rabbitmqVersion.VersionFull = strings.TrimSpace(string(out))

	if rabbitmqVersion.VersionFull == "" {
		logger.Error().Str("output", rabbitmqVersion.VersionFull).Msg("rabbitmq-diagnostics status returns empty")
		return
	}

	err = json.Unmarshal([]byte(rabbitmqVersion.VersionFull), &rabbitmqVersion)
	if err != nil {
		logger.Error().Err(err).Msg("Error parsing RabbitMQ version JSON")
		return
	}

	// actual server version
	rabbitmqVersion.Version = rabbitmqVersion.RabbitMQVersion

	err = lib.DB.Model(&lib.Version{}).Where("name = ?", "RabbitMQ").First(&oldRabbitMQVersion).Error
	if err != nil {
		logger.Error().Err(err).Str("component", "osHealth").Str("application", "RabbitMQ").Str("operation", "query_version").Msg("Error querying RabbitMQ version from database")
		return
	}

	rabbitmqBody, _ := json.Marshal(rabbitmqVersion)

	logger.Debug().Str("rabbitmq_version_full", string(rabbitmqBody)).Msg("Parsed RabbitMQ version JSON")

	if oldRabbitMQVersion.Version == "" && rabbitmqVersion.Version != "" {
		logger.Info().Str("application", "RabbitMQ").Str("version", rabbitmqVersion.Version).Msg("RabbitMQ version detected and recorded")
		lib.DB.Model(&lib.Version{}).Where("name = ?", "RabbitMQ").Updates(
			lib.Version{
				Version:      rabbitmqVersion.Version,
				VersionMulti: string(rabbitmqBody),
				Status:       "installed"},
		)
		return
	}

	if oldRabbitMQVersion.Version != "" && rabbitmqVersion.Version != oldRabbitMQVersion.Version {
		logger.Info().Str("application", "RabbitMQ").Str("old_version", oldRabbitMQVersion.Version).Str("new_version", rabbitmqVersion.Version).Msg("RabbitMQ version changed")

		news := lib.News{
			Title:       fmt.Sprintf("%s sunucusunun RabbitMQ sürümü güncellendi.", lib.GlobalConfig.Hostname),
			Description: fmt.Sprintf("%s sunucusunda RabbitMQ, %s sürümünden %s sürümüne yükseltildi.", lib.GlobalConfig.Hostname, oldRabbitMQVersion.Version, rabbitmqVersion.Version),
		}

		lib.CreateRedmineNews(news)

		lib.DB.Model(&lib.Version{}).Where("name = ?", "RabbitMQ").Updates(
			lib.Version{
				Version:      rabbitmqVersion.Version,
				VersionMulti: string(rabbitmqBody),
				Status:       "installed"},
		)
	}
}
