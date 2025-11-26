package duplicate

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/mackeper/m_backuper/internal/index"
)

// Cleaner handles deletion of duplicate files
type Cleaner struct {
	index  *index.DB
	logger *slog.Logger
}

// NewCleaner creates a new duplicate cleaner
func NewCleaner(idx *index.DB, logger *slog.Logger) *Cleaner {
	return &Cleaner{
		index:  idx,
		logger: logger,
	}
}

// DeleteResult contains the result of a delete operation
type DeleteResult struct {
	Path        string
	Size        int64
	Deleted     bool
	Err         error
}

// DeleteFiles deletes the specified file paths and updates the index
func (c *Cleaner) DeleteFiles(paths []string, dryRun bool) ([]DeleteResult, error) {
	results := make([]DeleteResult, 0, len(paths))

	for _, path := range paths {
		result := DeleteResult{
			Path: path,
		}

		// Get file info before deletion
		info, err := os.Stat(path)
		if err != nil {
			result.Err = fmt.Errorf("stat file: %w", err)
			results = append(results, result)
			c.logger.Error("failed to stat file", "path", path, "error", err)
			continue
		}

		result.Size = info.Size()

		if dryRun {
			// In dry run mode, just log what would be deleted
			c.logger.Info("would delete file", "path", path, "size", result.Size)
			result.Deleted = false
		} else {
			// Actually delete the file
			if err := os.Remove(path); err != nil {
				result.Err = fmt.Errorf("remove file: %w", err)
				results = append(results, result)
				c.logger.Error("failed to delete file", "path", path, "error", err)
				continue
			}

			// Update index
			if err := c.index.DeleteFile(path); err != nil {
				c.logger.Warn("failed to remove from index", "path", path, "error", err)
				// Don't fail the operation if index update fails
			}

			c.logger.Info("deleted file", "path", path, "size", result.Size)
			result.Deleted = true
		}

		results = append(results, result)
	}

	return results, nil
}

// SelectByStrategy selects which files to keep/delete based on a strategy
type KeepStrategy string

const (
	KeepOldest  KeepStrategy = "oldest"
	KeepNewest  KeepStrategy = "newest"
	KeepFirst   KeepStrategy = "first"
	KeepShortest KeepStrategy = "shortest" // Shortest path
)

// SelectFilesToDelete returns the paths of files that should be deleted
// based on the keep strategy
func SelectFilesToDelete(group index.DuplicateGroup, strategy KeepStrategy) []string {
	if len(group.Files) < 2 {
		return nil
	}

	// Determine which file to keep
	keepIdx := 0

	switch strategy {
	case KeepOldest:
		// Keep the file with the oldest modification time
		for i := 1; i < len(group.Files); i++ {
			if group.Files[i].ModTime.Before(group.Files[keepIdx].ModTime) {
				keepIdx = i
			}
		}

	case KeepNewest:
		// Keep the file with the newest modification time
		for i := 1; i < len(group.Files); i++ {
			if group.Files[i].ModTime.After(group.Files[keepIdx].ModTime) {
				keepIdx = i
			}
		}

	case KeepFirst:
		// Keep the first file (default keepIdx = 0)

	case KeepShortest:
		// Keep the file with the shortest path
		for i := 1; i < len(group.Files); i++ {
			if len(group.Files[i].Path) < len(group.Files[keepIdx].Path) {
				keepIdx = i
			}
		}
	}

	// Build list of files to delete (all except the one to keep)
	toDelete := make([]string, 0, len(group.Files)-1)
	for i, file := range group.Files {
		if i != keepIdx {
			toDelete = append(toDelete, file.Path)
		}
	}

	return toDelete
}
