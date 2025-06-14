package models

import "time"

type ScreenshotInfo struct {
	CommandID int       `json:"command_id"`
	Timestamp time.Time `json:"timestamp"`
	URLPath   string    `json:"url_path"`
	Filename  string    `json:"filename"`
}
