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

func PostalCheck(logger zerolog.Logger) {
	var postalVersion PostalVersion
	var oldPostalVersion lib.Version
	postalCliPresent := false
	dockerPresent := false
	dockerContainersPresent := false

	_, err := exec.LookPath("postal")
	if err == nil {
		logger.Debug().Msg("Postal CLI found.")
		postalCliPresent = true
	}

	_, err = exec.Command("docker", "ps").Output()
	if err == nil {
		logger.Debug().Msg("Docker CLI found.")
		dockerPresent = true
	}

	if dockerPresent {
		out, err := exec.Command("docker", "ps", "--format", "{{.Image}}").Output()
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			for _, line := range lines {
				if strings.Contains(strings.ToLower(line), "postal") {
					dockerContainersPresent = true
					break
				}
			}
		}
	}

	if postalCliPresent {
		logger.Debug().Msg("Postal CLI detected; proceeding with version check.")

		out, err := exec.Command("postal", "version").Output()
		if err != nil {
			logger.Error().Err(err).Msg("Error getting Postal version via CLI")
		}

		/* Example output of postal version:
		* [+] Building 0.0s (0/0)                                                          docker:default
		* [+] Building 0.0s (0/0)                                                          docker:default
		* Loading config from /config/postal.yml
		* 3.4.0
		* Second type of output: maybe for older systems?
		* [+] Building 0.0s (0/0)                                                          docker:default
		* [+] Building 0.0s (0/0)                                                          docker:default
		* Loading config from /config/postal.yml
		* Postal v3.4.0
		 */
		postalVersion.VersionFull = strings.TrimSpace(string(out))

		if postalVersion.VersionFull == "" {
			logger.Error().Str("output", postalVersion.VersionFull).Msg("postal version returns empty")
		}

		lines := strings.Split(postalVersion.VersionFull, "\n")
		lastLine := strings.TrimSpace(lines[len(lines)-1])

		if strings.Contains(strings.ToLower(lastLine), "postal") {
			// Postal v3.4.0
			parts := strings.Fields(postalVersion.VersionFull)
			postalVersion.Version = strings.TrimPrefix(parts[1], "v")
		} else {
			// 3.4.0
			postalVersion.Version = lastLine
		}
	}

	// Fallback for older versions: if postal CLI is not present but config is, try to detect via other means
	if postalVersion.Version == "" && dockerContainersPresent {
		logger.Debug().Msg("Postal configuration detected; proceeding with version check.")

		out, err := exec.Command("docker", "ps", "--format", "{{.Image}}").Output()
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			for _, line := range lines {
				if strings.Contains(strings.ToLower(line), "postal") {
					/* Example output lines (just a single one):
					* ghcr.io/postalserver/postal:2.1.0
					* ghcr.io/mudrockdev/postal:3.4.0
					* ghcr.io/monobilisim/postal:3.4.0
					 */
					parts := strings.Split(line, ":")
					if len(parts) == 2 {
						tag := parts[1]
						postalVersion.Version = tag
						break
					}
				}
			}
		}
	}

	err = lib.DB.Model(&lib.Version{}).Where("name = ?", "Postal").First(&oldPostalVersion).Error
	if err != nil {
		logger.Error().Err(err).Str("component", "osHealth").Str("application", "Postal").Str("operation", "query_version").Msg("Error querying Postal version from database")
		return
	}

	postalBody, _ := json.Marshal(postalVersion)

	if oldPostalVersion.Version == "" && postalVersion.Version != "" {
		logger.Info().Str("application", "Postal").Str("version", postalVersion.Version).Msg("Postal version detected and recorded")
		lib.DB.Model(&lib.Version{}).Where("name = ?", "Postal").Updates(
			lib.Version{
				Version:      postalVersion.Version,
				VersionMulti: string(postalBody),
				Status:       "installed"},
		)
		return
	}

	if oldPostalVersion.Version != "" && postalVersion.Version != oldPostalVersion.Version {
		logger.Info().Str("application", "Postal").Str("old_version", oldPostalVersion.Version).Str("new_version", postalVersion.Version).Msg("Postal version changed")

		news := lib.News{
			Title:       fmt.Sprintf("%s sunucusunun Postal sürümü güncellendi.", lib.GlobalConfig.Hostname),
			Description: fmt.Sprintf("%s sunucusunda Postal, %s sürümünden %s sürümüne yükseltildi.", lib.GlobalConfig.Hostname, oldPostalVersion.Version, postalVersion.Version),
		}

		lib.CreateRedmineNews(news)

		lib.DB.Model(&lib.Version{}).Where("name = ?", "Postal").Updates(
			lib.Version{
				Version:      postalVersion.Version,
				VersionMulti: string(postalBody),
				Status:       "installed"},
		)
	}
}
