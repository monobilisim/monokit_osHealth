//go:build osHealth

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	lib "github.com/monobilisim/monokit2/lib"
	"github.com/shirou/gopsutil/v4/disk"
)

func setupTestDisk(t *testing.T, sizeMB int) (string, func()) {
	// Strategy 1: Try Loopback
	loopMountPoint, loopCleanup := tryCreateLoopbackDisk(t, sizeMB)
	if loopMountPoint != "" {
		return loopMountPoint, loopCleanup
	}

	// Strategy 2: Fallback to /dev/shm (tmpfs)
	// This allows running in unprivileged containers (Docker) where mount is forbidden.
	t.Log("Loopback creation failed/skipped. Attempting fallback to /dev/shm (tmpfs)...")

	shmPath := "/dev/shm"
	usage, err := disk.Usage(shmPath)
	if err != nil {
		t.Skipf("Skipping disk test: could not get usage of %s: %v", shmPath, err)
		return "", func() {}
	}

	// Safety check: Don't try to fill a huge /dev/shm (e.g. host RAM size in some configs)
	// Limit to 2GB.
	const maxSafeSize = 2 * 1024 * 1024 * 1024
	if usage.Total > maxSafeSize {
		t.Skipf("Skipping disk test: /dev/shm is too large (%v bytes) to safely fill for testing", usage.Total)
		return "", func() {}
	}

	// We need to enable "tmpfs" support in the main application temporarily
	originalSupported := make([]string, len(supportedFilesystems))
	copy(originalSupported, supportedFilesystems)

	// Add tmpfs if not present
	hasTmpfs := false
	for _, fs := range supportedFilesystems {
		if fs == "tmpfs" {
			hasTmpfs = true
			break
		}
	}
	if !hasTmpfs {
		supportedFilesystems = append(supportedFilesystems, "tmpfs")
	}

	// Create a directory to hold our fill file, ensuring we don't mess up other things
	testDir := filepath.Join(shmPath, fmt.Sprintf("monokit-test-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(testDir, 0755); err != nil {
		supportedFilesystems = originalSupported // Restore
		t.Skipf("Skipping disk test: could not create test dir in %s: %v", shmPath, err)
		return "", func() {}
	}

	return shmPath, func() {
		os.RemoveAll(testDir)
		supportedFilesystems = originalSupported // Restore
	}
}

func tryCreateLoopbackDisk(t *testing.T, sizeMB int) (string, func()) {
	if _, err := exec.LookPath("mkfs.ext4"); err != nil {
		return "", nil
	}
	if _, err := exec.LookPath("mount"); err != nil {
		return "", nil
	}

	tmpDir, err := os.MkdirTemp("", "monokit-disk-test-")
	if err != nil {
		t.Logf("Failed to create temp dir: %v", err)
		return "", nil
	}

	imagePath := filepath.Join(tmpDir, "disk.img")
	mountPoint := filepath.Join(tmpDir, "mnt")

	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		os.RemoveAll(tmpDir)
		return "", nil
	}

	if err := exec.Command("truncate", "-s", fmt.Sprintf("%dM", sizeMB), imagePath).Run(); err != nil {
		os.RemoveAll(tmpDir)
		return "", nil
	}

	if out, err := exec.Command("mkfs.ext4", "-F", imagePath).CombinedOutput(); err != nil {
		t.Logf("mkfs.ext4 output: %s", string(out))
		os.RemoveAll(tmpDir)
		return "", nil
	}

	cmd := exec.Command("mount", "-o", "loop", imagePath, mountPoint)
	if out, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(tmpDir)
		// We expect this to fail in unprivileged containers, so we don't Errorf here.
		// Just log and return empty to signal "try fallback".
		t.Logf("Loopback mount failed (expected in docker): %v. Output: %s", err, string(out))
		return "", nil
	}

	time.Sleep(1 * time.Second)

	return mountPoint, func() {
		exec.Command("umount", mountPoint).Run()
		os.RemoveAll(tmpDir)
	}
}

