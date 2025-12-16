package pathutil

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// IsNetworkPath checks if the path appears to be a network path
func IsNetworkPath(path string) bool {
	// Windows UNC path: \\server\share or //server/share
	if runtime.GOOS == "windows" {
		return strings.HasPrefix(path, `\\`) || strings.HasPrefix(path, `//`)
	}

	// Unix-like: Check common mount points
	// Common network mount locations
	networkPrefixes := []string{
		"/mnt/",
		"/media/",
		"/Volumes/", // macOS
	}

	for _, prefix := range networkPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// NormalizePath converts path to OS-appropriate format
func NormalizePath(path string) string {
	// Convert forward slashes to backslashes on Windows
	if runtime.GOOS == "windows" {
		// Handle UNC paths starting with //
		if strings.HasPrefix(path, "//") {
			path = strings.Replace(path, "//", `\\`, 1)
		}
		path = filepath.FromSlash(path)
	}
	return filepath.Clean(path)
}

// ValidatePath checks if a path is accessible and writable
func ValidatePath(path string) error {
	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			if IsNetworkPath(path) {
				return fmt.Errorf("path does not exist (is network drive mounted?): %s", path)
			}
			return fmt.Errorf("path does not exist: %s", path)
		}
		return fmt.Errorf("cannot access path: %w", err)
	}

	// Check if it's a directory
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}

	// Test write access by creating a temporary file
	testFile := filepath.Join(path, ".m_backuper_write_test")
	if err := os.WriteFile(testFile, []byte("test"), 0o600); err != nil {
		return fmt.Errorf("path is not writable: %w", err)
	}
	_ = os.Remove(testFile)

	return nil
}

// GetPathType returns a description of the path type for logging
func GetPathType(path string) string {
	if IsNetworkPath(path) {
		if runtime.GOOS == "windows" {
			return "Windows UNC network path"
		}
		return "network mount point"
	}
	return "local path"
}
