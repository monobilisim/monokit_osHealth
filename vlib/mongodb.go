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

func MongoDBCheck(logger zerolog.Logger) {
	var mongodbVersion MongoDBVersion
	var oldMongoDBVersion lib.Version

	_, err := exec.LookPath("mongod")
	if err != nil {
		logger.Debug().Msg("mongod binary not found; MongoDB may not be installed.")
		return
	}

	/* Example output of mongod -version:
	* {"t":{"$date":"2025-12-24T14:38:10.024+03:00"},"s":"I",  "c":"-",        "id":8991200, "ctx":"main","msg":"Shuffling initializers","attr":{"seed":3159116060}}
	* db version v8.2.3
	* Build Info: {
	*   "version": "8.2.3",
	*   "gitVersion": "36f41c9c30a2f13f834d033ba03c3463c891fb01",
	*   "openSSLVersion": "OpenSSL 3.0.17 1 Jul 2025",
	*   "modules": [],
	*   "allocator": "tcmalloc-google",
	*   "environment": {
	*       "distmod": "debian12",
	*       "distarch": "x86_64",
	*       "target_arch": "x86_64"
	*   }
	* }
	 */
	out, err := exec.Command("mongod", "-version").Output()
	if err != nil {
		logger.Error().Err(err).Msg("Error executing mongod -version command")
		return
	}

	mongodbVersion.VersionFull = strings.TrimSpace(string(out))

	var jsonBody MongoDBVersion

	json.Unmarshal([]byte(strings.TrimSpace(strings.Split(mongodbVersion.VersionFull, "Info:")[1])), &jsonBody)
	if err != nil {
		logger.Error().Err(err).Msg("Error marshalling mongod version output to JSON")
		return
	}

	mongodbVersion.Version = jsonBody.Version
	mongodbVersion.GitVersion = jsonBody.GitVersion
	mongodbVersion.OpenSSLVersion = jsonBody.OpenSSLVersion
	mongodbVersion.Modules = jsonBody.Modules
	mongodbVersion.Environment.Distmod = jsonBody.Environment.Distmod
	mongodbVersion.Environment.Distarch = jsonBody.Environment.Distarch
	mongodbVersion.Environment.TargetArch = jsonBody.Environment.TargetArch

	if mongodbVersion.Version == "" {
		logger.Error().Msg("Unable to parse MongoDB version from mongod -version output")
		return
	}

	err = lib.DB.Model(&lib.Version{}).Where("name = ?", "MongoDB").First(&oldMongoDBVersion).Error
	if err != nil {
		logger.Error().Err(err).Msg("Error querying MongoDB version from database")
		return
	}

	mongodbJson, err := json.Marshal(mongodbVersion)
	if err != nil {
		logger.Error().Err(err).Msg("Error marshalling MongoDB version to JSON")
		return
	}

	if oldMongoDBVersion.Version == "" && oldMongoDBVersion.Version != mongodbVersion.Version {
		logger.Info().Str("version", mongodbVersion.Version).Msg("MongoDB installed")

		lib.DB.Model(&lib.Version{}).Where("name = ?", "MongoDB").Updates(lib.Version{
			Version:      mongodbVersion.Version,
			VersionMulti: string(mongodbJson),
			Status:       "installed",
		})
	}

	if oldMongoDBVersion.Version != "" && oldMongoDBVersion.Version != mongodbVersion.Version {
		logger.Info().Str("old_version", oldMongoDBVersion.Version).Str("new_version", mongodbVersion.Version).Msg("MongoDB version updated")

		news := lib.News{
			Title:       fmt.Sprintf("%s sunucusunun MongoDB sürümü güncellendi.", lib.GlobalConfig.Hostname),
			Description: fmt.Sprintf("%s sunucusunda MongoDB, %s sürümünden %s sürümüne yükseltildi.", lib.GlobalConfig.Hostname, oldMongoDBVersion.Version, mongodbVersion.Version),
		}

		lib.CreateRedmineNews(news)

		lib.DB.Model(&lib.Version{}).Where("name = ?", "MongoDB").Updates(
			lib.Version{
				Version:      mongodbVersion.Version,
				VersionMulti: string(mongodbJson),
				Status:       "installed",
			})
	}
}
