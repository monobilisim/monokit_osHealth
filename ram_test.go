//go:build osHealth

package main

import (
	"runtime"
	"runtime/debug"
	"testing"
	"time"

	lib "github.com/monobilisim/monokit2/lib"
	"github.com/shirou/gopsutil/v4/mem"
)

func fillRam(t *testing.T, targetPercent float64) func() {
	var chunks [][]byte
	chunkSize := 100 * 1024 * 1024 // 100MB

	for {
		v, err := mem.VirtualMemory()
		if err != nil {
			t.Logf("Failed to get memory stats: %v", err)
			break
		}

		usedPercent := float64(v.Total-v.Available) / float64(v.Total) * 100
		if usedPercent >= targetPercent {
			break
		}

		// Allocate
		newChunk := make([]byte, chunkSize)
		// Touch pages
		for i := 0; i < len(newChunk); i += 4096 {
			newChunk[i] = 1
		}
		chunks = append(chunks, newChunk)
	}

	return func() {
		chunks = nil
		runtime.GC()
		debug.FreeOSMemory()
	}
}

func TestCheckSystemRAM(t *testing.T) {
	lib.InitConfig(configFiles...)
	lib.InitializeDatabase()

	moduleName := "memory"

	cleanup := fillRam(t, 92.0) // Fill RAM to 92%

	t.Log("Running CheckSystemRAM with 90% ram usage")

	CheckSystemRAM(lib.Logger)

	// Verify DOWN alarm was created
	alarm, err := lib.GetLastZulipAlarm(pluginName, moduleName)
	if err != nil {
		t.Errorf("Failed to get last alarm: %v", err)
	}

	if alarm.Status != down {
		t.Errorf("Expected alarm status '%s', got '%s'. Content: %s", down, alarm.Status, alarm.Content)
	}

	cleanup() // Free up RAM

	// Wait for RAM to drop below limit
	limit := float64(lib.OsHealthConfig.RamUsageAlarm.Limit)
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

WaitForRam:
	for {
		select {
		case <-timeout:
			t.Log("Timeout waiting for RAM to drop")
			break WaitForRam
		case <-ticker.C:
			v, err := mem.VirtualMemory()
			if err == nil {
				used := float64(v.Total-v.Available) / float64(v.Total) * 100
				if used < limit {
					break WaitForRam
				}
			}
		}
	}

	CheckSystemRAM(lib.Logger)

	// Verify UP alarm was created
	alarm, err = lib.GetLastZulipAlarm(pluginName, moduleName)
	if err != nil {
		t.Errorf("Failed to get last alarm: %v", err)
	}

	if alarm.Status != up {
		t.Errorf("Expected alarm status '%s', got '%s'. Content: %s", up, alarm.Status, alarm.Content)
	}
}
