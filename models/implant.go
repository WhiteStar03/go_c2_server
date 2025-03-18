package models

import "time"

type Implant struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	UserID    int       `json:"user_id"`    // Foreign key to the user
	ImplantID string    `json:"implant_id"` // Unique ID for the implant
	Status    string    `json:"status"`     // Status of the implant (e.g., "online", "offline")
	LastSeen  time.Time `json:"last_seen"`  // Timestamp of the last activity
	IPAddress string    `json:"ip_address"` // IP address of the implant
	CreatedAt time.Time `json:"created_at"` // Timestamp when the implant was created
	UpdatedAt time.Time `json:"updated_at"` // Timestamp when the implant was last updated
}
