package copier

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// LocalCopier copies files to a local filesystem destination
type LocalCopier struct {
	destRoot string
}

// NewLocalCopier creates a new LocalCopier with the given destination root
func NewLocalCopier(destRoot string) *LocalCopier {
	return &LocalCopier{
		destRoot: destRoot,
	}
}

// Copy copies a file from src to the destination, preserving directory structure
func (c *LocalCopier) Copy(src, dst string) (int64, error) {
	slog.Debug("copying file", "src", src, "dst", dst)

	// Create destination directory if it doesn't exist
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		slog.Error("failed to create destination directory", "dir", dstDir, "error", err)
		return 0, fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		slog.Error("failed to open source file", "src", src, "error", err)
		return 0, fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		slog.Error("failed to create destination file", "dst", dst, "error", err)
		return 0, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy file contents
	bytesCopied, err := io.Copy(dstFile, srcFile)
	if err != nil {
		slog.Error("failed to copy file contents", "src", src, "dst", dst, "error", err)
		return bytesCopied, fmt.Errorf("failed to copy file contents: %w", err)
	}

	slog.Info("copied file", "src", src, "dst", dst, "bytes", bytesCopied)
	return bytesCopied, nil
}

// Close closes any open connections (no-op for local copier)
func (c *LocalCopier) Close() error {
	return nil
}
