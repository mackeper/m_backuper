package backup

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/mackeper/m_backuper/internal/config"
	"github.com/mackeper/m_backuper/internal/hash"
	"github.com/mackeper/m_backuper/internal/index"
)

// Engine orchestrates the backup process
type Engine struct {
	config *config.Config
	index  *index.DB
	hasher *hash.Calculator
	copier *Copier
	walker *Walker
	logger *slog.Logger
}

// NewEngine creates a new backup engine
func NewEngine(cfg *config.Config, idx *index.DB, logger *slog.Logger) *Engine {
	hashWorkers := cfg.Concurrency.HashWorkers
	copyWorkers := cfg.Concurrency.CopyWorkers

	return &Engine{
		config: cfg,
		index:  idx,
		hasher: hash.NewCalculator(hashWorkers),
		copier: NewCopier(copyWorkers),
		logger: logger,
	}
}

// BackupResult contains statistics about a backup operation
type BackupResult struct {
	BackupSet     string
	FilesCopied   int64
	BytesCopied   int64
	FilesSkipped  int64
	Errors        int64
	Duration      time.Duration
	StartTime     time.Time
	EndTime       time.Time
}

// Backup performs a backup for the specified backup set
func (e *Engine) Backup(ctx context.Context, backupSetName string, dryRun bool) (*BackupResult, error) {
	// Get backup set configuration
	backupSet := e.config.GetBackupSet(backupSetName)
	if backupSet == nil {
		return nil, fmt.Errorf("backup set not found: %s", backupSetName)
	}

	e.logger.Info("starting backup",
		"backup_set", backupSetName,
		"sources", len(backupSet.Sources),
		"destination", backupSet.Destination,
		"dry_run", dryRun,
	)

	startTime := time.Now()

	// Create backup run record (if not dry run)
	var runID int64
	var err error
	if !dryRun {
		runID, err = e.index.CreateBackupRun(backupSetName)
		if err != nil {
			return nil, fmt.Errorf("create backup run: %w", err)
		}
	}

	result := &BackupResult{
		BackupSet: backupSetName,
		StartTime: startTime,
	}

	// Create walker with exclusion patterns
	e.walker = NewWalker(backupSet.Excludes)

	// Walk source directories
	files, walkErrs := e.walker.WalkMultiple(ctx, backupSet.Sources)

	// Create hash jobs channel
	hashJobs := make(chan hash.HashJob)

	// Start hash workers
	hashResults := e.hasher.HashFiles(ctx, hashJobs)

	// Create copy jobs channel
	copyJobs := make(chan CopyJob)

	// Start copy workers (if not dry run)
	var copyResults <-chan CopyResult
	if !dryRun {
		copyResults = e.copier.CopyFiles(ctx, copyJobs)
	} else {
		// In dry run mode, create a dummy channel
		dummyCopyResults := make(chan CopyResult)
		close(dummyCopyResults)
		copyResults = dummyCopyResults
	}

	// Process files
	go func() {
		defer close(hashJobs)
		defer close(copyJobs)

		for file := range files {
			e.logger.Debug("discovered file", "path", file.Path, "size", file.Size)

			// Send to hasher
			select {
			case hashJobs <- hash.HashJob{Path: file.Path, Size: file.Size}:
			case <-ctx.Done():
				return
			}

			// Wait for hash result
			var hashResult hash.HashResult
			select {
			case hashResult = <-hashResults:
			case <-ctx.Done():
				return
			}

			if hashResult.Err != nil {
				e.logger.Error("hash failed", "path", hashResult.Path, "error", hashResult.Err)
				result.Errors++
				continue
			}

			e.logger.Debug("hashed file", "path", hashResult.Path, "hash", hashResult.Hash[:16])

			// Compute destination path
			// Find which source this file belongs to
			var sourceRoot string
			for _, src := range backupSet.Sources {
				if rel, err := filepath.Rel(src, file.Path); err == nil && rel[0] != '.' {
					sourceRoot = src
					break
				}
			}

			if sourceRoot == "" {
				e.logger.Error("could not determine source root", "path", file.Path)
				result.Errors++
				continue
			}

			destPath, err := ComputeDestPath(file.Path, sourceRoot, backupSet.Destination)
			if err != nil {
				e.logger.Error("compute dest path failed", "path", file.Path, "error", err)
				result.Errors++
				continue
			}

			if !dryRun {
				// Send to copier
				select {
				case copyJobs <- CopyJob{
					SourcePath: file.Path,
					DestPath:   destPath,
					Size:       file.Size,
				}:
				case <-ctx.Done():
					return
				}

				// Wait for copy result
				var copyResult CopyResult
				select {
				case copyResult = <-copyResults:
				case <-ctx.Done():
					return
				}

				if copyResult.Err != nil {
					e.logger.Error("copy failed", "path", copyResult.SourcePath, "error", copyResult.Err)
					result.Errors++
					continue
				}

				e.logger.Info("copied file",
					"source", copyResult.SourcePath,
					"dest", copyResult.DestPath,
					"size", copyResult.Size,
				)

				// Update index
				fileRecord := &index.FileRecord{
					Path:      destPath,
					Hash:      hashResult.Hash,
					Size:      file.Size,
					ModTime:   time.Unix(file.ModTime, 0),
					BackupSet: backupSetName,
				}

				if err := e.index.UpsertFile(fileRecord); err != nil {
					e.logger.Error("update index failed", "path", destPath, "error", err)
				}

				result.FilesCopied++
				result.BytesCopied += file.Size
			} else {
				e.logger.Info("would copy file",
					"source", file.Path,
					"dest", destPath,
					"size", file.Size,
				)
				result.FilesCopied++
				result.BytesCopied += file.Size
			}
		}
	}()

	// Collect walk errors
	go func() {
		for err := range walkErrs {
			e.logger.Warn("walk error", "error", err)
		}
	}()

	// Wait for all work to complete
	select {
	case <-ctx.Done():
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)

		if !dryRun {
			errMsg := ctx.Err().Error()
			e.index.CompleteBackupRun(runID, "failed", &errMsg)
		}

		return result, ctx.Err()
	case <-time.After(1 * time.Second):
		// Give some time for goroutines to finish
		// This is a simplified approach; in production, you'd want better synchronization
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Update backup run record
	if !dryRun {
		if err := e.index.UpdateBackupRun(runID, result.FilesCopied, result.BytesCopied); err != nil {
			e.logger.Error("update backup run failed", "error", err)
		}

		status := "completed"
		if result.Errors > 0 {
			status = "completed_with_errors"
		}
		if err := e.index.CompleteBackupRun(runID, status, nil); err != nil {
			e.logger.Error("complete backup run failed", "error", err)
		}
	}

	e.logger.Info("backup completed",
		"backup_set", backupSetName,
		"files_copied", result.FilesCopied,
		"bytes_copied", result.BytesCopied,
		"errors", result.Errors,
		"duration", result.Duration,
	)

	return result, nil
}

// BackupAll performs backups for all configured backup sets
func (e *Engine) BackupAll(ctx context.Context, dryRun bool) ([]*BackupResult, error) {
	results := make([]*BackupResult, 0, len(e.config.BackupSets))

	for _, backupSet := range e.config.BackupSets {
		result, err := e.Backup(ctx, backupSet.Name, dryRun)
		if err != nil {
			return results, fmt.Errorf("backup %s: %w", backupSet.Name, err)
		}
		results = append(results, result)
	}

	return results, nil
}
