package models

import "time"

type Implant struct {
	ID          int       `json:"id" gorm:"primaryKey"`
	UserID      int       `json:"user_id"`                             // Foreign key to the user
	UniqueToken string    `json:"unique_token" gorm:"unique;not null"` // Unique token for the implant, used for authentication
	Status      string    `json:"status"`                              // Status of the implant (e.g., "online", "offline")
	TargetOS    string    `json:"target_os" gorm:"default:'unknown'"`  // Added: "windows", "linux", or "unknown"
	LastSeen    time.Time `json:"last_seen"`                           // Timestamp of the last activity
	IPAddress   string    `json:"ip_address"`                          // IP address of the implant
	Deployed    bool      `json:"deployed" gorm:"default:false"`       // Whether the implant is deployed or not
	CreatedAt   time.Time `json:"created_at"`                          // Timestamp when the implant was created
	UpdatedAt   time.Time `json:"updated_at"`                          // Timestamp when the implant was last updated
}
