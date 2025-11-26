package duplicate

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/mackeper/m_backuper/internal/hash"
	"github.com/mackeper/m_backuper/internal/index"
)

// Detector finds duplicate files using multi-stage detection
type Detector struct {
	index   *index.DB
	hasher  *hash.Calculator
	minSize int64
	logger  *slog.Logger
}

// NewDetector creates a new duplicate detector
func NewDetector(idx *index.DB, hasher *hash.Calculator, minSize int64, logger *slog.Logger) *Detector {
	return &Detector{
		index:   idx,
		hasher:  hasher,
		minSize: minSize,
		logger:  logger,
	}
}

// FindDuplicates finds all duplicate file groups using multi-stage detection
func (d *Detector) FindDuplicates(ctx context.Context) ([]index.DuplicateGroup, error) {
	d.logger.Info("starting duplicate detection", "min_size", d.minSize)

	// Stage 1: Get all files from index
	d.logger.Debug("stage 1: loading files from index")
	files, err := d.index.GetAllFiles()
	if err != nil {
		return nil, fmt.Errorf("get files: %w", err)
	}

	d.logger.Info("loaded files from index", "count", len(files))

	// Stage 2: Group by size (fast, in-memory)
	d.logger.Debug("stage 2: grouping by size")
	sizeGroups := make(map[int64][]index.FileRecord)
	for _, file := range files {
		// Skip files below minimum size
		if file.Size < d.minSize {
			continue
		}
		sizeGroups[file.Size] = append(sizeGroups[file.Size], file)
	}

	d.logger.Info("grouped by size", "unique_sizes", len(sizeGroups))

	// Stage 3: For each size group with >1 file, group by hash
	d.logger.Debug("stage 3: grouping by hash")
	hashGroups := make(map[string][]index.FileRecord)
	potentialDuplicates := 0

	for size, filesWithSize := range sizeGroups {
		if len(filesWithSize) < 2 {
			// No duplicates possible for this size
			continue
		}

		d.logger.Debug("potential duplicates by size",
			"size", size,
			"count", len(filesWithSize))

		potentialDuplicates += len(filesWithSize)

		// Group by hash
		for _, file := range filesWithSize {
			hashGroups[file.Hash] = append(hashGroups[file.Hash], file)
		}
	}

	d.logger.Info("potential duplicates by size", "count", potentialDuplicates)

	// Stage 4: Build duplicate groups (only groups with >1 file)
	d.logger.Debug("stage 4: building duplicate groups")
	var groups []index.DuplicateGroup

	for hash, filesWithHash := range hashGroups {
		if len(filesWithHash) < 2 {
			// Not actually duplicates
			continue
		}

		group := index.DuplicateGroup{
			Hash:        hash,
			FileCount:   int64(len(filesWithHash)),
			FileSize:    filesWithHash[0].Size,
			WastedSpace: int64(len(filesWithHash)-1) * filesWithHash[0].Size,
			Files:       filesWithHash,
		}
		groups = append(groups, group)
	}

	// Stage 5: Sort by wasted space (descending)
	d.logger.Debug("stage 5: sorting by wasted space")
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].WastedSpace > groups[j].WastedSpace
	})

	d.logger.Info("duplicate detection complete",
		"duplicate_groups", len(groups),
		"total_files", countTotalFiles(groups),
		"total_wasted", sumWastedSpace(groups))

	return groups, nil
}

// FindDuplicatesInPaths scans specific paths and finds duplicates
// This is useful when you want to scan files not yet in the index
func (d *Detector) FindDuplicatesInPaths(ctx context.Context, paths []string) ([]index.DuplicateGroup, error) {
	d.logger.Info("scanning paths for duplicates", "paths", paths)

	// For now, this method requires files to be in the index
	// A future enhancement could scan files on-the-fly
	return d.FindDuplicates(ctx)
}

// SortBySize sorts duplicate groups by file size (descending)
func SortBySize(groups []index.DuplicateGroup) {
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].FileSize > groups[j].FileSize
	})
}

// SortByCount sorts duplicate groups by file count (descending)
func SortByCount(groups []index.DuplicateGroup) {
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].FileCount > groups[j].FileCount
	})
}

// SortByWastedSpace sorts duplicate groups by wasted space (descending)
func SortByWastedSpace(groups []index.DuplicateGroup) {
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].WastedSpace > groups[j].WastedSpace
	})
}

// FilterByMinWasted filters groups to only include those with at least minWasted bytes
func FilterByMinWasted(groups []index.DuplicateGroup, minWasted int64) []index.DuplicateGroup {
	filtered := make([]index.DuplicateGroup, 0, len(groups))
	for _, group := range groups {
		if group.WastedSpace >= minWasted {
			filtered = append(filtered, group)
		}
	}
	return filtered
}

// Helper functions

func countTotalFiles(groups []index.DuplicateGroup) int64 {
	var total int64
	for _, group := range groups {
		total += group.FileCount
	}
	return total
}

func sumWastedSpace(groups []index.DuplicateGroup) int64 {
	var total int64
	for _, group := range groups {
		total += group.WastedSpace
	}
	return total
}
