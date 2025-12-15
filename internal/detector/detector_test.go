package detector

import (
	"io/fs"
	"testing"
	"time"
)

// mockFileInfo implements fs.FileInfo for testing
//
//nolint:govet // fieldalignment: test struct, performance not critical
type mockFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return m.size }
func (m mockFileInfo) Mode() fs.FileMode  { return m.mode }
func (m mockFileInfo) ModTime() time.Time { return m.modTime }
func (m mockFileInfo) IsDir() bool        { return m.isDir }
func (m mockFileInfo) Sys() interface{}   { return nil }

func TestSizeDetectorReturnsTrueWhenSizeDiffers(t *testing.T) {
	detector := NewSizeDetector()

	// File info with size 100
	info := mockFileInfo{
		name:    "test.txt",
		size:    100,
		modTime: time.Now(),
	}

	// Previous state with size 50
	state := FileState{
		Size:    50,
		ModTime: time.Now().Unix(),
	}

	if !detector.HasChanged("test.txt", info, state) {
		t.Error("expected HasChanged to return true when size differs")
	}
}

func TestSizeDetectorReturnsFalseWhenSizeMatches(t *testing.T) {
	detector := NewSizeDetector()

	// File info with size 100
	info := mockFileInfo{
		name:    "test.txt",
		size:    100,
		modTime: time.Now(),
	}

	// Previous state with size 100 (same)
	state := FileState{
		Size:    100,
		ModTime: time.Now().Unix(),
	}

	if detector.HasChanged("test.txt", info, state) {
		t.Error("expected HasChanged to return false when size matches")
	}
}

func TestSizeDetectorReturnsTrueForNewFile(t *testing.T) {
	detector := NewSizeDetector()

	// File info with size 100
	info := mockFileInfo{
		name:    "test.txt",
		size:    100,
		modTime: time.Now(),
	}

	// Empty state (new file)
	state := FileState{}

	if !detector.HasChanged("test.txt", info, state) {
		t.Error("expected HasChanged to return true for new file (empty state)")
	}
}

func TestInterfaceCanBeSwapped(t *testing.T) {
	// Verify that SizeDetector implements ChangeDetector interface
	var _ ChangeDetector = (*SizeDetector)(nil)

	// Create instances through interface
	detector := ChangeDetector(NewSizeDetector())

	info := mockFileInfo{
		name:    "test.txt",
		size:    100,
		modTime: time.Now(),
	}

	state := FileState{
		Size:    50,
		ModTime: time.Now().Unix(),
	}

	// Should work through interface
	if !detector.HasChanged("test.txt", info, state) {
		t.Error("expected HasChanged to work through interface")
	}
}

func TestSizeDetectorIgnoresModTime(t *testing.T) {
	detector := NewSizeDetector()

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	// File info with size 100 and today's mod time
	info := mockFileInfo{
		name:    "test.txt",
		size:    100,
		modTime: now,
	}

	// Previous state with size 100 (same) but yesterday's mod time (different)
	state := FileState{
		Size:    100,
		ModTime: yesterday.Unix(),
	}

	// Should return false because size hasn't changed (ignores mod time)
	if detector.HasChanged("test.txt", info, state) {
		t.Error("expected HasChanged to ignore mod time and only check size")
	}
}
