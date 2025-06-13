// models/screenshot_info.go
package models

import "time"

type ScreenshotInfo struct {
	CommandID int       `json:"command_id"`
	Timestamp time.Time `json:"timestamp"` // from command's updated_at
	URLPath   string    `json:"url_path"`  // Relative path as stored (e.g., "c2_screenshots/implant-id/filename.png")
	Filename  string    `json:"filename"`  // Just the filename (e.g., "screenshot_cmd62_timestamp.png")
}
