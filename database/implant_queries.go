package database

import (
	"awesomeProject/config"
	"awesomeProject/models"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func GetImplantsByUserID(userID int) ([]models.Implant, error) {
	var implants []models.Implant
	result := config.DB.Omit("ID").Where("user_id = ?", userID).Order("last_seen desc").Find(&implants) // Order by last_seen
	if result.Error != nil {
		return nil, result.Error
	}
	return implants, nil
}

func GetImplantByID(implantID string) (*models.Implant, error) {
	var implant models.Implant
	result := config.DB.Where("unique_token = ?", implantID).First(&implant)
	if result.Error != nil {
		return nil, result.Error
	}
	return &implant, nil
}

func GetPendingCommandsForImplant(implantID string) ([]models.Command, error) {
	var commands []models.Command
	result := config.DB.Where("implant_id = ? AND status = ?", implantID, "pending").Order("created_at asc").Find(&commands)
	if result.Error != nil {
		return nil, result.Error
	}
	return commands, nil
}

func GetCommandStatus(commandID int) (string, error) {
	var command models.Command
	result := config.DB.Select("status").Where("id = ?", commandID).First(&command)
	if result.Error != nil {
		return "", result.Error
	}
	return command.Status, nil
}

func MarkCommandAsExecuted(commandID int, output string) error {
	updateData := map[string]interface{}{
		"status": "executed",
		"output": output,
	}
	result := config.DB.Model(&models.Command{}).Where("id = ?", commandID).Updates(updateData)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func CreateImplant(userID int, targetOS string) (*models.Implant, error) {
	uniqueToken := uuid.New().String()
	implant := models.Implant{
		UserID:      userID,
		UniqueToken: uniqueToken,
		Status:      "new",
		TargetOS:    targetOS,
		Deployed:    false,
		LastSeen:    time.Now(),
		IPAddress:   "",
	}
	result := config.DB.Create(&implant)
	if result.Error != nil {
		return nil, result.Error
	}
	return &implant, nil
}

func GetCommandsByImplantID(implantID string) ([]models.Command, error) {
	var commands []models.Command
	result := config.DB.Where("implant_id = ?", implantID).Order("created_at asc").Find(&commands)
	if result.Error != nil {
		return nil, result.Error
	}
	return commands, nil
}

func GetImplantByToken(userID int, token string) (*models.Implant, error) {
	var imp models.Implant
	err := config.DB.
		Where("unique_token = ? AND user_id = ?", token, userID).
		First(&imp).Error
	if err != nil {
		return nil, err
	}
	return &imp, nil
}

func UpdateStatusForInactiveImplants(thresholdDuration time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-thresholdDuration)

	result := config.DB.Model(&models.Implant{}).
		Where("status = ? AND last_seen < ?", "online", cutoffTime).
		Update("status", "offline")

	if result.Error != nil {
		fmt.Printf("Error in UpdateStatusForInactiveImplants query: %v\n", result.Error)
		return 0, result.Error
	}
	return result.RowsAffected, nil
}
