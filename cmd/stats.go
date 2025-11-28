package cmd

import (
	"fmt"

	"github.com/mackeper/m_backuper/internal/display"
	"github.com/mackeper/m_backuper/internal/stats"
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
	// Create calculator
	calculator := stats.NewCalculator(db, cfg.Database.Path)

	// Calculate statistics
	dbStats, err := calculator.Calculate()
	if err != nil {
		return fmt.Errorf("calculate stats: %w", err)
	}

	// Check if database is empty
	if dbStats.TotalFiles == 0 {
		fmt.Println("No files indexed")
		fmt.Println("\nTo index files, run:")
		fmt.Println("  m_backuper scan /path/to/directory")
		fmt.Println("  m_backuper backup <name>")
		return nil
	}

	// Format and print output
	formatter := &display.TableFormatter{}
	output, err := formatter.FormatStats(dbStats)
	if err != nil {
		return fmt.Errorf("format stats: %w", err)
	}

	fmt.Println(output)
	return nil
}
