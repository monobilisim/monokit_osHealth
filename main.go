//go:build osHealth

package main

import (
	"os"
	"os/exec"
	"strings"

	lib "github.com/monobilisim/monokit2/lib"
)

// comes from -ldflags "-X 'main.version=version'" flag in ci build
var version string
var pluginName string = "osHealth"
var up string = "up"
var down string = "down"
var configFiles []string = []string{"os.yml"}

func main() {
	if len(os.Args) > 1 {
		lib.HandleCommonPluginArgs(os.Args, version, configFiles)
		return
	}

	err := lib.InitConfig(configFiles...)
	if err != nil {
		panic("Failed to initialize config: " + err.Error())
	}

	logger, err := lib.InitLogger()
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}

	lib.InitializeDatabase()

	logger.Info().Msg("Starting OS Health monitoring plugin...")

	// checks supported application versions and reports when updated
	if lib.OsHealthConfig.VersionAlarm.Enabled {
		CheckApplicationVersion(logger)
	}

	// checks system load
	if lib.OsHealthConfig.SystemLoadAlarm.Enabled {
		CheckSystemLoad(logger)
	}

	// checks system RAM usage
	if lib.OsHealthConfig.RamUsageAlarm.Enabled {
		CheckSystemRAM(logger)
	}

	// checks system disk usage
	if lib.OsHealthConfig.DiskUsageAlarm.Enabled {
		CheckSystemDisk(logger)
	}

	// checks ZFS pool health and usage
	if lib.OsHealthConfig.DiskUsageAlarm.Enabled && hasZFS() {
		CheckSystemDiskZFS(logger)
	}

	// checks systemd services status
	if lib.OsHealthConfig.ServiceHealthAlarm.Enabled && hasSystemd() {
		CheckSystemInit(logger)
	}

	if lib.OsHealthConfig.PowerAlarm.Enabled {
		CheckSystemPowerHealth(logger)
	}
}

// checks if there is an active ZFS pool
func hasZFS() bool {
	_, err := exec.LookPath("zpool")
	if err != nil {
		return false
	}

	cmd := exec.Command("zpool", "list", "-H")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return len(strings.TrimSpace(string(output))) > 0
}

// checks if systemd is available
func hasSystemd() bool {
	_, err := exec.LookPath("systemctl")
	if err != nil {
		return false
	}
	return true
}
