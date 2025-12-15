package copier

import "io"

// Copier is a pluggable interface for copying files to a destination
type Copier interface {
	// Copy copies a file from src to dst, preserving directory structure
	// Returns the number of bytes copied and any error
	Copy(src, dst string) (int64, error)

	// Close closes any open connections or resources
	Close() error
}

// copyFile is a helper function to copy file contents
func copyFile(src, dst string, copyFunc func(io.Reader, io.Writer) (int64, error)) (int64, error) {
	// This will be implemented by specific copier implementations
	return 0, nil
}
