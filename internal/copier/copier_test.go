package copier

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalCopierCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create source directory: %v", err)
	}

	srcFile := filepath.Join(srcDir, "test.txt")
	testContent := []byte("test content for copying")
	if err := os.WriteFile(srcFile, testContent, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Create copier
	dstRoot := filepath.Join(tmpDir, "dst")
	copier := NewLocalCopier(dstRoot)

	// Copy file
	dstFile := filepath.Join(dstRoot, "test.txt")
	bytesCopied, err := copier.Copy(srcFile, dstFile)
	if err != nil {
		t.Fatalf("failed to copy file: %v", err)
	}

	// Verify bytes copied
	if bytesCopied != int64(len(testContent)) {
		t.Errorf("expected %d bytes copied, got %d", len(testContent), bytesCopied)
	}

	// Verify destination file exists
	if _, err := os.Stat(dstFile); os.IsNotExist(err) {
		t.Fatal("destination file was not created")
	}

	// Verify destination file content
	dstContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("failed to read destination file: %v", err)
	}

	if string(dstContent) != string(testContent) {
		t.Errorf("content mismatch: expected %q, got %q", testContent, dstContent)
	}
}

func TestLocalCopierPreserveDirectoryStructure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file in nested directory
	srcDir := filepath.Join(tmpDir, "src", "level1", "level2")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create source directory: %v", err)
	}

	srcFile := filepath.Join(srcDir, "test.txt")
	testContent := []byte("test content")
	if err := os.WriteFile(srcFile, testContent, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Create copier
	dstRoot := filepath.Join(tmpDir, "dst")
	copier := NewLocalCopier(dstRoot)

	// Copy file to destination with same nested structure
	dstFile := filepath.Join(dstRoot, "level1", "level2", "test.txt")
	_, err := copier.Copy(srcFile, dstFile)
	if err != nil {
		t.Fatalf("failed to copy file: %v", err)
	}

	// Verify destination file exists in nested directory
	if _, err := os.Stat(dstFile); os.IsNotExist(err) {
		t.Fatal("destination file was not created in nested directory")
	}

	// Verify directory structure was created
	dstDir := filepath.Join(dstRoot, "level1", "level2")
	if _, err := os.Stat(dstDir); os.IsNotExist(err) {
		t.Fatal("destination directory structure was not created")
	}
}

func TestLocalCopierHandleCopyErrors(t *testing.T) {
	tmpDir := t.TempDir()

	// Try to copy non-existent file
	copier := NewLocalCopier(tmpDir)
	_, err := copier.Copy("/nonexistent/file.txt", filepath.Join(tmpDir, "dst.txt"))
	if err == nil {
		t.Error("expected error when copying non-existent file, got nil")
	}
}

func TestLocalCopierCreateDestinationDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	srcFile := filepath.Join(tmpDir, "src.txt")
	testContent := []byte("test")
	if err := os.WriteFile(srcFile, testContent, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Create copier
	dstRoot := filepath.Join(tmpDir, "dst")
	copier := NewLocalCopier(dstRoot)

	// Copy to destination in non-existent nested directory
	dstFile := filepath.Join(dstRoot, "new", "nested", "dir", "file.txt")
	_, err := copier.Copy(srcFile, dstFile)
	if err != nil {
		t.Fatalf("failed to copy file: %v", err)
	}

	// Verify destination directory was created
	dstDir := filepath.Join(dstRoot, "new", "nested", "dir")
	if _, err := os.Stat(dstDir); os.IsNotExist(err) {
		t.Fatal("destination directory was not created automatically")
	}
}

func TestLocalCopierImplementsCopierInterface(t *testing.T) {
	// Verify LocalCopier implements Copier interface
	var _ Copier = (*LocalCopier)(nil)
}

func TestLocalCopierClose(t *testing.T) {
	copier := NewLocalCopier("/tmp/test")

	// Close should not error
	if err := copier.Close(); err != nil {
		t.Errorf("Close() returned unexpected error: %v", err)
	}
}
