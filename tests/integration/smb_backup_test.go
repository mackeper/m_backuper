//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mackeper/m_backuper/internal/backup"
	"github.com/mackeper/m_backuper/internal/copier"
	"github.com/mackeper/m_backuper/internal/detector"
	"github.com/mackeper/m_backuper/internal/pathutil"
	"github.com/mackeper/m_backuper/internal/scanner"
	"github.com/mackeper/m_backuper/internal/state"
)

func TestSMBBackupFullFlow(t *testing.T) {
	// Get SMB mount point from environment
	smbMount := os.Getenv("SMB_MOUNT")
	if smbMount == "" {
		t.Fatal("SMB_MOUNT environment variable not set")
	}

	t.Logf("Using SMB mount: %s", smbMount)

	// Verify mount is accessible
	if err := pathutil.ValidatePath(smbMount); err != nil {
		t.Fatalf("SMB mount not accessible: %v", err)
	}

	// Verify it's recognized as a network path
	if !pathutil.IsNetworkPath(smbMount) {
		t.Logf("Note: Path %s not recognized as network path (this is OK for mounted shares)", smbMount)
	}

	// Create test source data
	srcDir := t.TempDir()
	testFiles := map[string]string{
		"file1.txt":            "content of file 1",
		"file2.txt":            "content of file 2",
		"subdir/file3.txt":     "content of file 3 in subdirectory",
		"subdir/deep/file4.go": "package main\n\nfunc main() {}\n",
	}

	for relPath, content := range testFiles {
		fullPath := filepath.Join(srcDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create backup destination on SMB share
	dstDir := filepath.Join(smbMount, "integration-test-backup")
	defer os.RemoveAll(dstDir) // Cleanup

	// Initialize backup components
	s := scanner.New([]string{})
	d := detector.NewSizeDetector()
	c := copier.NewLocalCopier(dstDir)
	defer c.Close()

	st := state.New()
	deviceID := "integration-test-device"

	// Create and run backup
	b := backup.New(s, d, c, st, deviceID)

	t.Log("Starting backup to SMB share...")
	if err := b.Run([]string{srcDir}, dstDir); err != nil {
		t.Fatalf("Backup failed: %v", err)
	}
	t.Log("Backup completed successfully")

	// Verify all files were backed up to SMB share
	for relPath, expectedContent := range testFiles {
		backupPath := filepath.Join(dstDir, deviceID, srcDir, relPath)
		t.Logf("Verifying file: %s", backupPath)

		content, err := os.ReadFile(backupPath)
		if err != nil {
			t.Errorf("Failed to read backed up file %s: %v", backupPath, err)
			continue
		}

		if string(content) != expectedContent {
			t.Errorf("Content mismatch for %s:\nGot: %q\nExpected: %q",
				relPath, string(content), expectedContent)
		}
	}

	// Verify state was updated
	if st.FileCount() != len(testFiles) {
		t.Errorf("State file count = %d, expected %d", st.FileCount(), len(testFiles))
	}
}

func TestSMBBackupIncremental(t *testing.T) {
	smbMount := os.Getenv("SMB_MOUNT")
	if smbMount == "" {
		t.Fatal("SMB_MOUNT environment variable not set")
	}

	// Create test source data
	srcDir := t.TempDir()
	file1 := filepath.Join(srcDir, "file1.txt")
	file2 := filepath.Join(srcDir, "file2.txt")

	os.WriteFile(file1, []byte("initial content 1"), 0o644)
	os.WriteFile(file2, []byte("initial content 2"), 0o644)

	// Setup backup
	dstDir := filepath.Join(smbMount, "incremental-test-backup")
	defer os.RemoveAll(dstDir)

	s := scanner.New([]string{})
	d := detector.NewSizeDetector()
	c := copier.NewLocalCopier(dstDir)
	defer c.Close()

	st := state.New()
	deviceID := "incremental-test"

	// First backup
	b1 := backup.New(s, d, c, st, deviceID)
	t.Log("Running initial backup...")
	if err := b1.Run([]string{srcDir}, dstDir); err != nil {
		t.Fatalf("Initial backup failed: %v", err)
	}

	if st.FileCount() != 2 {
		t.Errorf("After first backup: state file count = %d, expected 2", st.FileCount())
	}

	// Modify one file
	t.Log("Modifying file1.txt...")
	os.WriteFile(file1, []byte("modified content 1 - much longer"), 0o644)

	// Second backup (incremental)
	b2 := backup.New(s, d, c, st, deviceID)
	t.Log("Running incremental backup...")
	if err := b2.Run([]string{srcDir}, dstDir); err != nil {
		t.Fatalf("Incremental backup failed: %v", err)
	}

	// Verify updated content on SMB share
	backupPath := filepath.Join(dstDir, deviceID, file1)
	content, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("Failed to read updated file from SMB: %v", err)
	}

	expected := "modified content 1 - much longer"
	if string(content) != expected {
		t.Errorf("Updated file content mismatch:\nGot: %q\nExpected: %q",
			string(content), expected)
	}

	t.Log("Incremental backup verified successfully")
}

func TestSMBBackupWithIgnorePatterns(t *testing.T) {
	smbMount := os.Getenv("SMB_MOUNT")
	if smbMount == "" {
		t.Fatal("SMB_MOUNT environment variable not set")
	}

	// Create test source with files to ignore
	srcDir := t.TempDir()
	files := map[string]bool{
		"important.txt":       false, // Should be backed up
		"data.tmp":            true,  // Should be ignored (*.tmp)
		".cache/cache.dat":    true,  // Should be ignored (.cache/*)
		"node_modules/lib.js": true,  // Should be ignored (**/node_modules/**)
		"src/main.go":         false, // Should be backed up
		"build/output.tmp":    true,  // Should be ignored (*.tmp)
	}

	for relPath := range files {
		fullPath := filepath.Join(srcDir, relPath)
		os.MkdirAll(filepath.Dir(fullPath), 0o755)
		os.WriteFile(fullPath, []byte("content"), 0o644)
	}

	// Setup backup with ignore patterns
	ignorePatterns := []string{"*.tmp", ".cache/*", "**/node_modules/**"}
	dstDir := filepath.Join(smbMount, "ignore-pattern-test")
	defer os.RemoveAll(dstDir)

	s := scanner.New(ignorePatterns)
	d := detector.NewSizeDetector()
	c := copier.NewLocalCopier(dstDir)
	defer c.Close()

	st := state.New()
	deviceID := "ignore-test"

	// Run backup
	b := backup.New(s, d, c, st, deviceID)
	t.Log("Running backup with ignore patterns...")
	if err := b.Run([]string{srcDir}, dstDir); err != nil {
		t.Fatalf("Backup failed: %v", err)
	}

	// Verify only non-ignored files were backed up
	for relPath, shouldIgnore := range files {
		backupPath := filepath.Join(dstDir, deviceID, srcDir, relPath)
		_, err := os.Stat(backupPath)

		if shouldIgnore {
			if !os.IsNotExist(err) {
				t.Errorf("File %s should have been ignored but was backed up", relPath)
			}
		} else {
			if err != nil {
				t.Errorf("File %s should have been backed up but wasn't: %v", relPath, err)
			}
		}
	}

	// Should have backed up exactly 2 files (important.txt and src/main.go)
	expectedCount := 2
	if st.FileCount() != expectedCount {
		t.Errorf("State file count = %d, expected %d", st.FileCount(), expectedCount)
	}

	t.Log("Ignore patterns verified successfully")
}

func TestSMBMountValidation(t *testing.T) {
	smbMount := os.Getenv("SMB_MOUNT")
	if smbMount == "" {
		t.Fatal("SMB_MOUNT environment variable not set")
	}

	t.Run("mount is accessible", func(t *testing.T) {
		err := pathutil.ValidatePath(smbMount)
		if err != nil {
			t.Errorf("SMB mount validation failed: %v", err)
		}
	})

	t.Run("can write to mount", func(t *testing.T) {
		testFile := filepath.Join(smbMount, "write-test.txt")
		defer os.Remove(testFile)

		if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
			t.Errorf("Cannot write to SMB mount: %v", err)
		}

		// Verify we can read it back
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Errorf("Cannot read from SMB mount: %v", err)
		}

		if string(content) != "test" {
			t.Errorf("Content mismatch after write/read to SMB")
		}
	})

	t.Run("can create directories", func(t *testing.T) {
		testDir := filepath.Join(smbMount, "test-dir")
		defer os.RemoveAll(testDir)

		if err := os.MkdirAll(testDir, 0o755); err != nil {
			t.Errorf("Cannot create directory on SMB mount: %v", err)
		}

		info, err := os.Stat(testDir)
		if err != nil || !info.IsDir() {
			t.Errorf("Created directory is not accessible: %v", err)
		}
	})
}
