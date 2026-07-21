//go:build osHealth

package vlib

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/monobilisim/monokit2/lib"
	"github.com/rs/zerolog"
)

func ZabbixCheck(logger zerolog.Logger) {
	var zabbixVersion ZabbixVersion
	var oldZabbixVersion lib.Version

	if _, err := exec.LookPath("zabbix_server"); err != nil {
		logger.Debug().Msg("zabbix_server not found, skipping Zabbix version check")
		return
	}

	/* Example output of `zabbix_server --version`:
	* zabbix_server (Zabbix) 7.0.21
	* Revision 8810935a39b 28 October 2025, compilation time: Nov  3 2025 00:00:00
	*
	* Copyright (C) 2025 Zabbix SIA
	* License AGPLv3: GNU Affero General Public License version 3 <https://www.gnu.org/licenses/>.
	* This is free software: you are free to change and redistribute it according to
	* the license. There is NO WARRANTY, to the extent permitted by law.
	*
	* This product includes software developed by the OpenSSL Project
	* for use in the OpenSSL Toolkit (http://www.openssl.org/).
	*
	* Compiled with OpenSSL 3.2.6 30 Sep 2025
	* Running with OpenSSL 3.2.6 30 Sep 2025
	 */
	out, err := exec.Command("zabbix_server", "--version").Output()
	if err != nil {
		logger.Error().Err(err).Msg("Error getting Zabbix version")
		return
	}

	zabbixVersion.VersionFull = strings.TrimSpace(string(out))

	if zabbixVersion.VersionFull == "" {
		logger.Error().Str("output", zabbixVersion.VersionFull).Msg("zabbix_server --version returns empty")
		return
	}

	for _, line := range strings.Split(zabbixVersion.VersionFull, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "zabbix_server") {
			parts := strings.Fields(trimmed)
			zabbixVersion.Version = parts[2]
			break
		}
	}

	err = lib.DB.Model(&lib.Version{}).Where("name = ?", "Zabbix").First(&oldZabbixVersion).Error
	if err != nil {
		logger.Error().Err(err).Str("component", "osHealth").Str("application", "Zabbix").Str("operation", "query_version").Msg("Error querying Zabbix version from database")
		return
	}

	zabbixBody, _ := json.Marshal(zabbixVersion)

	if oldZabbixVersion.Version == "" && zabbixVersion.Version != "" {
		logger.Info().Str("application", "Zabbix").Str("version", zabbixVersion.Version).Msg("Zabbix version detected and recorded")
		lib.DB.Model(&lib.Version{}).Where("name = ?", "Zabbix").Updates(
			lib.Version{
				Version:      zabbixVersion.Version,
				VersionMulti: string(zabbixBody),
				Status:       "installed"},
		)
		return
	}

	if oldZabbixVersion.Version != "" && oldZabbixVersion.Version != zabbixVersion.Version {
		logger.Info().Str("application", "Zabbix").Str("old_version", oldZabbixVersion.Version).Str("new_version", zabbixVersion.Version).Msg("Zabbix version updated")

		news := lib.News{
			Title:       fmt.Sprintf("%s sunucusunun Zabbix sürümü güncellendi.", lib.GlobalConfig.Hostname),
			Description: fmt.Sprintf("%s sunucusunda Zabbix, %s sürümünden %s sürümüne yükseltildi.", lib.GlobalConfig.Hostname, oldZabbixVersion.Version, zabbixVersion.Version),
		}

		lib.CreateRedmineNews(news)

		lib.DB.Model(&lib.Version{}).Where("name = ?", "Zabbix").Updates(
			lib.Version{
				Version:      zabbixVersion.Version,
				VersionMulti: string(zabbixBody),
				Status:       "installed"},
		)
		return
	}
}
