package models

import "time"

type Implant struct {
	ID          int       `json:"id" gorm:"primaryKey"`
	UserID      int       `json:"user_id"`
	UniqueToken string    `json:"unique_token" gorm:"unique;not null"`
	Status      string    `json:"status"`
	TargetOS    string    `json:"target_os" gorm:"default:'unknown'"`
	LastSeen    time.Time `json:"last_seen"`
	IPAddress   string    `json:"ip_address"`
	Deployed    bool      `json:"deployed" gorm:"default:false"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
