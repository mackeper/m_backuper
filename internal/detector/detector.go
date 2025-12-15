package detector

import "io/fs"

type FileState struct {
	Size    int64
	ModTime int64 // Unix timestamp
}

type ChangeDetector interface {
	HasChanged(path string, info fs.FileInfo, state FileState) bool
}
