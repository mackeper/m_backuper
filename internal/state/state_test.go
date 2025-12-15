package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	state := New()

	if state == nil {
		t.Fatal("New() returned nil")
	}
	if state.Files == nil {
		t.Fatal("Files map not initialized")
	}
	if len(state.Files) != 0 {
		t.Error("expected empty Files map")
	}
}

func TestSaveStateToFile(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	state := New()
	state.SetFileState("/path/to/file1.txt", 1024)
	state.SetFileState("/path/to/file2.txt", 2048)

	if err := state.SaveTo(statePath); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Fatal("state file was not created")
	}

	// Verify file count
	if state.FileCount() != 2 {
		t.Errorf("expected 2 files, got %d", state.FileCount())
	}
}

func TestLoadStateFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Save initial state
	originalState := New()
	originalState.SetFileState("/path/to/file1.txt", 1024)
	originalState.SetFileState("/path/to/file2.txt", 2048)
	if err := originalState.SaveTo(statePath); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	// Load state
	loadedState, err := LoadFrom(statePath)
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}

	// Verify loaded state
	if loadedState.FileCount() != 2 {
		t.Errorf("expected 2 files, got %d", loadedState.FileCount())
	}

	// Verify specific file states
	file1State, exists := loadedState.GetFileState("/path/to/file1.txt")
	if !exists {
		t.Error("file1.txt not found in loaded state")
	}
	if file1State.Size != 1024 {
		t.Errorf("expected size 1024, got %d", file1State.Size)
	}

	file2State, exists := loadedState.GetFileState("/path/to/file2.txt")
	if !exists {
		t.Error("file2.txt not found in loaded state")
	}
	if file2State.Size != 2048 {
		t.Errorf("expected size 2048, got %d", file2State.Size)
	}
}

func TestHandleMissingStateFile(t *testing.T) {
	nonExistentPath := filepath.Join(t.TempDir(), "nonexistent.json")

	// Loading non-existent file should not error (first run scenario)
	state, err := LoadFrom(nonExistentPath)
	if err != nil {
		t.Errorf("expected no error for missing state file, got: %v", err)
	}
	if state.FileCount() != 0 {
		t.Error("expected empty state for missing file")
	}
}

func TestHandleCorruptedStateFile(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Write invalid JSON
	invalidJSON := []byte(`{this is not valid json}`)
	if err := os.WriteFile(statePath, invalidJSON, 0644); err != nil {
		t.Fatalf("failed to write invalid state file: %v", err)
	}

	// Loading corrupted file should return error
	_, err := LoadFrom(statePath)
	if err == nil {
		t.Error("expected error for corrupted state file, got nil")
	}
}

func TestFileWriteErrors(t *testing.T) {
	// Try to write to a directory that cannot be created (read-only parent)
	// This is hard to test portably, so we'll test a simpler case:
	// trying to write to a file path that is actually a directory
	tmpDir := t.TempDir()
	dirAsFile := filepath.Join(tmpDir, "should-be-file")

	// Create a directory with the same name as the file we want to write
	if err := os.Mkdir(dirAsFile, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	state := New()
	err := state.SaveTo(dirAsFile)
	if err == nil {
		t.Error("expected error when trying to write to a directory, got nil")
	}
}

func TestSetAndGetFileState(t *testing.T) {
	state := New()

	path := "/path/to/test.txt"
	size := int64(12345)

	// Initially, file should not exist
	_, exists := state.GetFileState(path)
	if exists {
		t.Error("file should not exist initially")
	}

	// Set file state
	state.SetFileState(path, size)

	// Now file should exist
	fileState, exists := state.GetFileState(path)
	if !exists {
		t.Error("file should exist after SetFileState")
	}
	if fileState.Size != size {
		t.Errorf("expected size %d, got %d", size, fileState.Size)
	}
	if fileState.BackedUp == "" {
		t.Error("BackedUp timestamp should not be empty")
	}

	// Verify timestamp is valid
	_, err := time.Parse(time.RFC3339, fileState.BackedUp)
	if err != nil {
		t.Errorf("invalid BackedUp timestamp: %v", err)
	}
}

func TestRemoveFileState(t *testing.T) {
	state := New()

	path := "/path/to/test.txt"
	state.SetFileState(path, 1024)

	// Verify file exists
	_, exists := state.GetFileState(path)
	if !exists {
		t.Error("file should exist after SetFileState")
	}

	// Remove file
	state.RemoveFileState(path)

	// Verify file no longer exists
	_, exists = state.GetFileState(path)
	if exists {
		t.Error("file should not exist after RemoveFileState")
	}
}

func TestLastRunUpdatedOnSave(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	state := New()

	// LastRun should be zero initially
	if !state.LastRun.IsZero() {
		t.Error("LastRun should be zero initially")
	}

	// Save state
	before := time.Now()
	if err := state.SaveTo(statePath); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}
	after := time.Now()

	// LastRun should be updated
	if state.LastRun.IsZero() {
		t.Error("LastRun should not be zero after save")
	}
	if state.LastRun.Before(before) || state.LastRun.After(after) {
		t.Error("LastRun timestamp is outside expected range")
	}

	// Load state and verify LastRun persisted
	loadedState, err := LoadFrom(statePath)
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}
	if loadedState.LastRun.IsZero() {
		t.Error("LastRun should not be zero in loaded state")
	}
}
