//go:build !linux && !windows

package main

import (
	"fmt"
	"os/exec"
)

func init() {

	if takeScreenshot == nil {
		takeScreenshot = func() (string, error) {
			return "", fmt.Errorf("screenshot functionality is not supported on this platform")
		}
	}

	if doSelfDelete == nil {
		doSelfDelete = func(selfExePath string, originalLauncherPath string) {
			// No-op or log: Self-delete not implemented for this platform
		}
	}

	if setOSSpecificAttrs == nil {
		setOSSpecificAttrs = func(cmd *exec.Cmd) {
			// No-op: No specific OS attributes to set for this platform by default
		}
	}
}
