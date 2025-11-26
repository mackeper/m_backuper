package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/mackeper/m_backuper/internal/backup"
	"github.com/spf13/cobra"
)

var (
	backupAll    bool
	backupDryRun bool
)

var backupCmd = &cobra.Command{
	Use:   "backup [name]",
	Short: "Run a backup operation",
	Long: `Run a backup operation for the specified backup set.
Use --all to backup all configured backup sets.

The backup process will:
  1. Walk the source directories
  2. Calculate SHA256 hashes for each file
  3. Copy files to the destination (preserving directory structure)
  4. Update the file index database

Use --dry-run to preview what would be backed up without actually copying files.`,
	Example: `  # Backup a specific backup set
  m_backuper backup photos

  # Backup all configured sets
  m_backuper backup --all

  # Preview a backup without copying
  m_backuper backup photos --dry-run`,
	RunE: runBackup,
}

func init() {
	rootCmd.AddCommand(backupCmd)
	backupCmd.Flags().BoolVar(&backupAll, "all", false, "backup all configured backup sets")
	backupCmd.Flags().BoolVar(&backupDryRun, "dry-run", false, "preview backup without copying files")
}

func runBackup(cmd *cobra.Command, args []string) error {
	// Validate arguments
	if !backupAll && len(args) == 0 {
		return fmt.Errorf("backup set name required (or use --all)")
	}

	if backupAll && len(args) > 0 {
		return fmt.Errorf("cannot specify backup set name with --all flag")
	}

	// Create backup engine
	engine := backup.NewEngine(cfg, db, log)

	ctx := context.Background()

	if backupAll {
		// Backup all sets
		log.Info("backing up all configured backup sets")
		results, err := engine.BackupAll(ctx, backupDryRun)
		if err != nil {
			return fmt.Errorf("backup all: %w", err)
		}

		// Print summary
		fmt.Println("\nBackup Summary")
		fmt.Println("==============")
		var totalFiles, totalBytes int64
		for _, result := range results {
			fmt.Printf("\n%s:\n", result.BackupSet)
			fmt.Printf("  Files copied: %d\n", result.FilesCopied)
			fmt.Printf("  Bytes copied: %s\n", humanize.Bytes(uint64(result.BytesCopied)))
			fmt.Printf("  Errors: %d\n", result.Errors)
			fmt.Printf("  Duration: %s\n", result.Duration.Round(time.Second))

			totalFiles += result.FilesCopied
			totalBytes += result.BytesCopied
		}

		fmt.Printf("\nTotal:\n")
		fmt.Printf("  Files: %d\n", totalFiles)
		fmt.Printf("  Size: %s\n", humanize.Bytes(uint64(totalBytes)))

	} else {
		// Backup single set
		backupSetName := args[0]
		log.Info("starting backup", "backup_set", backupSetName)

		result, err := engine.Backup(ctx, backupSetName, backupDryRun)
		if err != nil {
			return fmt.Errorf("backup: %w", err)
		}

		// Print summary
		fmt.Println("\nBackup Summary")
		fmt.Println("==============")
		fmt.Printf("Backup set: %s\n", result.BackupSet)
		fmt.Printf("Files copied: %d\n", result.FilesCopied)
		fmt.Printf("Bytes copied: %s\n", humanize.Bytes(uint64(result.BytesCopied)))
		fmt.Printf("Errors: %d\n", result.Errors)
		fmt.Printf("Duration: %s\n", result.Duration.Round(time.Second))

		if backupDryRun {
			fmt.Println("\n(Dry run - no files were actually copied)")
		}
	}

	return nil
}
