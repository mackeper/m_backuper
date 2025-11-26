package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/mackeper/m_backuper/internal/duplicate"
	"github.com/mackeper/m_backuper/internal/hash"
	"github.com/spf13/cobra"
)

var (
	cleanAuto    bool
	cleanDryRun  bool
	cleanKeep    string
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Delete duplicate files",
	Long: `Delete duplicate files interactively or automatically.

By default, this command runs in interactive mode, asking for
confirmation before deleting each duplicate group.

Use --auto to automatically delete duplicates based on a keep strategy:
  - oldest: keep the file with the oldest modification time
  - newest: keep the file with the newest modification time
  - first: keep the first file found (alphabetically)
  - shortest: keep the file with the shortest path

Use --dry-run to preview what would be deleted without actually deleting.`,
	Example: `  # Interactive cleanup (confirms each group)
  m_backuper clean

  # Auto-delete keeping oldest files (dry run)
  m_backuper clean --auto --keep oldest --dry-run

  # Auto-delete keeping files with shortest paths
  m_backuper clean --auto --keep shortest`,
	RunE: runClean,
}

func init() {
	rootCmd.AddCommand(cleanCmd)
	cleanCmd.Flags().BoolVar(&cleanAuto, "auto", false, "automatically delete duplicates")
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "preview deletions without actually deleting")
	cleanCmd.Flags().StringVar(&cleanKeep, "keep", "first", "strategy for auto mode (oldest|newest|first|shortest)")
}

func runClean(cmd *cobra.Command, args []string) error {
	log.Info("starting cleanup", "auto", cleanAuto, "dry_run", cleanDryRun, "keep", cleanKeep)

	// Validate keep strategy
	var strategy duplicate.KeepStrategy
	switch cleanKeep {
	case "oldest":
		strategy = duplicate.KeepOldest
	case "newest":
		strategy = duplicate.KeepNewest
	case "first":
		strategy = duplicate.KeepFirst
	case "shortest":
		strategy = duplicate.KeepShortest
	default:
		return fmt.Errorf("invalid keep strategy: %s (use oldest, newest, first, or shortest)", cleanKeep)
	}

	// Create detector
	hasher := hash.NewCalculator(cfg.Concurrency.HashWorkers)
	detector := duplicate.NewDetector(db, hasher, cfg.Duplicates.MinFileSize, log)

	ctx := context.Background()

	// Find duplicates
	groups, err := detector.FindDuplicates(ctx)
	if err != nil {
		return fmt.Errorf("find duplicates: %w", err)
	}

	if len(groups) == 0 {
		fmt.Println("No duplicates found")
		return nil
	}

	// Create cleaner
	cleaner := duplicate.NewCleaner(db, log)

	// Process each group
	var totalDeleted int64
	var totalFreed int64

	for i, group := range groups {
		fmt.Printf("\nGroup %d/%d - Hash: %s\n", i+1, len(groups), group.Hash[:16]+"...")
		fmt.Printf("Files: %d | Size: %s | Wasted: %s\n",
			group.FileCount,
			humanize.Bytes(uint64(group.FileSize)),
			humanize.Bytes(uint64(group.WastedSpace)))

		for j, file := range group.Files {
			fmt.Printf("  [%d] %s\n", j+1, file.Path)
		}

		var toDelete []string

		if cleanAuto {
			// Auto mode: use strategy to select files to delete
			toDelete = duplicate.SelectFilesToDelete(group, strategy)
			fmt.Printf("\nAuto-delete strategy '%s' selected %d files to delete\n", cleanKeep, len(toDelete))
		} else {
			// Interactive mode: ask user
			fmt.Print("\nDelete duplicates for this group? (y/n/s=skip): ")
			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("read input: %w", err)
			}

			response = strings.TrimSpace(strings.ToLower(response))

			switch response {
			case "y", "yes":
				// Ask which strategy to use
				fmt.Printf("Which files to keep? (oldest/newest/first/shortest) [first]: ")
				strategyStr, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("read input: %w", err)
				}

				strategyStr = strings.TrimSpace(strings.ToLower(strategyStr))
				if strategyStr == "" {
					strategyStr = "first"
				}

				var userStrategy duplicate.KeepStrategy
				switch strategyStr {
				case "oldest":
					userStrategy = duplicate.KeepOldest
				case "newest":
					userStrategy = duplicate.KeepNewest
				case "first":
					userStrategy = duplicate.KeepFirst
				case "shortest":
					userStrategy = duplicate.KeepShortest
				default:
					fmt.Printf("Invalid strategy, using 'first'\n")
					userStrategy = duplicate.KeepFirst
				}

				toDelete = duplicate.SelectFilesToDelete(group, userStrategy)

			case "n", "no":
				fmt.Println("Skipping this group")
				continue

			case "s", "skip":
				fmt.Println("Skipping this group")
				continue

			default:
				fmt.Println("Invalid response, skipping this group")
				continue
			}
		}

		if len(toDelete) == 0 {
			continue
		}

		// Show what will be deleted
		fmt.Println("\nFiles to be deleted:")
		for _, path := range toDelete {
			fmt.Printf("  - %s\n", path)
		}

		if !cleanAuto && !cleanDryRun {
			// Final confirmation in interactive mode
			fmt.Print("\nConfirm deletion? (y/n): ")
			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("read input: %w", err)
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Println("Skipping deletion")
				continue
			}
		}

		// Delete files
		results, err := cleaner.DeleteFiles(toDelete, cleanDryRun)
		if err != nil {
			return fmt.Errorf("delete files: %w", err)
		}

		// Count successful deletions
		for _, result := range results {
			if result.Err == nil {
				if cleanDryRun {
					fmt.Printf("  Would delete: %s (%s)\n", result.Path, humanize.Bytes(uint64(result.Size)))
				} else if result.Deleted {
					fmt.Printf("  Deleted: %s (%s)\n", result.Path, humanize.Bytes(uint64(result.Size)))
				}
				totalDeleted++
				totalFreed += result.Size
			} else {
				fmt.Printf("  Error deleting %s: %v\n", result.Path, result.Err)
			}
		}
	}

	// Print summary
	fmt.Println("\nCleanup Summary")
	fmt.Println("===============")
	fmt.Printf("Files deleted: %d\n", totalDeleted)
	fmt.Printf("Space freed: %s\n", humanize.Bytes(uint64(totalFreed)))

	if cleanDryRun {
		fmt.Println("\n(Dry run - no files were actually deleted)")
	}

	return nil
}