func TestCheckSystemDisk(t *testing.T) {
	// Add sbin to path testing runs in Debian
	currentPath := os.Getenv("PATH")
	newPath := fmt.Sprintf("/usr/sbin:%s", currentPath)
	os.Setenv("PATH", newPath)

	lib.InitConfig(configFiles...)
	lib.InitializeDatabase()

	moduleName := "disk"

	// Setup disk (loopback or fallback to /dev/shm)
	mountPoint, cleanup := setupTestDisk(t, 100)
	if mountPoint == "" {
		// cleanup might be nil if completely skipped
		return
	}
	defer cleanup()

	t.Logf("Using test disk at %s", mountPoint)

	// Verify it shows up in gopsutil
	parts, err := disk.Partitions(true)
	if err != nil {
		t.Errorf("Failed to get partitions: %v", err)
	}
	found := false
	for _, p := range parts {
		if p.Mountpoint == mountPoint {
			found = true
			break
		}
	}
	if !found {
		t.Logf("Warning: gopsutil did not see the new mount point %s immediately. Continuing anyway as CheckSystemDisk might see it.", mountPoint)
	}

	// Current state should be OK
	t.Log("Running initial CheckSystemDisk (expecting no alarm)")
	CheckSystemDisk(lib.Logger)

	// Fill the disk to 92%
	usage, err := disk.Usage(mountPoint)
	if err != nil {
		t.Errorf("Failed to get usage of test disk: %v", err)
		return
	}

	total := usage.Total
	target := uint64(float64(total) * 0.92)

	// Determine where to write the fill file
	var fillPath string
	if mountPoint == "/dev/shm" {
		fillPath = filepath.Join(mountPoint, fmt.Sprintf("monokit-fill-%d.dat", time.Now().UnixNano()))
	} else {
		fillPath = filepath.Join(mountPoint, "fill.dat")
	}

	f, err := os.Create(fillPath)
	if err != nil {
		t.Errorf("Failed to create fill file: %v", err)
		return
	}

	// Ensure we cleanup the specific file, in case cleanup() doesn't cover it (e.g. for /dev/shm)
	defer os.Remove(fillPath)

	// Write in chunks
	chunkSize := 1024 * 1024 // 1MB
	buf := make([]byte, chunkSize)

	toWrite := int64(target - usage.Used)
	// Add a bit more to be safe (e.g. +1MB) to strictly exceed
	toWrite += 1024 * 1024

	// Safety check again: Don't write negative amount
	if toWrite < 0 {
		toWrite = 1024 * 1024
	}

	t.Logf("Filling disk: Total %d, Current Used %d, Writing %d", total, usage.Used, toWrite)

	for toWrite > 0 {
		w := int64(chunkSize)
		if toWrite < w {
			w = toWrite
		}
		if _, err := f.Write(buf[:w]); err != nil {
			f.Close()
			t.Errorf("Failed to write to fill file: %v", err)
			return
		}
		toWrite -= w
	}
	f.Close()

	// Force sync
	exec.Command("sync").Run()

	t.Log("Running CheckSystemDisk with >90% usage")
	CheckSystemDisk(lib.Logger)

	// Verify DOWN alarm
	alarm, err := lib.GetLastZulipAlarm(pluginName, moduleName)
	if err != nil {
		t.Errorf("Failed to get last alarm: %v", err)
	}
	if alarm.Status != down {
		t.Errorf("Expected alarm status '%s', got '%s'. Content: %s", down, alarm.Status, alarm.Content)
	}

	// Verify Redmine Issue
	issue, err := lib.GetLastRedmineIssue(pluginName, moduleName)
	if err != nil {
		t.Errorf("Failed to get last issue: %v", err)
	}
	if issue.Status != down {
		t.Errorf("Expected issue status '%s', got '%s'", down, issue.Status)
	}

	// Cleanup to recover
	if err := os.Remove(fillPath); err != nil {
		t.Errorf("Failed to remove fill file: %v", err)
	}
	exec.Command("sync").Run()

	t.Log("Running CheckSystemDisk after cleanup")
	CheckSystemDisk(lib.Logger)

	// Verify UP alarm
	alarm, err = lib.GetLastZulipAlarm(pluginName, moduleName)
	if err != nil {
		t.Errorf("Failed to get last alarm: %v", err)
	}
	if alarm.Status != up {
		t.Errorf("Expected alarm status '%s', got '%s'. Content: %s", up, alarm.Status, alarm.Content)
	}

	// Verify Redmine Issue Closed
	issue, err = lib.GetLastRedmineIssue(pluginName, moduleName)
	if err != nil {
		t.Errorf("Failed to get last issue: %v", err)
	}
	if issue.Status != up {
		t.Errorf("Expected issue status '%s', got '%s'", up, issue.Status)
	}
}
