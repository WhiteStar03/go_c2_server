package database

import (
	"awesomeProject/config"
	"awesomeProject/models" // Replace with your actual models package
	"fmt"
	"github.com/google/uuid"
	"time"
)

// GetImplantsByUserID returns all implants owned by a specific user.
func GetImplantsByUserID(userID int) ([]models.Implant, error) {
	var implants []models.Implant

	// Query the database for implants owned by the user
	result := config.DB.Where("user_id = ?", userID).Find(&implants)
	if result.Error != nil {
		return nil, result.Error
	}

	return implants, nil
}

// GetImplantByID returns an implant by its ID.
func GetImplantByID(implantID string) (*models.Implant, error) {
	var implant models.Implant

	// Query the database for the implant
	result := config.DB.Where("unique_token = ?", implantID).First(&implant)
	if result.Error != nil {
		return nil, result.Error
	}

	return &implant, nil
}

// GetPendingCommandsForImplant returns all pending commands for the implant.
func GetPendingCommandsForImplant(implantID string) ([]models.Command, error) {
	var commands []models.Command

	result := config.DB.Where("implant_id = ? AND status = ?", implantID, "pending").Find(&commands)
	if result.Error != nil {
		return nil, result.Error
	}
	fmt.Println(commands)
	return commands, nil
}

// GetCommandStatus retrieves the status of a command by its ID
func GetCommandStatus(commandID int) (string, error) {
	var command models.Command
	result := config.DB.Where("id = ?", commandID).First(&command)
	if result.Error != nil {
		return "", result.Error
	}

	return command.Status, nil
}

// MarkCommandAsExecuted updates the status of a command to "executed".
// MarkCommandAsExecuted updates the status and output of a command.
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

func CreateImplant(userID int) (*models.Implant, error) {
	uniqueToken := uuid.New().String()

	implant := models.Implant{
		UserID:      userID,
		UniqueToken: uniqueToken,
		Status:      "new",
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
