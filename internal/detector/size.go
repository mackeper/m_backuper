package detector

import "io/fs"

type SizeDetector struct{}

func NewSizeDetector() *SizeDetector {
	return &SizeDetector{}
}

func (d *SizeDetector) HasChanged(path string, info fs.FileInfo, state FileState) bool {
	// If state has zero values, this is a new file
	if state.Size == 0 && state.ModTime == 0 {
		return true
	}

	// Check if size has changed
	return info.Size() != state.Size
}
