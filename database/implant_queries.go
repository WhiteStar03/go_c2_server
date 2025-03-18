package database

import (
	"awesomeProject/config"
	"awesomeProject/models" // Replace with your actual models package
	"fmt"
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
	result := config.DB.Where("implant_id = ?", implantID).First(&implant)
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

// MarkCommandAsExecuted updates the status of a command to "executed".
func MarkCommandAsExecuted(commandID int) error {
	result := config.DB.Model(&models.Command{}).Where("id = ?", commandID).Update("status", "executed")
	if result.Error != nil {
		return result.Error
	}

	return nil
}
