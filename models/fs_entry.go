// Package models awesomeProject/models/fs_entry.go
package models

import "time"

type FileSystemEntry struct {
	Name        string    `json:"name"`
	IsDir       bool      `json:"is_dir"`
	Size        int64     `json:"size"`
	ModTime     time.Time `json:"mod_time"`
	Permissions string    `json:"permissions"`
	Path        string    `json:"path"`
}

type FileSystemListing struct {
	RequestedPath string            `json:"requested_path"`
	Entries       []FileSystemEntry `json:"entries"`
	Error         string            `json:"error,omitempty"`
}
