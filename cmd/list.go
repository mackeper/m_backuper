package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/dustin/go-humanize"
	"github.com/mackeper/m_backuper/internal/duplicate"
	"github.com/mackeper/m_backuper/internal/hash"
	"github.com/mackeper/m_backuper/internal/index"
	"github.com/spf13/cobra"
)

var (
	listSort      string
	listMinWasted int64
	listFormat    string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all duplicate file groups",
	Long: `List all duplicate file groups found in the index.

Duplicates are identified by matching SHA256 hashes. The output
shows groups of files that have identical content.

Results can be sorted by:
  - size: file size (largest first)
  - count: number of duplicates (most first)
  - wasted: wasted space (most first, default)

Output formats:
  - table: human-readable table (default)
  - json: machine-readable JSON`,
	Example: `  # List all duplicates
  m_backuper list

  # Sort by file count
  m_backuper list --sort count

  # Show only duplicates wasting at least 100MB
  m_backuper list --min-wasted 104857600

  # Output as JSON
  m_backuper list --format json`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVar(&listSort, "sort", "wasted", "sort order (size|count|wasted)")
	listCmd.Flags().Int64Var(&listMinWasted, "min-wasted", 0, "minimum wasted space in bytes")
	listCmd.Flags().StringVar(&listFormat, "format", "table", "output format (table|json)")
}

func runList(cmd *cobra.Command, args []string) error {
	log.Info("listing duplicates", "sort", listSort, "format", listFormat)

	// Create detector
	hasher := hash.NewCalculator(cfg.Concurrency.HashWorkers)
	detector := duplicate.NewDetector(db, hasher, cfg.Duplicates.MinFileSize, log)

	ctx := context.Background()

	// Find duplicates
	groups, err := detector.FindDuplicates(ctx)
	if err != nil {
		return fmt.Errorf("find duplicates: %w", err)
	}

	// Filter by minimum wasted space
	if listMinWasted > 0 {
		groups = duplicate.FilterByMinWasted(groups, listMinWasted)
	}

	// Sort
	switch listSort {
	case "size":
		duplicate.SortBySize(groups)
	case "count":
		duplicate.SortByCount(groups)
	case "wasted":
		duplicate.SortByWastedSpace(groups)
	default:
		return fmt.Errorf("invalid sort option: %s (use size, count, or wasted)", listSort)
	}

	// Output
	switch listFormat {
	case "table":
		printDuplicatesTable(groups)
	case "json":
		printDuplicatesJSON(groups)
	default:
		return fmt.Errorf("invalid format option: %s (use table or json)", listFormat)
	}

	return nil
}

func printDuplicatesTable(groups []index.DuplicateGroup) {
	if len(groups) == 0 {
		fmt.Println("No duplicates found")
		return
	}

	fmt.Println("\nDuplicate Groups")
	fmt.Println("================")

	var totalWasted int64
	for i, group := range groups {
		fmt.Printf("\n%d. Hash: %s\n", i+1, group.Hash[:16]+"...")
		fmt.Printf("   Files: %d | Size: %s | Wasted: %s\n",
			group.FileCount,
			humanize.Bytes(uint64(group.FileSize)),
			humanize.Bytes(uint64(group.WastedSpace)))

		for j, file := range group.Files {
			fmt.Printf("   [%d] %s\n", j+1, file.Path)
		}

		totalWasted += group.WastedSpace
	}

	fmt.Printf("\nSummary\n")
	fmt.Printf("=======\n")
	fmt.Printf("Duplicate groups: %d\n", len(groups))
	fmt.Printf("Total wasted space: %s\n", humanize.Bytes(uint64(totalWasted)))
}

func printDuplicatesJSON(groups []index.DuplicateGroup) {
	type jsonOutput struct {
		Groups []index.DuplicateGroup `json:"groups"`
		Summary struct {
			GroupCount   int    `json:"group_count"`
			TotalWasted  int64  `json:"total_wasted"`
		} `json:"summary"`
	}

	output := jsonOutput{
		Groups: groups,
	}

	output.Summary.GroupCount = len(groups)
	for _, group := range groups {
		output.Summary.TotalWasted += group.WastedSpace
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
	}
}
