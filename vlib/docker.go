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

func DockerCheck(logger zerolog.Logger) {
	var dockerVersion DockerVersion
	var oldDockerVersion lib.Version

	if _, err := exec.LookPath("docker"); err != nil {
		logger.Debug().Msg("Docker CLI not found, skipping version check")
		return
	}

	out, err := exec.Command("docker", "version").CombinedOutput()
	if err != nil {
		logger.Debug().Err(err).Msg("docker version returned non-zero exit; attempting to parse partial output")
	}

	/* Example output of `docker version`:
	Client:
		Version:           28.5.2
		API version:       1.51
		Go version:        go1.24.9
		Git commit:        1.fc41
		Built:             Wed Nov  5 00:00:00 2025
		OS/Arch:           linux/amd64
		Context:           default

	Server:
		Engine:
		  	Version:          28.5.2
			API version:      1.51 (minimum version 1.24)
		  	Go version:       go1.24.9
			Git commit:       1.fc41
		  	Built:            Wed Nov  5 00:00:00 2025
			OS/Arch:          linux/amd64
		  	Experimental:     false
		containerd:
		  	Version:          1.7.29
			GitCommit:        1.fc41
		runc:
		  	Version:          1.3.3
			GitCommit:
		tini-static:
		  	Version:          0.19.0
			GitCommit:
	*/

	raw := string(out)
	ParseDockerVersion(raw, &dockerVersion)

	oldDockerVersionErr := lib.DB.Model(&lib.Version{}).Where("name = ?", "Docker").First(&oldDockerVersion).Error
	if oldDockerVersionErr != nil {
		logger.Error().Err(oldDockerVersionErr).Msg("Error querying Docker version from database")
		return
	}

	dockerJson, _ := json.Marshal(dockerVersion)

	// first time record the version
	if oldDockerVersion.Version == "" && dockerVersion.Server.Engine.Version != "" {
		logger.Info().Str("new_version", dockerVersion.Server.Engine.Version).Msg("Docker Engine version has been recorded for the first time")

		lib.DB.Model(&lib.Version{}).Where("name = ?", "Docker").Updates(lib.Version{
			Version:      dockerVersion.Server.Engine.Version,
			VersionMulti: string(dockerJson),
			Status:       "installed",
		})
		return
	}

	// version has been changed
	if oldDockerVersion.Version != "" && oldDockerVersion.Version != dockerVersion.Server.Engine.Version {
		logger.Info().Str("old_version", oldDockerVersion.Version).
			Str("new_version", dockerVersion.Server.Engine.Version).
			Msg("Docker Engine version has been updated")

		news := lib.News{
			Title:       fmt.Sprintf("%s sunucusunun Docker Engine sürümü güncellendi.", lib.GlobalConfig.Hostname),
			Description: fmt.Sprintf("%s sunucusunda Docker Engine, %s sürümünden %s sürümüne yükseltildi.", lib.GlobalConfig.Hostname, oldDockerVersion.Version, dockerVersion.Server.Engine.Version),
		}

		lib.CreateRedmineNews(news)

		lib.DB.Model(&lib.Version{}).Where("name = ?", "Docker").Updates(lib.Version{
			Version:      dockerVersion.Server.Engine.Version,
			VersionMulti: string(dockerJson),
			Status:       "installed",
		})
	}
}

func assignClientField(c *struct {
	Version    string
	APIVersion string
	GoVersion  string
	GitCommit  string
	Built      string
	OSArch     string
	Context    string
}, key, val string) {
	switch key {
	case "Version":
		c.Version = val
	case "API version":
		c.APIVersion = val
	case "Go version":
		c.GoVersion = val
	case "Git commit":
		c.GitCommit = val
	case "Built":
		c.Built = val
	case "OS/Arch":
		c.OSArch = val
	case "Context":
		c.Context = val
	}
}

func assignEngineField(e *struct {
	Version      string
	APIVersion   string
	GoVersion    string
	GitCommit    string
	Built        string
	OSArch       string
	Experimental bool
}, key, val string) {
	switch key {
	case "Version":
		e.Version = val
	case "API version":
		e.APIVersion = val
	case "Go version":
		e.GoVersion = val
	case "Git commit":
		e.GitCommit = val
	case "Built":
		e.Built = val
	case "OS/Arch":
		e.OSArch = val
	case "Experimental":
		e.Experimental = (val == "true")
	}
}

func assignSimpleVersion(v *struct {
	Version   string
	GitCommit string
}, key, val string) {
	switch key {
	case "Version":
		v.Version = val
	case "GitCommit", "Git commit":
		v.GitCommit = val
	}
}

func ParseDockerVersion(raw string, dockerVersion *DockerVersion) {
	lines := strings.Split(raw, "\n")

	var (
		section    string
		subsection string
	)

	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		// Top-level sections
		switch trimmed {
		case "Client:":
			section = "client"
			subsection = ""
			continue
		case "Server:":
			section = "server"
			subsection = ""
			continue
		}

		// Server subsections
		if section == "server" && strings.HasSuffix(trimmed, ":") {
			subsection = strings.TrimSuffix(trimmed, ":")
			continue
		}

		// Key-value line
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// Remove annotations like "(minimum version ...)"
		if i := strings.Index(val, "("); i != -1 {
			val = strings.TrimSpace(val[:i])
		}

		switch section {
		case "client":
			assignClientField(&dockerVersion.Client, key, val)

		case "server":
			switch strings.ToLower(subsection) {
			case "engine":
				assignEngineField(&dockerVersion.Server.Engine, key, val)
			case "containerd":
				assignSimpleVersion(&dockerVersion.Server.Containerd, key, val)
			case "runc":
				assignSimpleVersion(&dockerVersion.Server.Runc, key, val)
			case "tini-static":
				assignSimpleVersion(&dockerVersion.Server.TiniStatic, key, val)
			}
		}
	}
}
