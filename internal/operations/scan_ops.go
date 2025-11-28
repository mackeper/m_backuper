package operations

import (
	"context"
	"log/slog"
	"time"

	"github.com/mackeper/m_backuper/internal/backup"
	"github.com/mackeper/m_backuper/internal/hash"
	"github.com/mackeper/m_backuper/internal/index"
)

// ScanOperation handles scanning directories and indexing files
type ScanOperation struct {
	db       *index.DB
	logger   *slog.Logger
	progress ProgressCallback
}

// ScanOptions configures how the scan is performed
type ScanOptions struct {
	Paths       []string         // Paths to scan
	MinSize     int64            // Minimum file size to scan
	UpdateIndex bool             // Whether to update the database index
	BackupSet   string           // Optional backup set name for indexing
	HashWorkers int              // Number of parallel hash workers
	Progress    ProgressCallback // Optional progress callback
}

// ScanResult represents the result of a scan operation
type ScanResult struct {
	FilesScanned int64         // Number of files scanned
	BytesScanned int64         // Bytes scanned
	FilesIndexed int64         // Number of files added to index
	Errors       int64         // Number of errors encountered
	Duration     time.Duration // Time taken
}

// NewScanOperation creates a new scan operation
func NewScanOperation(db *index.DB, logger *slog.Logger) *ScanOperation {
	return &ScanOperation{
		db:     db,
		logger: logger,
	}
}

// Run executes a scan operation
func (op *ScanOperation) Run(ctx context.Context, opts ScanOptions) (*ScanResult, error) {
	startTime := time.Now()

	result := &ScanResult{
		FilesScanned: 0,
		BytesScanned: 0,
		FilesIndexed: 0,
		Errors:       0,
	}

	// Create walker (no exclusions for scan)
	walker := backup.NewWalker(nil)

	// Create hasher
	hasher := hash.NewCalculator(opts.HashWorkers)

	// Walk paths
	files, walkErrs := walker.WalkMultiple(ctx, opts.Paths)

	// Create hash jobs channel
	hashJobs := make(chan hash.HashJob)

	// Start hash workers
	hashResults := hasher.HashFiles(ctx, hashJobs)

	// Process files
	go func() {
		defer close(hashJobs)

		for file := range files {
			// Skip files below minimum size
			if file.Size < opts.MinSize {
				continue
			}

			op.logger.Debug("scanning file", "path", file.Path, "size", file.Size)

			// Send to hasher
			select {
			case hashJobs <- hash.HashJob{Path: file.Path, Size: file.Size}:
				result.FilesScanned++
				result.BytesScanned += file.Size

				// Report progress
				if opts.Progress != nil {
					opts.Progress(OperationProgress{
						Stage:         "scanning",
						FilesComplete: result.FilesScanned,
						BytesComplete: result.BytesScanned,
						CurrentFile:   file.Path,
					})
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	// Collect hash results
	resultsChan := make(chan struct{})
	go func() {
		defer close(resultsChan)

		for hashResult := range hashResults {
			if hashResult.Err != nil {
				op.logger.Error("hash failed", "path", hashResult.Path, "error", hashResult.Err)
				result.Errors++
				continue
			}

			op.logger.Debug("hashed file", "path", hashResult.Path, "hash", hashResult.Hash[:16])

			if opts.UpdateIndex {
				// Update index
				fileRecord := &index.FileRecord{
					Path:      hashResult.Path,
					Hash:      hashResult.Hash,
					Size:      hashResult.Size,
					ModTime:   time.Now(), // Use current time as approximation
					BackupSet: opts.BackupSet,
				}

				if err := op.db.UpsertFile(fileRecord); err != nil {
					op.logger.Error("update index failed", "path", hashResult.Path, "error", err)
					result.Errors++
					continue
				}

				result.FilesIndexed++

				// Report progress
				if opts.Progress != nil {
					opts.Progress(OperationProgress{
						Stage:         "indexing",
						FilesComplete: result.FilesIndexed,
						BytesComplete: result.BytesScanned,
						CurrentFile:   hashResult.Path,
					})
				}
			}
		}
	}()

	// Collect walk errors
	for err := range walkErrs {
		op.logger.Warn("walk error", "error", err)
		result.Errors++
	}

	// Wait for hash results processing to complete
	<-resultsChan

	result.Duration = time.Since(startTime)

	// Final progress update
	if opts.Progress != nil {
		opts.Progress(OperationProgress{
			Stage:         "complete",
			FilesComplete: result.FilesScanned,
			BytesComplete: result.BytesScanned,
			Percentage:    100,
		})
	}

	return result, nil
}
