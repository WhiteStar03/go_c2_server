package controllers

import (
	"awesomeProject/config"
	"net/http"

	"awesomeProject/models"
	"github.com/gin-gonic/gin"
)

// GetCommandsByImplantID returns all commands for a specific implant.
// GetCommandsByImplantID returns all commands for a specific implant.
func GetCommandsByImplantID(implantID string) ([]models.Command, error) {
	var commands []models.Command

	// Fetch all commands for the implant
	result := config.DB.Where("implant_id = ?", implantID).Find(&commands)
	if result.Error != nil {
		return nil, result.Error
	}

	return commands, nil
}

// SendRealTimeCommand sends a command to an implant and returns the output.
func SendRealTimeCommand(c *gin.Context) {
	var request struct {
		ImplantID string `json:"implant_id"`
		Command   string `json:"command"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// Simulate command execution and output
	output := executeCommand(request.Command)

	// Save the command and its output in the database
	command := models.Command{
		ImplantID: request.ImplantID,
		Command:   request.Command,
		Status:    "executed",
		Output:    output,
	}

	result := config.DB.Create(&command)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save command"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Command executed successfully",
		"output":  output,
	})
}

// executeCommand simulates command execution and returns the output.
func executeCommand(command string) string {
	// Replace this with actual command execution logic
	switch command {
	case "ls":
		return "file1.txt\nfile2.txt\nfile3.txt"
	case "pwd":
		return "/home/user"
	default:
		return "Unknown command"
	}
}
