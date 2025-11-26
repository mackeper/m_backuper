package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/mackeper/m_backuper/internal/backup"
	"github.com/mackeper/m_backuper/internal/hash"
	"github.com/mackeper/m_backuper/internal/index"
	"github.com/spf13/cobra"
)

var (
	scanUpdateIndex bool
	scanMinSize     int64
)

var scanCmd = &cobra.Command{
	Use:   "scan [path...]",
	Short: "Scan paths and update the file index",
	Long: `Scan one or more paths for files, calculate their hashes,
and update the file index database.

This is useful for:
  - Indexing backup drives before detecting duplicates
  - Adding new files to the index
  - Updating hashes for modified files

Files below the minimum size (default 1MB) are skipped for performance.`,
	Example: `  # Scan a backup drive
  m_backuper scan /mnt/external/backup

  # Scan multiple paths
  m_backuper scan /mnt/backup1 /mnt/backup2

  # Scan with custom minimum file size (10MB)
  m_backuper scan /mnt/backup --min-size 10485760`,
	Args: cobra.MinimumNArgs(1),
	RunE: runScan,
}

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().BoolVar(&scanUpdateIndex, "update-index", true, "update the file index database")
	scanCmd.Flags().Int64Var(&scanMinSize, "min-size", 0, "minimum file size to scan (bytes, 0 = use config default)")
}

func runScan(cmd *cobra.Command, args []string) error {
	paths := args

	// Use config min size if not specified
	if scanMinSize == 0 {
		scanMinSize = cfg.Duplicates.MinFileSize
	}

	log.Info("starting scan",
		"paths", paths,
		"min_size", scanMinSize,
		"update_index", scanUpdateIndex,
	)

	startTime := time.Now()

	// Create walker
	walker := backup.NewWalker(nil) // No exclusions for scan

	// Create hasher
	hasher := hash.NewCalculator(cfg.Concurrency.HashWorkers)

	ctx := context.Background()

	// Walk paths
	files, walkErrs := walker.WalkMultiple(ctx, paths)

	// Create hash jobs channel
	hashJobs := make(chan hash.HashJob)

	// Start hash workers
	hashResults := hasher.HashFiles(ctx, hashJobs)

	// Counters
	var filesScanned int64
	var bytesScanned int64
	var filesIndexed int64
	var errors int64

	// Process files
	go func() {
		defer close(hashJobs)

		for file := range files {
			// Skip files below minimum size
			if file.Size < scanMinSize {
				continue
			}

			log.Debug("scanning file", "path", file.Path, "size", file.Size)

			// Send to hasher
			select {
			case hashJobs <- hash.HashJob{Path: file.Path, Size: file.Size}:
				filesScanned++
				bytesScanned += file.Size
			case <-ctx.Done():
				return
			}
		}
	}()

	// Collect hash results
	go func() {
		for result := range hashResults {
			if result.Err != nil {
				log.Error("hash failed", "path", result.Path, "error", result.Err)
				errors++
				continue
			}

			log.Debug("hashed file", "path", result.Path, "hash", result.Hash[:16])

			if scanUpdateIndex {
				// Update index
				fileRecord := &index.FileRecord{
					Path: result.Path,
					Hash: result.Hash,
					Size: result.Size,
					ModTime: time.Now(), // Use current time as approximation
				}

				if err := db.UpsertFile(fileRecord); err != nil {
					log.Error("update index failed", "path", result.Path, "error", err)
					errors++
					continue
				}

				filesIndexed++
			}

			// Print progress every 100 files
			if filesScanned%100 == 0 {
				fmt.Printf("\rScanned: %d files (%s)", filesScanned, humanize.Bytes(uint64(bytesScanned)))
			}
		}
	}()

	// Collect walk errors
	for err := range walkErrs {
		log.Warn("walk error", "error", err)
		errors++
	}

	fmt.Println() // New line after progress

	duration := time.Since(startTime)

	// Print summary
	fmt.Println("\nScan Summary")
	fmt.Println("============")
	fmt.Printf("Files scanned: %d\n", filesScanned)
	fmt.Printf("Bytes scanned: %s\n", humanize.Bytes(uint64(bytesScanned)))
	if scanUpdateIndex {
		fmt.Printf("Files indexed: %d\n", filesIndexed)
	}
	fmt.Printf("Errors: %d\n", errors)
	fmt.Printf("Duration: %s\n", duration.Round(time.Second))

	if filesScanned > 0 {
		rate := float64(bytesScanned) / duration.Seconds()
		fmt.Printf("Scan rate: %s/s\n", humanize.Bytes(uint64(rate)))
	}

	return nil
}
