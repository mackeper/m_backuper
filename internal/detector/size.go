package detector

import "io/fs"

// SizeDetector detects changes based on file size only
type SizeDetector struct{}

// NewSizeDetector creates a new SizeDetector
func NewSizeDetector() *SizeDetector {
	return &SizeDetector{}
}

// HasChanged returns true if the file size has changed
func (d *SizeDetector) HasChanged(path string, info fs.FileInfo, state FileState) bool {
	// If state has zero values, this is a new file
	if state.Size == 0 && state.ModTime == 0 {
		return true
	}

	// Check if size has changed
	return info.Size() != state.Size
}
