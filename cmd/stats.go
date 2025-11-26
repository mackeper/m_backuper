package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show database statistics",
	Long: `Display statistics about indexed files in the database.

Shows total files, size, breakdown by backup set, and detected root directories.`,
	Example: `  # Show database stats
  m_backuper stats`,
	RunE: runStats,
}

func init() {
	rootCmd.AddCommand(statsCmd)
}

func runStats(cmd *cobra.Command, args []string) error {
	// Get all files from database
	files, err := db.GetAllFiles()
	if err != nil {
		return fmt.Errorf("get files: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No files indexed")
		fmt.Println("\nTo index files, run:")
		fmt.Println("  m_backuper scan /path/to/directory")
		fmt.Println("  m_backuper backup <name>")
		return nil
	}

	// Calculate statistics
	var totalSize int64
	backupSets := make(map[string]struct {
		count int64
		size  int64
	})
	rootDirs := make(map[string]int)

	for _, file := range files {
		totalSize += file.Size

		// Track by backup set
		setName := file.BackupSet
		if setName == "" {
			setName = "(no backup set)"
		}
		stats := backupSets[setName]
		stats.count++
		stats.size += file.Size
		backupSets[setName] = stats

		// Detect root directories (top 2 levels of path)
		parts := strings.Split(filepath.Clean(file.Path), string(filepath.Separator))
		if len(parts) >= 3 {
			root := filepath.Join(string(filepath.Separator), parts[1], parts[2])
			rootDirs[root]++
		} else if len(parts) >= 2 {
			root := filepath.Join(string(filepath.Separator), parts[1])
			rootDirs[root]++
		}
	}

	// Get database file size
	dbInfo, err := os.Stat(cfg.Database.Path)
	var dbSize int64
	if err == nil {
		dbSize = dbInfo.Size()
	}

	// Print statistics
	fmt.Println("\nDatabase Statistics")
	fmt.Println("===================")
	fmt.Printf("Database: %s\n", cfg.Database.Path)
	if dbSize > 0 {
		fmt.Printf("Database size: %s\n", humanize.Bytes(uint64(dbSize)))
	}
	fmt.Println()

	fmt.Printf("Total files: %s\n", humanize.Comma(int64(len(files))))
	fmt.Printf("Total size: %s\n", humanize.Bytes(uint64(totalSize)))
	fmt.Println()

	// Backup sets breakdown
	fmt.Println("Files by Backup Set:")
	fmt.Println("--------------------")
	for setName, stats := range backupSets {
		fmt.Printf("  %-30s %8s files  %12s\n",
			setName,
			humanize.Comma(stats.count),
			humanize.Bytes(uint64(stats.size)))
	}
	fmt.Println()

	// Root directories
	fmt.Println("Top Root Directories:")
	fmt.Println("---------------------")

	// Sort root dirs by count
	type rootDirStat struct {
		path  string
		count int
	}
	var sortedRoots []rootDirStat
	for path, count := range rootDirs {
		sortedRoots = append(sortedRoots, rootDirStat{path, count})
	}

	// Simple bubble sort by count (descending)
	for i := 0; i < len(sortedRoots); i++ {
		for j := i + 1; j < len(sortedRoots); j++ {
			if sortedRoots[j].count > sortedRoots[i].count {
				sortedRoots[i], sortedRoots[j] = sortedRoots[j], sortedRoots[i]
			}
		}
	}

	// Show top 10 root directories
	maxShow := 10
	if len(sortedRoots) < maxShow {
		maxShow = len(sortedRoots)
	}

	for i := 0; i < maxShow; i++ {
		fmt.Printf("  %-50s %8s files\n",
			sortedRoots[i].path,
			humanize.Comma(int64(sortedRoots[i].count)))
	}

	if len(sortedRoots) > maxShow {
		fmt.Printf("  ... and %d more\n", len(sortedRoots)-maxShow)
	}

	return nil
}
