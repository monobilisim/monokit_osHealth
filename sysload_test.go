//go:build osHealth

package main

import (
	"runtime"
	"testing"
	"time"

	lib "github.com/monobilisim/monokit2/lib"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/load"
)

func generateLoad(t *testing.T) func() {
	numCPU := runtime.NumCPU()
	done := make(chan struct{})

	// Spawn 2x the number of CPUs to ensure we generate significant load
	// and increase the run queue length.
	for i := 0; i < numCPU*2; i++ {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					// Busy loop to consume CPU
					_ = 0 + 0
				}
			}
		}()
	}

	return func() {
		close(done)
	}
}

func TestCheckSystemLoad(t *testing.T) {
	lib.InitConfig(configFiles...)
	lib.InitializeDatabase()

	moduleName := "sysload"

	// Get CPU count and calculate limit
	cpuCores, _ := cpu.Counts(false)
	limitMultiplier := lib.OsHealthConfig.SystemLoadAlarm.LimitMultiplier
	limit := limitMultiplier * float64(cpuCores)

	t.Logf("System Load Limit: %.2f (Multiplier: %.2f, Cores: %d)", limit, limitMultiplier, cpuCores)

	// Start generating load
	cleanup := generateLoad(t)
	t.Log("Started generating CPU load...")

	// Wait for load to rise above limit
	// Load average is a moving average, so it takes time to rise.
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(2 * time.Second)

	loadHigh := false

WaitForLoadHigh:
	for {
		select {
		case <-timeout:
			t.Log("Timeout waiting for system load to rise")
			break WaitForLoadHigh
		case <-ticker.C:
			avg, err := load.Avg()
			if err != nil {
				continue
			}
			t.Logf("Current Load1: %.2f (Target: > %.2f)", avg.Load1, limit)

			if avg.Load1 >= limit {
				loadHigh = true
				break WaitForLoadHigh
			}
		}
	}
	ticker.Stop()

	if !loadHigh {
		cleanup()
		t.Skip("Skipping test: Could not generate enough system load within timeout. This is expected in some environments.")
	}

	// Trigger the check
	CheckSystemLoad(lib.Logger)

	// Verify DOWN alarm was created
	alarm, err := lib.GetLastZulipAlarm(pluginName, moduleName)
	if err != nil {
		cleanup()
		t.Errorf("Failed to get last alarm: %v", err)
	}

	if alarm.Status != down {
		t.Errorf("Expected alarm status '%s', got '%s'. Content: %s", down, alarm.Status, alarm.Content)
	}

	// Verify DOWN Redmine issue was created
	issue, err := lib.GetLastRedmineIssue(pluginName, moduleName)
	if err != nil {
		cleanup()
		t.Errorf("Failed to get last Redmine issue: %v", err)
	}

	if issue.Status != down {
		t.Errorf("Expected Redmine issue status '%s', got '%s'", down, issue.Status)
	}

	// Stop generating load
	cleanup()
	t.Log("Stopped generating load. Waiting for load to drop...")

	// Wait for load to drop below limit
	// This can take a while as load average decays exponentially.
	// We might increase the limit temporarily to speed up the test pass condition
	// if the natural decay is too slow, but let's try waiting first.
	timeout = time.After(1 * time.Minute)
	ticker = time.NewTicker(2 * time.Second)

	loadLow := false

WaitForLoadLow:
	for {
		select {
		case <-timeout:
			t.Log("Timeout waiting for system load to drop")
			break WaitForLoadLow
		case <-ticker.C:
			avg, err := load.Avg()
			if err != nil {
				continue
			}
			t.Logf("Current Load1: %.2f (Target: < %.2f)", avg.Load1, limit)

			if avg.Load1 < limit {
				loadLow = true
				break WaitForLoadLow
			}
		}
	}
	ticker.Stop()

	if !loadLow {
		// If natural decay is too slow, we can force the test to pass by raising the limit
		// just for the check, OR fail.
		// If it times out, we'll try to run the check anyway and see if it fails.
		t.Log("Load didn't drop fast enough, proceeding to check anyway (might fail)")
	}

	CheckSystemLoad(lib.Logger)

	// Verify UP alarm was created
	alarm, err = lib.GetLastZulipAlarm(pluginName, moduleName)
	if err != nil {
		t.Errorf("Failed to get last alarm: %v", err)
	}

	if alarm.Status != up {
		t.Errorf("Expected alarm status '%s', got '%s'. Content: %s", up, alarm.Status, alarm.Content)
	}

	// Verify UP Redmine issue was created (or updated)
	issue, err = lib.GetLastRedmineIssue(pluginName, moduleName)
	if err != nil {
		t.Errorf("Failed to get last Redmine issue: %v", err)
	}

	if issue.Status != up {
		t.Errorf("Expected Redmine issue status '%s', got '%s'", up, issue.Status)
	}
}
