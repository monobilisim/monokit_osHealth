//go:build osHealth && linux

package main

import (
	"os"
	"os/exec"
	"testing"
	"time"

	lib "github.com/monobilisim/monokit2/lib"
)

func TestCheckSystemInit(t *testing.T) {
	lib.InitConfig(configFiles...)
	lib.InitializeDatabase()

	// test service unit name used in Redmine
	serviceModule := "postgresql-unit"

	// dummy service content
	serviceContent := `[Unit]
Description=PostgreSQL Test Service

[Service]
ExecStart=/bin/sleep 3600
Type=simple

[Install]
WantedBy=multi-user.target
`

	t.Log("Creating dummy postgresql service")

	err := os.WriteFile("/etc/systemd/system/postgresql.service", []byte(serviceContent), 0644)
	if err != nil {
		t.Errorf("Failed to create service file: %v", err)
	}

	err = exec.Command("systemctl", "daemon-reload").Run()
	if err != nil {
		t.Errorf("Failed to reload systemd daemon: %v", err)
	}

	err = exec.Command("systemctl", "enable", "--now", "postgresql.service").Run()
	if err != nil {
		t.Errorf("Failed to start postgresql service: %v", err)
	}

	t.Log("Run CheckSystemInit for first time to save service state")
	// run it for saving postgresql.service as active
	CheckSystemInit(lib.Logger)

	err = exec.Command("systemctl", "stop", "postgresql.service").Run()
	if err != nil {
		t.Errorf("Failed to stop postgresql service: %v", err)
	}

	time.Sleep(3 * time.Second)

	t.Log("Run CheckSystemInit after postgresql.service is stopped")

	CheckSystemInit(lib.Logger)

	issue, err := lib.GetLastRedmineIssue(pluginName, serviceModule)
	if err != nil {
		t.Errorf("Could not get last Redmine issue from %s", serviceModule)
	}

	if issue.Status != down {
		t.Errorf("Expected Redmine issue status '%s', got '%s'", down, issue.Status)
	}

	err = exec.Command("systemctl", "start", "postgresql.service").Run()
	if err != nil {
		t.Errorf("Failed to start postgresql service: %v", err)
	}

	t.Log("Run CheckSystemInit after postgresql.service is started")

	CheckSystemInit(lib.Logger)

	issue, err = lib.GetLastRedmineIssue(pluginName, serviceModule)
	if err != nil {
		t.Errorf("Could not get last Redmine issue from %s", serviceModule)
	}

	if issue.Status != up {
		t.Errorf("Expected Redmine issue status '%s', got '%s'", up, issue.Status)
	}
}
