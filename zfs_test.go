// go:build osHealth

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	lib "github.com/monobilisim/monokit2/lib"
)

func TestCheckSystemDiskZFS(t *testing.T) {
	lib.InitConfig(configFiles...)
	lib.InitializeDatabase()

	mockZpoolHealthy := `#!/usr/bin/env bash
echo "pool1 ONLINE 0%"
echo "pool2  ONLINE  8%"`

	mockZpoolUnhealthy := `#!/usr/bin/env bash
echo "pool1  DEGRADED  0%"
echo "pool2 ONLINE 98%"`

	zpoolPath := "/usr/local/bin/zpool"
	zpoolPathExists, err := os.Stat(zpoolPath)
	if os.IsNotExist(err) || zpoolPathExists.IsDir() {
		err := os.MkdirAll(filepath.Dir(zpoolPath), 0755)
		if err != nil {
			t.Errorf("Failed to create directory for mock zpool script: %v", err)
		}
	}

	err = os.WriteFile(zpoolPath, []byte(mockZpoolUnhealthy), 0755)
	if err != nil {
		t.Errorf("Failed to write mock zpool script: %v", err)
	}

	t.Log("Testing unhealthy ZFS pools...")

	CheckSystemDiskZFS(lib.Logger)

	time.Sleep(5 * time.Second)

	// down test
	moduleName := "zfsHealth"

	lastAlarm, err := lib.GetLastZulipAlarm(pluginName, moduleName)
	if err != nil {
		t.Errorf("Failed to get last Zulip alarm: %v", err)
	}

	if lastAlarm.Status != down {
		t.Errorf("Expected last alarm status to be 'down', got '%s'", lastAlarm.Status)
	}

	lastIssue, err := lib.GetLastRedmineIssue(pluginName, moduleName)
	if err != nil {
		t.Errorf("Failed to get last Redmine issue: %v", err)
	}

	if lastIssue.Status != down {
		t.Errorf("Expected last issue status to be 'down', got '%s'", lastIssue.Status)
	}

	moduleName = "zfsCapacity"

	lastAlarm, err = lib.GetLastZulipAlarm(pluginName, moduleName)
	if err != nil {
		t.Errorf("Failed to get last Zulip alarm: %v", err)
	}

	if lastAlarm.Status != down {
		t.Errorf("Expected last alarm status to be 'down', got '%s'", lastAlarm.Status)
	}

	lastIssue, err = lib.GetLastRedmineIssue(pluginName, moduleName)
	if err != nil {
		t.Errorf("Failed to get last Redmine issue: %v", err)
	}

	if lastIssue.Status != down {
		t.Errorf("Expected last issue status to be 'down', got '%s'", lastIssue.Status)
	}

	t.Log("Testing unhealthy ZFS pools again...")

	CheckSystemDiskZFS(lib.Logger)

	time.Sleep(5 * time.Second)

	// issue still occurring
	moduleName = "zfsHealth"

	lastAlarm, err = lib.GetLastZulipAlarm(pluginName, moduleName)
	if err != nil {
		t.Errorf("Failed to get last Zulip alarm: %v", err)
	}

	if lastAlarm.Status != down {
		t.Errorf("Expected last alarm status to be 'down', got '%s'", lastAlarm.Status)
	}

	lastIssue, err = lib.GetLastRedmineIssue(pluginName, moduleName)
	if err != nil {
		t.Errorf("Failed to get last Redmine issue: %v", err)
	}

	if lastIssue.Status != down {
		t.Errorf("Expected last issue status to be 'down', got '%s'", lastIssue.Status)
	}

	moduleName = "zfsCapacity"

	lastAlarm, err = lib.GetLastZulipAlarm(pluginName, moduleName)
	if err != nil {
		t.Errorf("Failed to get last Zulip alarm: %v", err)
	}

	if lastAlarm.Status != down {
		t.Errorf("Expected last alarm status to be 'down', got '%s'", lastAlarm.Status)
	}

	lastIssue, err = lib.GetLastRedmineIssue(pluginName, moduleName)
	if err != nil {
		t.Errorf("Failed to get last Redmine issue: %v", err)
	}

	if lastIssue.Status != down {
		t.Errorf("Expected last issue status to be 'down', got '%s'", lastIssue.Status)
	}

	err = os.WriteFile(zpoolPath, []byte(mockZpoolHealthy), 0755)
	if err != nil {
		t.Errorf("Failed to write mock zpool script: %v", err)
	}

	t.Log("Testing healthy ZFS pools...")

	CheckSystemDiskZFS(lib.Logger)

	time.Sleep(5 * time.Second)

	// up test
	moduleName = "zfsHealth"

	lastAlarm, err = lib.GetLastZulipAlarm(pluginName, moduleName)
	if err != nil {
		t.Errorf("Failed to get last Zulip alarm: %v", err)
	}

	if lastAlarm.Status != up {
		t.Errorf("Expected last alarm status to be 'up', got '%s'", lastAlarm.Status)
	}

	lastIssue, err = lib.GetLastRedmineIssue(pluginName, moduleName)
	if err != nil {
		t.Errorf("Failed to get last Redmine issue: %v", err)
	}

	if lastIssue.Status != up {
		t.Errorf("Expected last issue status to be 'up', got '%s'", lastIssue.Status)
	}

	moduleName = "zfsCapacity"

	lastAlarm, err = lib.GetLastZulipAlarm(pluginName, moduleName)
	if err != nil {
		t.Errorf("Failed to get last Zulip alarm: %v", err)
	}

	if lastAlarm.Status != up {
		t.Errorf("Expected last alarm status to be 'up', got '%s'", lastAlarm.Status)
	}

	lastIssue, err = lib.GetLastRedmineIssue(pluginName, moduleName)
	if err != nil {
		t.Errorf("Failed to get last Redmine issue: %v", err)
	}

	if lastIssue.Status != up {
		t.Errorf("Expected last issue status to be 'up', got '%s'", lastIssue.Status)
	}
}
