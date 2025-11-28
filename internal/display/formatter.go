package display

import (
	"fmt"

	"github.com/mackeper/m_backuper/internal/config"
	"github.com/mackeper/m_backuper/internal/index"
)

// Formatter defines the interface for different output formats
type Formatter interface {
	// FormatDuplicateGroups formats duplicate file groups
	FormatDuplicateGroups(groups []index.DuplicateGroup) (string, error)

	// FormatStats formats database statistics
	FormatStats(stats *DatabaseStats) (string, error)

	// FormatBackupSets formats backup set configurations
	FormatBackupSets(sets []config.BackupSet) (string, error)
}

// DatabaseStats holds statistics about the indexed database
type DatabaseStats struct {
	TotalFiles      int64
	TotalSize       int64
	DatabaseSize    int64
	DatabasePath    string
	BackupSets      []BackupSetStats
	RootDirectories []RootDirStats
}

// BackupSetStats holds statistics for a single backup set
type BackupSetStats struct {
	Name  string
	Count int64
	Size  int64
}

// RootDirStats holds statistics for a root directory
type RootDirStats struct {
	Path  string
	Count int
}

// DuplicateSummary holds summary information about duplicate groups
type DuplicateSummary struct {
	GroupCount  int
	TotalWasted int64
	TotalFiles  int64
}

// NewFormatter creates a formatter based on the format type
func NewFormatter(format string) (Formatter, error) {
	switch format {
	case "table":
		return &TableFormatter{}, nil
	case "json":
		return &JSONFormatter{}, nil
	default:
		return nil, fmt.Errorf("unknown format: %s (use table or json)", format)
	}
}
