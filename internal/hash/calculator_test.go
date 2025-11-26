package hash

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestHashFile(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	content := []byte("Hello, World!")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Calculate expected hash
	expectedHash := sha256.Sum256(content)
	expectedHashStr := hex.EncodeToString(expectedHash[:])

	// Create calculator and hash the file
	calc := NewCalculator(1)
	hash, err := calc.HashFile(testFile)

	if err != nil {
		t.Fatalf("HashFile failed: %v", err)
	}

	if hash != expectedHashStr {
		t.Errorf("Hash mismatch: got %s, want %s", hash, expectedHashStr)
	}
}

func TestHashFileNonExistent(t *testing.T) {
	calc := NewCalculator(1)
	_, err := calc.HashFile("/nonexistent/file.txt")

	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestHashPartial(t *testing.T) {
	// Create a test file with known content
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	content := []byte("Hello, World! This is a longer message.")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Hash only first 10 bytes
	calc := NewCalculator(1)
	hash, err := calc.HashPartial(testFile, 10)

	if err != nil {
		t.Fatalf("HashPartial failed: %v", err)
	}

	// Calculate expected hash of first 10 bytes
	expectedHash := sha256.Sum256(content[:10])
	expectedHashStr := hex.EncodeToString(expectedHash[:])

	if hash != expectedHashStr {
		t.Errorf("Partial hash mismatch: got %s, want %s", hash, expectedHashStr)
	}
}

func TestHashFilesParallel(t *testing.T) {
	// Create multiple test files
	tmpDir := t.TempDir()

	files := []struct {
		name    string
		content string
	}{
		{"file1.txt", "Content of file 1"},
		{"file2.txt", "Content of file 2"},
		{"file3.txt", "Content of file 3"},
	}

	expectedHashes := make(map[string]string)

	for _, f := range files {
		path := filepath.Join(tmpDir, f.name)
		content := []byte(f.content)
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", f.name, err)
		}

		// Calculate expected hash
		hash := sha256.Sum256(content)
		expectedHashes[path] = hex.EncodeToString(hash[:])
	}

	// Hash files in parallel
	calc := NewCalculator(2)
	ctx := context.Background()

	jobs := make(chan HashJob, len(files))
	for _, f := range files {
		path := filepath.Join(tmpDir, f.name)
		info, _ := os.Stat(path)
		jobs <- HashJob{Path: path, Size: info.Size()}
	}
	close(jobs)

	results := calc.HashFiles(ctx, jobs)

	// Collect results
	resultCount := 0
	for result := range results {
		resultCount++

		if result.Err != nil {
			t.Errorf("Hashing failed for %s: %v", result.Path, result.Err)
			continue
		}

		expectedHash, ok := expectedHashes[result.Path]
		if !ok {
			t.Errorf("Unexpected file in results: %s", result.Path)
			continue
		}

		if result.Hash != expectedHash {
			t.Errorf("Hash mismatch for %s: got %s, want %s", result.Path, result.Hash, expectedHash)
		}
	}

	if resultCount != len(files) {
		t.Errorf("Expected %d results, got %d", len(files), resultCount)
	}
}

func TestHashFilesWithCancellation(t *testing.T) {
	// Create test files
	tmpDir := t.TempDir()

	for i := 0; i < 5; i++ {
		path := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt")
		if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	calc := NewCalculator(2)
	ctx, cancel := context.WithCancel(context.Background())

	jobs := make(chan HashJob, 5)
	for i := 0; i < 5; i++ {
		path := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt")
		jobs <- HashJob{Path: path, Size: 12}
	}
	close(jobs)

	results := calc.HashFiles(ctx, jobs)

	// Cancel after receiving one result
	result := <-results
	if result.Err != nil {
		t.Errorf("First result had error: %v", result.Err)
	}

	cancel()

	// Drain remaining results
	for range results {
		// Just drain
	}

	// Test passes if no panic and context cancellation works
}

func TestNewCalculator(t *testing.T) {
	tests := []struct {
		name       string
		numWorkers int
		wantMin    int // Minimum expected workers (at least 1)
	}{
		{"Explicit workers", 4, 4},
		{"Zero workers (auto)", 0, 1}, // Should use runtime.NumCPU()
		{"Negative workers (auto)", -1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calc := NewCalculator(tt.numWorkers)

			if calc == nil {
				t.Fatal("NewCalculator returned nil")
			}

			if calc.numWorkers < tt.wantMin {
				t.Errorf("numWorkers = %d, want at least %d", calc.numWorkers, tt.wantMin)
			}
		})
	}
}
