package cmd

import (
	"context"
	"fmt"

	"github.com/mackeper/m_backuper/internal/display"
	"github.com/mackeper/m_backuper/internal/operations"
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

	// Create duplicate operation
	dupOp := operations.NewDuplicateOperation(db, cfg, log)

	ctx := context.Background()

	// Find duplicates with options
	groups, err := dupOp.FindDuplicates(ctx, operations.FindOptions{
		SortBy:    listSort,
		MinWasted: listMinWasted,
	})
	if err != nil {
		return err
	}

	// Format output
	formatter, err := display.NewFormatter(listFormat)
	if err != nil {
		return err
	}

	output, err := formatter.FormatDuplicateGroups(groups)
	if err != nil {
		return fmt.Errorf("format output: %w", err)
	}

	fmt.Println(output)
	return nil
}
