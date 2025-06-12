package models

import "time"

type Command struct {
	ID        int       `json:"id" gorm:"primaryKey"`             // Unique ID for the command
	ImplantID string    `json:"implant_id" gorm:"size:255"`       // Foreign key to the implant
	Command   string    `json:"command" gorm:"size:1024"`         // The command to execute
	Status    string    `json:"status" gorm:"size:50"`            // Status of the command (e.g., "pending", "executed")
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"` // Timestamp created
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"` // Timestamp updated
	Output    string    `json:"output" gorm:"type:text"`          // Output 
}
