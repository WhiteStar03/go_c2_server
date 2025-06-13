// In implant/main.go or a new fs_utils.go
package main

// (Define FileSystemEntry and FileSystemListing structs here as well for the implant's use)
import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

func listDirectory(path string) (FileSystemListing, error) {
	listing := FileSystemListing{RequestedPath: path}

	// Ensure path is absolute and clean. This is important for consistency
	// and helps prevent some path traversal if not careful (though primary defense is on C2)
	absPath, err := filepath.Abs(path)
	if err != nil {
		listing.Error = fmt.Sprintf("Could not get absolute path for '%s': %v", path, err)
		return listing, err
	}
	listing.RequestedPath = absPath // Store the cleaned, absolute path

	files, err := os.ReadDir(absPath)
	if err != nil {
		return listing, err // Error will be set by caller based on this
	}

	for _, file := range files {
		info, err := file.Info() // os.DirEntry.Info() gets FileInfo
		entry := FileSystemEntry{
			Name:  file.Name(),
			IsDir: file.IsDir(),
			Path:  filepath.Join(absPath, file.Name()), // Send full path for convenience
		}
		if err == nil { // If .Info() succeeded
			entry.Size = info.Size()
			entry.ModTime = info.ModTime()
			entry.Permissions = info.Mode().String()
		} else {
			// Could log this error, or set a placeholder for permissions
			entry.Permissions = "?????????"
		}
		listing.Entries = append(listing.Entries, entry)
	}
	return listing, nil
}

func listRoots() (FileSystemListing, error) {
	listing := FileSystemListing{RequestedPath: "__ROOTS__"}
	if runtime.GOOS == "windows" {
		for drive := 'A'; drive <= 'Z'; drive++ {
			path := string(drive) + ":\\"
			_, err := os.Stat(path)
			if err == nil {
				listing.Entries = append(listing.Entries, FileSystemEntry{
					Name:        path,
					IsDir:       true,
					ModTime:     time.Time{},  // Not easily available for drive itself
					Permissions: "dr--r--r--", // Placeholder
					Path:        path,
				})
			}
		}
	} else { // Linux, macOS, etc.
		listing.Entries = append(listing.Entries, FileSystemEntry{
			Name:        "/",
			IsDir:       true,
			ModTime:     time.Time{},  // Stat "/" for actual modtime if needed
			Permissions: "drwxr-xr-x", // Placeholder
			Path:        "/",
		})
	}
	if len(listing.Entries) == 0 {
		listing.Error = "No roots found or error determining roots."
		return listing, fmt.Errorf(listing.Error)
	}
	return listing, nil
}
