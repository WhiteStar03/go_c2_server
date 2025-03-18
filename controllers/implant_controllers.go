package controllers

import (
	"awesomeProject/config"
	"awesomeProject/models"
	"net/http"

	"awesomeProject/database"
	"github.com/gin-gonic/gin"
)

// GetUserImplants returns all implants owned by the current user.
func GetUserImplants(c *gin.Context) {
	// Get the current user ID from the context (set by your AuthMiddleware)
	userID := c.MustGet("user_id").(int)

	// Query the database for implants owned by the user
	implants, err := database.GetImplantsByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch implants"})
		return
	}

	// Return the list of implants
	c.JSON(http.StatusOK, gin.H{
		"implants": implants,
	})
}

// SendCommand sends a command to a specific implant, ensuring it is owned by the current user.
func SendCommand(c *gin.Context) {
	// Get the current user ID from the context
	userID := c.MustGet("user_id").(int)

	// Extract the implant ID and command from the request body
	var request struct {
		ImplantID string `json:"implant_id"`
		Command   string `json:"command"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return // Stop execution after sending the error response
	}

	// Verify that the implant is owned by the current user
	implant, err := database.GetImplantByID(request.ImplantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch implant"})
		return // Stop execution after sending the error response
	}

	if implant.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not own this implant"})
		return // Stop execution after sending the error response
	}

	// Create a new command
	newCommand := models.Command{
		ImplantID: request.ImplantID,
		Command:   request.Command,
		Status:    "pending", // Default status
	}

	// Insert the new command into the database
	result := config.DB.Create(&newCommand)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send command"})
		return // Stop execution after sending the error response
	}

	// Send the command to the implant (you would implement this logic)
	// For now, we'll just log the command and return a success response.
	c.JSON(http.StatusOK, gin.H{
		"message":    "Command sent successfully",
		"implant_id": request.ImplantID,
		"command":    request.Command,
	})
	// No need for a return statement here, as this is the end of the function
}

func GetCommandsForImplant(c *gin.Context) {
	// Extract the implant ID from the request (e.g., via query parameter or header)
	implantID := c.Query("implant_id")
	if implantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Implant ID is required"})
		return
	}

	// Fetch pending commands for the implant
	commands, err := database.GetPendingCommandsForImplant(implantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch commands"})
		return
	}

	// Mark the commands as "executed" (optional, depending on your workflow)
	for _, command := range commands {
		err := database.MarkCommandAsExecuted(command.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update command status"})
			return
		}
	}

	// Return the commands to the implant
	c.JSON(http.StatusOK, gin.H{
		"commands": commands,
	})
}
