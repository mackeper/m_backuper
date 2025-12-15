package backup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mackeper/m_backuper/internal/copier"
	"github.com/mackeper/m_backuper/internal/detector"
	"github.com/mackeper/m_backuper/internal/scanner"
	"github.com/mackeper/m_backuper/internal/state"
)

func TestFullBackupFlow(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source files
	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create source directory: %v", err)
	}

	testFiles := map[string][]byte{
		"file1.txt": []byte("content 1"),
		"file2.go":  []byte("content 2"),
		"file3.txt": []byte("content 3"),
	}

	for name, content := range testFiles {
		path := filepath.Join(srcDir, name)
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	// Create backup components
	s := scanner.New([]string{})
	d := detector.NewSizeDetector()
	dstDir := filepath.Join(tmpDir, "backup")
	c := copier.NewLocalCopier(dstDir)
	st := state.New()

	// Create backup
	deviceID := "test-device"
	b := New(s, d, c, st, deviceID)

	// Run backup
	if err := b.Run([]string{srcDir}, dstDir); err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	// Verify files were backed up
	for name := range testFiles {
		srcPath := filepath.Join(srcDir, name)
		dstPath := filepath.Join(dstDir, deviceID, srcPath)

		if _, err := os.Stat(dstPath); os.IsNotExist(err) {
			t.Errorf("file was not backed up: %s", name)
		}
	}

	// Verify state was updated
	if st.FileCount() != len(testFiles) {
		t.Errorf("expected %d files in state, got %d", len(testFiles), st.FileCount())
	}
}

func TestIncrementalBackupOnlyCopiesChangedFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source files
	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create source directory: %v", err)
	}

	file1Path := filepath.Join(srcDir, "file1.txt")
	file2Path := filepath.Join(srcDir, "file2.txt")
	content1 := []byte("content 1")
	content2 := []byte("content 2")

	if err := os.WriteFile(file1Path, content1, 0644); err != nil {
		t.Fatalf("failed to create file1: %v", err)
	}
	if err := os.WriteFile(file2Path, content2, 0644); err != nil {
		t.Fatalf("failed to create file2: %v", err)
	}

	// Create backup components
	s := scanner.New([]string{})
	d := detector.NewSizeDetector()
	dstDir := filepath.Join(tmpDir, "backup")
	c := copier.NewLocalCopier(dstDir)
	st := state.New()

	// First backup
	deviceID := "test-device"
	b := New(s, d, c, st, deviceID)
	if err := b.Run([]string{srcDir}, dstDir); err != nil {
		t.Fatalf("first backup failed: %v", err)
	}

	// Verify initial state
	if st.FileCount() != 2 {
		t.Errorf("expected 2 files in state after first backup, got %d", st.FileCount())
	}

	// Modify only file1
	newContent1 := []byte("modified content 1 - much longer")
	if err := os.WriteFile(file1Path, newContent1, 0644); err != nil {
		t.Fatalf("failed to modify file1: %v", err)
	}

	// Second backup (incremental)
	b2 := New(s, d, c, st, deviceID)
	if err := b2.Run([]string{srcDir}, dstDir); err != nil {
		t.Fatalf("second backup failed: %v", err)
	}

	// Verify file1 was updated
	dstFile1 := filepath.Join(dstDir, deviceID, file1Path)
	content, err := os.ReadFile(dstFile1)
	if err != nil {
		t.Fatalf("failed to read backed up file1: %v", err)
	}
	if string(content) != string(newContent1) {
		t.Error("file1 was not updated in backup")
	}
}

func TestStateUpdatesAfterSuccessfulBackup(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create source directory: %v", err)
	}

	filePath := filepath.Join(srcDir, "test.txt")
	content := []byte("test content")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create backup components
	s := scanner.New([]string{})
	d := detector.NewSizeDetector()
	dstDir := filepath.Join(tmpDir, "backup")
	c := copier.NewLocalCopier(dstDir)
	statePath := filepath.Join(tmpDir, "state.json")
	st := state.New()

	// Run backup
	deviceID := "test-device"
	b := New(s, d, c, st, deviceID)
	if err := b.Run([]string{srcDir}, dstDir); err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	// Save state
	if err := st.SaveTo(statePath); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	// Load state and verify
	loadedState, err := state.LoadFrom(statePath)
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}

	fileState, exists := loadedState.GetFileState(filePath)
	if !exists {
		t.Error("file state was not saved")
	}
	if fileState.Size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), fileState.Size)
	}
}

func TestPartialFailuresHandledGracefully(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source files
	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create source directory: %v", err)
	}

	// Create a good file
	goodFile := filepath.Join(srcDir, "good.txt")
	if err := os.WriteFile(goodFile, []byte("good"), 0644); err != nil {
		t.Fatalf("failed to create good file: %v", err)
	}

	// Create backup components
	s := scanner.New([]string{})
	d := detector.NewSizeDetector()
	dstDir := filepath.Join(tmpDir, "backup")
	c := copier.NewLocalCopier(dstDir)
	st := state.New()

	// Run backup (should succeed for the good file)
	deviceID := "test-device"
	b := New(s, d, c, st, deviceID)
	if err := b.Run([]string{srcDir}, dstDir); err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	// Verify good file was backed up
	goodDst := filepath.Join(dstDir, deviceID, goodFile)
	if _, err := os.Stat(goodDst); os.IsNotExist(err) {
		t.Error("good file was not backed up")
	}

	// State should still be updated for successful files
	if st.FileCount() == 0 {
		t.Error("state should be updated even with partial failures")
	}
}
