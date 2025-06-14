package models

import "time"

type Command struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	ImplantID string    `json:"implant_id" gorm:"size:255"`
	Command   string    `json:"command" gorm:"size:1024"`
	Status    string    `json:"status" gorm:"size:50"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	Output    string    `json:"output" gorm:"type:text"`
}
