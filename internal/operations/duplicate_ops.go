package operations

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mackeper/m_backuper/internal/config"
	"github.com/mackeper/m_backuper/internal/duplicate"
	"github.com/mackeper/m_backuper/internal/hash"
	"github.com/mackeper/m_backuper/internal/index"
)

// DuplicateOperation handles duplicate file detection and cleanup
type DuplicateOperation struct {
	db     *index.DB
	cfg    *config.Config
	logger *slog.Logger
}

// FindOptions configures how duplicates are found and filtered
type FindOptions struct {
	SortBy    string // "size", "count", "wasted"
	MinWasted int64  // Minimum wasted space to include
	BackupSet string // Filter by backup set (empty = all)
}

// CleanOptions configures how duplicates are cleaned
type CleanOptions struct {
	Groups   []index.DuplicateGroup      // Groups to clean
	Strategy duplicate.KeepStrategy      // Which files to keep
	DryRun   bool                        // Preview without deleting
	Progress ProgressCallback            // Optional progress callback
}

// NewDuplicateOperation creates a new duplicate operation
func NewDuplicateOperation(db *index.DB, cfg *config.Config, logger *slog.Logger) *DuplicateOperation {
	return &DuplicateOperation{
		db:     db,
		cfg:    cfg,
		logger: logger,
	}
}

// FindDuplicates finds all duplicate groups with optional filtering and sorting
func (op *DuplicateOperation) FindDuplicates(ctx context.Context, opts FindOptions) ([]index.DuplicateGroup, error) {
	// Create detector
	hasher := hash.NewCalculator(op.cfg.Concurrency.HashWorkers)
	detector := duplicate.NewDetector(op.db, hasher, op.cfg.Duplicates.MinFileSize, op.logger)

	// Find duplicates
	groups, err := detector.FindDuplicates(ctx)
	if err != nil {
		return nil, fmt.Errorf("find duplicates: %w", err)
	}

	// Filter by minimum wasted space
	if opts.MinWasted > 0 {
		groups = duplicate.FilterByMinWasted(groups, opts.MinWasted)
	}

	// Sort according to options
	switch opts.SortBy {
	case "size":
		duplicate.SortBySize(groups)
	case "count":
		duplicate.SortByCount(groups)
	case "wasted", "":
		duplicate.SortByWastedSpace(groups)
	default:
		return nil, fmt.Errorf("invalid sort option: %s (use size, count, or wasted)", opts.SortBy)
	}

	return groups, nil
}

// CleanDuplicates deletes duplicate files based on the specified strategy
func (op *DuplicateOperation) CleanDuplicates(ctx context.Context, opts CleanOptions) (*CleanResult, error) {
	result := &CleanResult{
		FilesDeleted: 0,
		SpaceFreed:   0,
		Errors:       []CleanError{},
	}

	// Create cleaner
	cleaner := duplicate.NewCleaner(op.db, op.logger)

	// Process each group
	for i, group := range opts.Groups {
		// Report progress if callback provided
		if opts.Progress != nil {
			opts.Progress(OperationProgress{
				Stage:         "cleaning",
				FilesTotal:    int64(len(opts.Groups)),
				FilesComplete: int64(i),
				CurrentFile:   group.Hash[:16] + "...",
				Percentage:    float64(i) / float64(len(opts.Groups)) * 100,
			})
		}

		// Select files to delete based on strategy
		toDelete := duplicate.SelectFilesToDelete(group, opts.Strategy)

		if len(toDelete) == 0 {
			continue
		}

		// Delete files
		deleteResults, err := cleaner.DeleteFiles(toDelete, opts.DryRun)
		if err != nil {
			return result, fmt.Errorf("delete files: %w", err)
		}

		// Accumulate results
		for _, dr := range deleteResults {
			if dr.Err == nil {
				result.FilesDeleted++
				result.SpaceFreed += dr.Size
			} else {
				result.Errors = append(result.Errors, CleanError{
					Path:  dr.Path,
					Error: dr.Err,
				})
			}
		}
	}

	// Final progress update
	if opts.Progress != nil {
		opts.Progress(OperationProgress{
			Stage:         "complete",
			FilesTotal:    int64(len(opts.Groups)),
			FilesComplete: int64(len(opts.Groups)),
			Percentage:    100,
		})
	}

	return result, nil
}
