package backup

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/mackeper/m_backuper/internal/copier"
	"github.com/mackeper/m_backuper/internal/detector"
	"github.com/mackeper/m_backuper/internal/scanner"
	"github.com/mackeper/m_backuper/internal/state"
)

// Backup orchestrates the backup process
type Backup struct {
	scanner  *scanner.Scanner
	detector detector.ChangeDetector
	copier   copier.Copier
	state    *state.State
	deviceID string
}

// New creates a new Backup with the given components
func New(s *scanner.Scanner, d detector.ChangeDetector, c copier.Copier, st *state.State, deviceID string) *Backup {
	return &Backup{
		scanner:  s,
		detector: d,
		copier:   c,
		state:    st,
		deviceID: deviceID,
	}
}

// Run executes the backup process
func (b *Backup) Run(paths []string, backupRoot string) error {
	slog.Info("starting backup", "paths", paths, "device_id", b.deviceID)

	// Scan files
	slog.Info("scanning files...")
	files, err := b.scanner.Scan(paths)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}
	slog.Info("scan complete", "file_count", len(files))

	// Process each file
	copiedCount := 0
	skippedCount := 0
	errorCount := 0

	for _, file := range files {
		// Get file info for change detection
		fileInfo, err := os.Stat(file.Path)
		if err != nil {
			slog.Warn("failed to stat file", "path", file.Path, "error", err)
			errorCount++
			continue
		}

		// Check if file has changed
		fileState, exists := b.state.GetFileState(file.Path)
		detectorState := detector.FileState{
			Size:    fileState.Size,
			ModTime: 0, // We're only using size for now
		}

		if exists && !b.detector.HasChanged(file.Path, fileInfo, detectorState) {
			slog.Debug("file unchanged, skipping", "path", file.Path)
			skippedCount++
			continue
		}

		// Determine destination path
		destPath := filepath.Join(backupRoot, b.deviceID, file.Path)

		// Copy file
		slog.Debug("copying file", "src", file.Path, "dst", destPath)
		_, err = b.copier.Copy(file.Path, destPath)
		if err != nil {
			slog.Error("failed to copy file", "path", file.Path, "error", err)
			errorCount++
			continue
		}

		// Update state
		b.state.SetFileState(file.Path, file.Size)
		copiedCount++
	}

	// Save state
	slog.Info("saving state...")
	if err := b.state.Save(); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	slog.Info("backup complete",
		"total_files", len(files),
		"copied", copiedCount,
		"skipped", skippedCount,
		"errors", errorCount,
	)

	return nil
}
