package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/mackeper/m_backuper/internal/config"
	"github.com/mackeper/m_backuper/internal/index"
	"github.com/mackeper/m_backuper/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
	cfg     *config.Config
	db      *index.DB
	log     *slog.Logger
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "m_backuper",
	Short: "A backup tool with duplicate detection",
	Long: `m_backuper is a CLI and TUI tool for backing up files and detecting
duplicates across backup drives using SHA256 content hashing.

Features:
  - Full copy backups with configurable sources and destinations
  - SHA256-based duplicate detection across different paths
  - Interactive TUI for managing duplicates
  - CLI commands for automation and scripting`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize logger
		log = logger.New(verbose)

		// Skip config loading for config management commands
		skipConfigLoad := false
		if cmd.Parent() != nil && cmd.Parent().Name() == "config" {
			// Skip for: init, add, remove, list (they handle config loading themselves)
			if cmd.Name() == "init" || cmd.Name() == "add" || cmd.Name() == "remove" || cmd.Name() == "list" {
				skipConfigLoad = true
			}
		}

		if !skipConfigLoad {
			var err error
			cfg, err = config.Load(cfgFile)
			if err != nil {
				// For some commands, missing config is not fatal
				if cmd.Name() == "config" || cmd.Name() == "list" || cmd.Name() == "show" {
					log.Warn("no configuration file found, using defaults")
					cfg = &config.Config{}
				} else {
					return fmt.Errorf("load config: %w", err)
				}
			}

			// Open database
			db, err = index.Open(cfg.Database.Path)
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}
		}

		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		// Close database if opened
		if db != nil {
			return db.Close()
		}
		return nil
	},
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.m_backuper/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}
