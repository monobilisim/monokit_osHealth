//go:build osHealth

package vlib

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	lib "github.com/monobilisim/monokit2/lib"
	"github.com/rs/zerolog"
)

var prometheusVersionRegex = regexp.MustCompile(`version ([^ ]+)`)

func PrometheusCheck(logger zerolog.Logger) {
	var prometheusVersion PrometheusVersion
	var oldPrometheusVersion lib.Version

	_, err := exec.LookPath("prometheus")
	if err != nil {
		logger.Debug().Msg("Prometheus binary not found, skipping version check")
		return
	}

	out, err := exec.Command("prometheus", "--version").Output()
	if err != nil {
		logger.Error().Err(err).Msg("Error getting Prometheus version")
		return
	}

	/* Example output of `prometheus --version`:
	* prometheus, version 3.8.1 (branch: HEAD, revision: ed753444ffec98097399d0cfa9073c70a840b812)
	*   build user:       root@fcf67c824170
	*   build date:       20251216-08:46:44
	*   go version:       go1.25.5
	*   platform:         linux/amd64
	*   tags:             netgo,builtinassets
	 */
	prometheusVersion.VersionFull = strings.TrimSpace(string(out))

	if prometheusVersion.VersionFull == "" {
		logger.Error().Str("output", prometheusVersion.VersionFull).Msg("prometheus --version returns empty")
		return
	}

	for _, line := range strings.Split(prometheusVersion.VersionFull, "\n") {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "prometheus,") {
			parts := strings.Fields(trimmed)
			prometheusVersion.Version = strings.TrimSpace(parts[2])
		}

		if strings.HasPrefix(trimmed, "build user:") {
			prometheusVersion.BuildUser = strings.TrimSpace(strings.TrimPrefix(trimmed, "build user:"))
		}

		if strings.HasPrefix(trimmed, "build date:") {
			prometheusVersion.BuildDate = strings.TrimSpace(strings.TrimPrefix(trimmed, "build date:"))
		}

		if strings.HasPrefix(trimmed, "go version:") {
			prometheusVersion.GoVersion = strings.TrimSpace(strings.TrimPrefix(trimmed, "go version:"))
		}

		if strings.HasPrefix(trimmed, "platform:") {
			prometheusVersion.Platform = strings.TrimSpace(strings.TrimPrefix(trimmed, "platform:"))
		}

		if strings.HasPrefix(trimmed, "tags:") {
			prometheusVersion.Tags = strings.Split(strings.TrimSpace(strings.TrimPrefix(trimmed, "tags:")), ",")
		}
	}

	err = lib.DB.Model(&lib.Version{}).Where("name = ?", "Prometheus").First(&oldPrometheusVersion).Error
	if err != nil {
		logger.Error().Err(err).Str("component", "osHealth").Str("application", "Prometheus").Str("operation", "query_version").Msg("Error querying Prometheus version from database")
		return
	}

	prometheusJson, _ := json.Marshal(prometheusVersion)

	if oldPrometheusVersion.Version == "" && prometheusVersion.Version != "" {
		logger.Info().Str("application", "Prometheus").Str("version", prometheusVersion.Version).Msg("Prometheus version detected and recorded")
		lib.DB.Model(&lib.Version{}).Where("name = ?", "Prometheus").Updates(
			lib.Version{
				Version:      prometheusVersion.Version,
				VersionMulti: string(prometheusJson),
				Status:       "installed",
			},
		)
		return
	}

	if oldPrometheusVersion.Version != "" && prometheusVersion.Version != oldPrometheusVersion.Version {
		logger.Info().Str("application", "Prometheus").Str("old_version", oldPrometheusVersion.Version).Str("new_version", prometheusVersion.Version).Msg("Prometheus version changed")

		news := lib.News{
			Title:       fmt.Sprintf("%s sunucusunun Prometheus sürümü güncellendi.", lib.GlobalConfig.Hostname),
			Description: fmt.Sprintf("%s sunucusunda Prometheus, %s sürümünden %s sürümüne yükseltildi.", lib.GlobalConfig.Hostname, oldPrometheusVersion.Version, prometheusVersion.Version),
		}

		lib.CreateRedmineNews(news)

		lib.DB.Model(&lib.Version{}).Where("name = ?", "Prometheus").Updates(
			lib.Version{
				Version:      prometheusVersion.Version,
				VersionMulti: string(prometheusJson),
				Status:       "installed",
			},
		)
	}
}
