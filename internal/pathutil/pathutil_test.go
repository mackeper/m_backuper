package pathutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestIsNetworkPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		os       string
		expected bool
	}{
		// Windows UNC paths
		{
			name:     "Windows UNC with backslashes",
			path:     `\\server\share\backup`,
			os:       "windows",
			expected: true,
		},
		{
			name:     "Windows UNC with forward slashes",
			path:     `//server/share/backup`,
			os:       "windows",
			expected: true,
		},
		{
			name:     "Windows local path",
			path:     `C:\Users\test\backup`,
			os:       "windows",
			expected: false,
		},
		// Linux mount paths
		{
			name:     "Linux /mnt path",
			path:     "/mnt/smb/backup",
			os:       "linux",
			expected: true,
		},
		{
			name:     "Linux /media path",
			path:     "/media/user/share",
			os:       "linux",
			expected: true,
		},
		{
			name:     "Linux local path",
			path:     "/home/user/backup",
			os:       "linux",
			expected: false,
		},
		// macOS paths
		{
			name:     "macOS /Volumes path",
			path:     "/Volumes/backup",
			os:       "darwin",
			expected: true,
		},
		{
			name:     "macOS local path",
			path:     "/Users/test/backup",
			os:       "darwin",
			expected: false,
		},
		// Android/Termux paths
		{
			name:     "Android /mnt path",
			path:     "/mnt/remote/backup",
			os:       "linux",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Only run OS-specific tests on matching OS
			if tt.os != "" && runtime.GOOS != tt.os {
				t.Skipf("Skipping %s test on %s", tt.os, runtime.GOOS)
			}

			result := IsNetworkPath(tt.path)
			if result != tt.expected {
				t.Errorf("IsNetworkPath(%q) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		os       string
		expected string
	}{
		{
			name:     "Windows UNC path normalization",
			path:     "//server/share/backup",
			os:       "windows",
			expected: `\\server\share\backup`,
		},
		{
			name:     "Windows path with forward slashes",
			path:     "C:/Users/test/backup",
			os:       "windows",
			expected: `C:\Users\test\backup`,
		},
		{
			name:     "Linux path normalization",
			path:     "/mnt/smb//backup///data",
			os:       "linux",
			expected: "/mnt/smb/backup/data",
		},
		{
			name:     "macOS path normalization",
			path:     "/Volumes/backup/../backup/data",
			os:       "darwin",
			expected: "/Volumes/backup/data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.os != "" && runtime.GOOS != tt.os {
				t.Skipf("Skipping %s test on %s", tt.os, runtime.GOOS)
			}

			result := NormalizePath(tt.path)
			if result != tt.expected {
				t.Errorf("NormalizePath(%q) = %q, expected %q", tt.path, result, tt.expected)
			}
		})
	}
}

//nolint:gocyclo // Test function needs multiple subtests
func TestValidatePath(t *testing.T) {
	t.Run("valid writable directory", func(t *testing.T) {
		dir := t.TempDir()

		err := ValidatePath(dir)
		if err != nil {
			t.Errorf("ValidatePath(%q) returned error for valid directory: %v", dir, err)
		}

		// Verify test file was cleaned up
		testFile := filepath.Join(dir, ".m_backuper_write_test")
		if _, err := os.Stat(testFile); !os.IsNotExist(err) {
			t.Error("Test file was not cleaned up")
		}
	})

	t.Run("nonexistent path", func(t *testing.T) {
		err := ValidatePath("/nonexistent/path/that/does/not/exist")
		if err == nil {
			t.Error("ValidatePath should return error for nonexistent path")
		}
		if err != nil && err.Error() == "" {
			t.Error("Error message should not be empty")
		}
	})

	t.Run("nonexistent network path", func(t *testing.T) {
		// Use platform-appropriate network path
		var networkPath string
		if runtime.GOOS == "windows" {
			networkPath = `\\nonexistent\share`
		} else {
			networkPath = "/mnt/nonexistent/share"
		}

		err := ValidatePath(networkPath)
		if err == nil {
			t.Error("ValidatePath should return error for nonexistent network path")
		}
		// Should mention network drive in error message
		if err != nil && !contains(err.Error(), "network") && !contains(err.Error(), "mounted") {
			t.Logf("Warning: Error message doesn't mention network/mount: %v", err)
		}
	})

	t.Run("file instead of directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		file := filepath.Join(tmpDir, "file.txt")
		if err := os.WriteFile(file, []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}

		err := ValidatePath(file)
		if err == nil {
			t.Error("ValidatePath should return error for file instead of directory")
		}
		if err != nil && !contains(err.Error(), "not a directory") {
			t.Errorf("Error should mention 'not a directory', got: %v", err)
		}
	})

	t.Run("read-only directory", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping read-only test on Windows (requires different permissions model)")
		}

		tmpDir := t.TempDir()
		if err := os.Chmod(tmpDir, 0o444); err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = os.Chmod(tmpDir, 0o755) // Restore for cleanup
		}()

		err := ValidatePath(tmpDir)
		if err == nil {
			t.Error("ValidatePath should return error for read-only directory")
		}
		if err != nil && !contains(err.Error(), "writable") {
			t.Errorf("Error should mention 'writable', got: %v", err)
		}
	})
}

func TestGetPathType(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		os           string
		expectedType string
	}{
		{
			name:         "Windows UNC path",
			path:         `\\server\share`,
			os:           "windows",
			expectedType: "Windows UNC network path",
		},
		{
			name:         "Windows local path",
			path:         `C:\Users\test`,
			os:           "windows",
			expectedType: "local path",
		},
		{
			name:         "Linux network mount",
			path:         "/mnt/smb/backup",
			os:           "linux",
			expectedType: "network mount point",
		},
		{
			name:         "Linux local path",
			path:         "/home/user/backup",
			os:           "linux",
			expectedType: "local path",
		},
		{
			name:         "macOS network volume",
			path:         "/Volumes/backup",
			os:           "darwin",
			expectedType: "network mount point",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.os != "" && runtime.GOOS != tt.os {
				t.Skipf("Skipping %s test on %s", tt.os, runtime.GOOS)
			}

			result := GetPathType(tt.path)
			if result != tt.expectedType {
				t.Errorf("GetPathType(%q) = %q, expected %q", tt.path, result, tt.expectedType)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return s != "" && substr != "" && (s == substr || len(s) >= len(substr) && containsIgnoreCase(s, substr))
}

func containsIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
