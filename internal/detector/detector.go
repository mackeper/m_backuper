package detector

import "io/fs"

// FileState represents the stored state of a file from a previous backup
type FileState struct {
	Size    int64
	ModTime int64 // Unix timestamp
}

// ChangeDetector is a pluggable interface for detecting file changes
type ChangeDetector interface {
	HasChanged(path string, info fs.FileInfo, state FileState) bool
}
