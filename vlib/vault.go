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

func VaultCheck(logger zerolog.Logger) {
	var vaultVersion VaultVersion
	var oldVaultVersion lib.Version

	_, err := exec.LookPath("vault")
	if err != nil {
		logger.Debug().Msg("Vault binary not found, skipping version check")
		return
	}

	cmd := exec.Command("vault", "version")
	out, err := cmd.Output()
	if err != nil {
		logger.Error().Err(err).Msg("Error getting Vault version")
		return
	}

	/* Example outputs of `vault version`:
	* Vault v1.2.3
	* Vault v1.2.3, built 2022-05-03T08:34:11Z
	* Vault v1.21.1 (2453aac2638a6ae243341b4e0657fd8aea1cbf18), built 2025-11-18T13:04:32Z
	 */
	vaultVersion.VersionFull = strings.TrimSpace(string(out))

	if vaultVersion.VersionFull == "" {
		logger.Error().Str("output", vaultVersion.VersionFull).Msg("vault version returns empty")
		return
	}

	parts := strings.Fields(vaultVersion.VersionFull)

	if len(parts) == 2 {
		// extracts 1.2.3 from Vault v1.2.3
		vaultVersion.Version = strings.TrimPrefix(strings.TrimSpace(parts[1]), "v")
	}

	if len(parts) == 4 {
		// extracts 1.2.3 from Vault v1.2.3, built 2022-05-03T08:34:11Z
		vaultVersion.Version = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(parts[1]), "v"), ",")
	}

	if len(parts) == 5 {
		// extracts 1.21.1 from Vault v1.21.1 (2453aac2638a6ae243341b4e0657fd8aea1cbf18), built 2025-11-18T13:04:32Z
		vaultVersion.Version = strings.TrimPrefix(parts[1], "v")
	}

	if vaultVersion.Version == "" {
		logger.Error().Str("output", vaultVersion.VersionFull).Msg("Could not parse Vault version")
		return
	}

	err = lib.DB.Model(&lib.Version{}).Where("name = ?", "Vault").First(&oldVaultVersion).Error
	if err != nil {
		logger.Error().Err(err).Str("component", "osHealth").Str("application", "Vault").Str("operation", "query_version").Msg("Error querying Vault version from database")
		return
	}

	vaultBody, _ := json.Marshal(vaultVersion)

	if oldVaultVersion.Version == "" && vaultVersion.Version != "" {
		logger.Info().Str("application", "Vault").Str("version", vaultVersion.Version).Msg("Vault version detected and recorded")
		lib.DB.Model(&lib.Version{}).Where("name = ?", "Vault").Updates(
			lib.Version{
				Version:      vaultVersion.Version,
				VersionMulti: string(vaultBody),
				Status:       "installed"},
		)
		return
	}

	if oldVaultVersion.Version != "" && vaultVersion.Version != oldVaultVersion.Version {
		logger.Info().Str("application", "Vault").Str("old_version", oldVaultVersion.Version).Str("new_version", vaultVersion.Version).Msg("Vault version changed")

		news := lib.News{
			Title:       fmt.Sprintf("%s sunucusunun Vault sürümü güncellendi.", lib.GlobalConfig.Hostname),
			Description: fmt.Sprintf("%s sunucusunda Vault, %s sürümünden %s sürümüne yükseltildi.", lib.GlobalConfig.Hostname, oldVaultVersion.Version, vaultVersion.Version),
		}

		lib.CreateRedmineNews(news)

		lib.DB.Model(&lib.Version{}).Where("name = ?", "Vault").Updates(
			lib.Version{
				Version:      vaultVersion.Version,
				VersionMulti: string(vaultBody),
				Status:       "updated"},
		)
		return
	}
}
