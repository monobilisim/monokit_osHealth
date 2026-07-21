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

func HAProxyCheck(logger zerolog.Logger) {
	var haproxyVersion HAProxyVersion
	var oldHAProxyVersion lib.Version

	if _, err := exec.LookPath("haproxy"); err != nil {
		logger.Debug().Msg("haproxy binary not found, skipping version check")
		return
	}

	/* Example output of `haproxy -v`:
	* RHEL / Fedora:
	* HAProxy version 3.0.5-8e879a5 2024/09/19 - https://haproxy.org/
	* Status: long-term supported branch - will stop receiving fixes around Q2 2029.
	* Known bugs: http://www.haproxy.org/bugs/bugs-3.0.5.html
	* Running on: Linux 6.17.10-100.fc41.x86_64 #1 SMP PREEMPT_DYNAMIC Mon Dec  1 16:10:21 UTC 2025 x86_64
	* Debian:
	* HAProxy version 2.6.12-1+deb12u3 2025/10/03 - https://haproxy.org/
	* Status: long-term supported branch - will stop receiving fixes around Q2 2027.
	* Known bugs: http://www.haproxy.org/bugs/bugs-2.6.12.html
	* Running on: Linux 6.1.0-38-amd64 #1 SMP PREEMPT_DYNAMIC Debian 6.1.147-1 (2025-08-02) x86_64
	 */
	out, err := exec.Command("haproxy", "-v").CombinedOutput()
	if err != nil {
		logger.Debug().Err(err).Msg("haproxy -v returned non-zero; attempting to parse output")
	}

	text := strings.TrimSpace(string(out))
	haproxyVersion.VersionFull = text

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		// match HAProxy version line
		if strings.Contains(line, "HAProxy") || strings.Contains(line, "HA-Proxy") && strings.Contains(line, "version") {
			/* line:
			 * HAProxy version 3.0.5-8e879a5 2024/09/19 - https://haproxy.org/ // can be HA-Proxy
			 * HAProxy version 2.6.12-1+deb12u3 2025/10/03 - https://haproxy.org/ // can be HA-Proxy
			 */
			parts := strings.Fields(line)
			// 3.0.5-8e879a5 2024/09/19
			// 2.6.12-1+deb12u3
			actualVersionPart := parts[2]

			if strings.Contains(actualVersionPart, "+deb") {
				haproxyVersion.Version = strings.Split(actualVersionPart, "+deb")[0]
			} else {
				haproxyVersion.Version = strings.Split(actualVersionPart, "-")[0]
			}
		}

		if strings.Contains(line, "Status") {
			// Status: long-term supported branch - will stop receiving fixes around Q2 2029.
			haproxyVersion.Status = strings.TrimSpace(strings.Split(line, "Status:")[1])
		}

		if strings.Contains(line, "Known") {
			// Known bugs: http://www.haproxy.org/bugs/bugs-2.6.12.html
			haproxyVersion.KnownBugs = strings.TrimSpace(strings.Split(line, "bugs:")[1])
		}

		if strings.Contains(line, "Running") {
			// Running on: Linux 6.17.10-100.fc41.x86_64 #1 SMP PREEMPT_DYNAMIC Mon Dec  1 16:10:21 UTC 2025 x86_64
			// Running on: Linux 6.1.0-38-amd64 #1 SMP PREEMPT_DYNAMIC Debian 6.1.147-1 (2025-08-02) x86_64
			haproxyVersion.RunningOn = strings.TrimSpace(strings.SplitN(line, "on:", 2)[1])
		}
	}

	err = lib.DB.Model(&lib.Version{}).Where("name = ?", "HAProxy").First(&oldHAProxyVersion).Error
	if err != nil {
		logger.Error().Err(err).Str("component", "osHealth").Str("application", "HAProxy").Str("operation", "query_version").Msg("Error querying HAProxy version from database")
		return
	}

	haproxyBody, _ := json.Marshal(haproxyVersion)

	if oldHAProxyVersion.Version == "" && haproxyVersion.Version != "" {
		logger.Info().Str("application", "HAProxy").Str("version", haproxyVersion.Version).Msg("HAProxy version detected and recorded")
		lib.DB.Model(&lib.Version{}).Where("name = ?", "HAProxy").Updates(
			lib.Version{
				Version:      haproxyVersion.Version,
				VersionMulti: string(haproxyBody),
				Status:       "installed",
			})
		return
	}

	if oldHAProxyVersion.Version != "" && oldHAProxyVersion.Version != haproxyVersion.Version {
		logger.Info().Str("application", "HAProxy").
			Str("old_version", oldHAProxyVersion.Version).
			Str("new_version", haproxyVersion.Version).
			Msg("HAProxy version changed")

		news := lib.News{
			Title:       fmt.Sprintf("%s sunucusunun HAProxy sürümü güncellendi.", lib.GlobalConfig.Hostname),
			Description: fmt.Sprintf("%s sunucusunda HAProxy, %s sürümünden %s sürümüne yükseltildi.", lib.GlobalConfig.Hostname, oldHAProxyVersion.Version, haproxyVersion.Version),
		}

		lib.CreateRedmineNews(news)

		lib.DB.Model(&lib.Version{}).Where("name = ?", "HAProxy").Updates(
			lib.Version{
				Version:      haproxyVersion.Version,
				VersionMulti: string(haproxyBody),
				Status:       "installed",
			})
		return
	}
}
