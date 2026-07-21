//go:build osHealth

package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	lib "github.com/monobilisim/monokit2/lib"
)

func TestCheckSystemPowerHealth(t *testing.T) {
	lib.InitConfig(configFiles...)
	lib.InitializeDatabase()

	moduleName := "power"

	newTime := time.Now().Add(30 * time.Minute).UnixMilli()

	// [17:24:48@1] [root@monokit2-devel:~]# cat /run/systemd/shutdown/scheduled
	// USEC=1769007268634123
	// WARN_WALL=1
	// MODE=poweroff
	// UID=0
	// TTY=pts/1

	fileContent := fmt.Sprintf(`USEC=%d
WARN_WALL=1
MODE=poweroff
UID=0
TTY=pts/1
`, newTime)

	t.Log("Creating scheduled shutdown file")

	err := os.WriteFile("/run/systemd/shutdown/scheduled", []byte(fileContent), 0644)
	if err != nil {
		t.Errorf("Failed to create scheduled shutdown file: %v", err)
	}

	t.Log("Running CheckSystemPowerHealth")

	CheckSystemPowerHealth(lib.Logger)

	CheckSystemPowerHealth(lib.Logger)

	lastAlarm, err := lib.GetLastZulipAlarm(pluginName, moduleName)
	if err != nil {
		t.Errorf("Could not get last alarm from module %s: %v", moduleName, err)
	}

	if lastAlarm.Status == down || lastAlarm.Status == up {
		t.Logf("Power alarm sent successfully with status: %s", lastAlarm.Status)
	} else {
		t.Errorf("Unexpected alarm status: %s", lastAlarm.Status)
	}

	err = os.Remove("/run/systemd/shutdown/scheduled")
	if err != nil {
		t.Errorf("Failed to remove scheduled shutdown file: %v", err)
	}

	CheckSystemPowerHealth(lib.Logger)

	lastAlarms, err := lib.GetLastZulipAlarms(pluginName, moduleName)
	if err != nil {
		t.Errorf("Could not get last alarms from module %s: %v", moduleName, err)
	}

	if len(lastAlarms) != 1 {
		t.Errorf("Expected 1 alarm after recovery, got %d", len(lastAlarms))
	}

	t.Log("Power alarm sent successfully")
}
