package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/mackeper/m_backuper/internal/operations"
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

	// Create scan operation
	scanOp := operations.NewScanOperation(db, log)

	ctx := context.Background()

	// Track progress for display
	var lastFilesScanned int64
	progressCallback := func(progress operations.OperationProgress) {
		// Print progress every 100 files
		if progress.FilesComplete-lastFilesScanned >= 100 || progress.Stage == "complete" {
			fmt.Printf("\rScanned: %d files (%s)", progress.FilesComplete, humanize.Bytes(uint64(progress.BytesComplete)))
			lastFilesScanned = progress.FilesComplete
		}
	}

	// Run scan operation
	result, err := scanOp.Run(ctx, operations.ScanOptions{
		Paths:       paths,
		MinSize:     scanMinSize,
		UpdateIndex: scanUpdateIndex,
		HashWorkers: cfg.Concurrency.HashWorkers,
		Progress:    progressCallback,
	})
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	fmt.Println() // New line after progress

	// Print summary
	fmt.Println("\nScan Summary")
	fmt.Println("============")
	fmt.Printf("Files scanned: %d\n", result.FilesScanned)
	fmt.Printf("Bytes scanned: %s\n", humanize.Bytes(uint64(result.BytesScanned)))
	if scanUpdateIndex {
		fmt.Printf("Files indexed: %d\n", result.FilesIndexed)
	}
	fmt.Printf("Errors: %d\n", result.Errors)
	fmt.Printf("Duration: %s\n", result.Duration.Round(time.Second))

	if result.FilesScanned > 0 {
		rate := float64(result.BytesScanned) / result.Duration.Seconds()
		fmt.Printf("Scan rate: %s/s\n", humanize.Bytes(uint64(rate)))
	}

	return nil
}
