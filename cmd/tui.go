package cmd

import (
	"fmt"

	"github.com/mackeper/m_backuper/internal/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive TUI",
	Long: `Launch the interactive terminal user interface (TUI) for managing backups.

The TUI provides:
  - Config Manager: Add, edit, and remove backup sets
  - Statistics: View database statistics and backup info
  - Duplicate Browser: Browse and clean duplicate files
  - Backup Runner: Run backups with real-time progress

Navigation:
  - Use arrow keys or j/k to navigate
  - Press enter to select
  - Press esc or q to go back
  - Press ctrl+c to quit`,
	Example: `  # Launch the TUI
  m_backuper tui`,
	RunE: runTUI,
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

func runTUI(cmd *cobra.Command, args []string) error {
	log.Info("starting TUI")

	if err := tui.Run(cfg, cfgFile, db, log); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}
