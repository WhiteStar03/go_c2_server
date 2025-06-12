// models/screenshot_info.go
package models

import "time"

// ScreenshotInfo represents data for a single screenshot.
type ScreenshotInfo struct {
	CommandID int       `json:"command_id"`
	Timestamp time.Time `json:"timestamp"` 
	URLPath   string    `json:"url_path"`  
	Filename  string    `json:"filename"`  
}
