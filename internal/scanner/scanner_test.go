package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanDirectory(t *testing.T) {
	// Create temporary test directory structure
	tmpDir := t.TempDir()

	// Create test files
	testFiles := []string{
		"file1.txt",
		"file2.go",
		"subdir/file3.txt",
		"subdir/file4.log",
	}

	for _, f := range testFiles {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	// Scan the directory
	scanner := New([]string{})
	files, err := scanner.Scan([]string{tmpDir})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// Should find all files
	if len(files) != len(testFiles) {
		t.Errorf("expected %d files, got %d", len(testFiles), len(files))
	}
}

func TestIgnorePatterns(t *testing.T) {
	// Create temporary test directory structure
	tmpDir := t.TempDir()

	// Create test files
	testFiles := map[string]bool{
		"file1.txt":       false, // should be included
		"file2.tmp":       true,  // should be ignored (*.tmp)
		"file3.log":       false, // should be included
		".cache/data":     true,  // should be ignored (.cache/*)
		"subdir/file.txt": false, // should be included
		"subdir/file.tmp": true,  // should be ignored (*.tmp)
	}

	for f := range testFiles {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	// Scan with ignore patterns
	scanner := New([]string{"*.tmp", ".cache/*"})
	files, err := scanner.Scan([]string{tmpDir})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// Count files that should be included
	expectedCount := 0
	for _, shouldIgnore := range testFiles {
		if !shouldIgnore {
			expectedCount++
		}
	}

	if len(files) != expectedCount {
		t.Errorf("expected %d files, got %d", expectedCount, len(files))
		for _, f := range files {
			t.Logf("found: %s", f.Path)
		}
	}

	// Verify no ignored files are in the result
	for _, file := range files {
		relPath, _ := filepath.Rel(tmpDir, file.Path)
		if shouldIgnore, exists := testFiles[relPath]; exists && shouldIgnore {
			t.Errorf("file %s should have been ignored", relPath)
		}
	}
}

func TestNestedDirectories(t *testing.T) {
	// Create temporary test directory structure
	tmpDir := t.TempDir()

	// Create nested directory structure
	nestedFiles := []string{
		"level1/file1.txt",
		"level1/level2/file2.txt",
		"level1/level2/level3/file3.txt",
	}

	for _, f := range nestedFiles {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	// Scan the directory
	scanner := New([]string{})
	files, err := scanner.Scan([]string{tmpDir})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// Should find all nested files
	if len(files) != len(nestedFiles) {
		t.Errorf("expected %d files, got %d", len(nestedFiles), len(files))
	}
}

func TestPermissionErrorHandling(t *testing.T) {
	// This test is harder to implement in a portable way
	// as it requires creating files/directories with no read permissions
	// For now, we'll skip it or implement it only on Unix-like systems
	if os.Getenv("SKIP_PERMISSION_TESTS") != "" {
		t.Skip("skipping permission test")
	}

	tmpDir := t.TempDir()
	restrictedDir := filepath.Join(tmpDir, "restricted")

	// Create a directory with no read permissions
	if err := os.Mkdir(restrictedDir, 0o000); err != nil {
		t.Fatalf("failed to create restricted directory: %v", err)
	}
	defer func() {
		_ = os.Chmod(restrictedDir, 0o755) // Restore permissions for cleanup
	}()

	// Scan should not fail, just log and continue
	scanner := New([]string{})
	files, err := scanner.Scan([]string{tmpDir})
	if err != nil {
		t.Fatalf("scan should not fail on permission errors: %v", err)
	}

	// Should not include the restricted directory
	for _, file := range files {
		if filepath.Base(file.Path) == "restricted" {
			t.Error("should not have scanned restricted directory")
		}
	}
}

func TestSymlinkHandling(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a regular file
	targetFile := filepath.Join(tmpDir, "target.txt")
	if err := os.WriteFile(targetFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create target file: %v", err)
	}

	// Create a symlink to the file
	symlinkPath := filepath.Join(tmpDir, "link.txt")
	if err := os.Symlink(targetFile, symlinkPath); err != nil {
		t.Skip("symlink creation not supported on this platform")
	}

	// Scan the directory
	scanner := New([]string{})
	files, err := scanner.Scan([]string{tmpDir})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// Should handle symlinks correctly (follow them but avoid loops)
	// The exact count depends on how we handle symlinks
	// We should at least find the target file
	if len(files) < 1 {
		t.Error("should have found at least one file")
	}
}

func TestDoubleStarPattern(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory structure with node_modules at different levels
	testFiles := map[string]bool{
		"file1.txt":                        false, // should be included
		"node_modules/package.json":        true,  // should be ignored
		"src/node_modules/lib.js":          true,  // should be ignored
		"src/app.js":                       false, // should be included
		"src/components/node_modules/x.js": true,  // should be ignored
		"src/components/Button.js":         false, // should be included
	}

	for f := range testFiles {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	// Scan with ** pattern
	scanner := New([]string{"**/node_modules/**"})
	files, err := scanner.Scan([]string{tmpDir})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// Count files that should be included
	expectedCount := 0
	for _, shouldIgnore := range testFiles {
		if !shouldIgnore {
			expectedCount++
		}
	}

	if len(files) != expectedCount {
		t.Errorf("expected %d files, got %d", expectedCount, len(files))
		for _, f := range files {
			t.Logf("found: %s", f.Path)
		}
	}

	// Verify no ignored files are in the result
	for _, file := range files {
		if containsPath(file.Path, "node_modules") {
			t.Errorf("file %s should have been ignored (contains node_modules)", file.Path)
		}
	}
}

func TestMatchPattern(t *testing.T) {
	scanner := New([]string{})

	tests := []struct {
		path    string
		pattern string
		want    bool
	}{
		{"/path/to/file.tmp", "*.tmp", true},
		{"/path/to/file.txt", "*.tmp", false},
		{"/path/.cache/data", ".cache/*", true},
		{"/path/cache/data", ".cache/*", false},
		{"/path/to/node_modules/lib.js", "**/node_modules/**", true},
		{"/path/to/lib.js", "**/node_modules/**", false},
	}

	for _, tt := range tests {
		got := scanner.matchPattern(tt.path, tt.pattern)
		if got != tt.want {
			t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.want)
		}
	}
}

func containsPath(path, substr string) bool {
	return filepath.Base(filepath.Dir(path)) == substr ||
		filepath.Base(path) == substr ||
		filepath.Dir(path) != path && containsPath(filepath.Dir(path), substr)
}
