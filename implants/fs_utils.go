
package main


import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

func listDirectory(path string) (FileSystemListing, error) {
	listing := FileSystemListing{RequestedPath: path}

	
	
	absPath, err := filepath.Abs(path)
	if err != nil {
		listing.Error = fmt.Sprintf("Could not get absolute path for '%s': %v", path, err)
		return listing, err
	}
	listing.RequestedPath = absPath 

	files, err := os.ReadDir(absPath)
	if err != nil {
		return listing, err 
	}

	for _, file := range files {
		info, err := file.Info() 
		entry := FileSystemEntry{
			Name:  file.Name(),
			IsDir: file.IsDir(),
			Path:  filepath.Join(absPath, file.Name()), 
		}
		if err == nil { 
			entry.Size = info.Size()
			entry.ModTime = info.ModTime()
			entry.Permissions = info.Mode().String()
		} else {
			
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
					ModTime:     time.Time{},  
					Permissions: "dr--r--r--", 
					Path:        path,
				})
			}
		}
	} else { 
		listing.Entries = append(listing.Entries, FileSystemEntry{
			Name:        "/",
			IsDir:       true,
			ModTime:     time.Time{},  
			Permissions: "drwxr-xr-x", 
			Path:        "/",
		})
	}
	if len(listing.Entries) == 0 {
		listing.Error = "No roots found or error determining roots."
		return listing, fmt.Errorf(listing.Error)
	}
	return listing, nil
}
