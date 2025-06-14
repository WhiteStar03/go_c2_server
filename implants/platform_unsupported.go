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

		}
	}

	if setOSSpecificAttrs == nil {
		setOSSpecificAttrs = func(cmd *exec.Cmd) {

		}
	}
}
