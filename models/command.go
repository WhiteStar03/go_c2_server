package models

import "time"

// Command represents a command sent to an implant.
type Command struct {
	ID        int       `json:"id" gorm:"primaryKey"`             // Unique ID for the command
	ImplantID string    `json:"implant_id" gorm:"size:255"`       // Foreign key to the implant
	Command   string    `json:"command" gorm:"size:1024"`         // The command to execute
	Status    string    `json:"status" gorm:"size:50"`            // Status of the command (e.g., "pending", "executed")
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"` // Timestamp when the command was created
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"` // Timestamp when the command was last updated
}
