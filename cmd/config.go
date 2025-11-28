package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/mackeper/m_backuper/internal/configutil"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
	Long:  `Manage m_backuper configuration files.`,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create an example configuration file",
	Long: `Create an example configuration file at ~/.m_backuper/config.yaml

This command creates a starter configuration file with example backup sets
and sensible defaults. You can then edit this file to match your needs.`,
	RunE: runConfigInit,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration",
	Long:  `Display the current configuration loaded from the config file.`,
	RunE: runConfigShow,
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration file",
	Long:  `Validate the configuration file for errors.`,
	RunE: runConfigValidate,
}

var (
	addSources      []string
	addDestination  string
	addExcludes     []string
)

var configAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a backup set to configuration",
	Long: `Add a new backup set to the configuration file.

This command modifies the configuration file to add a new backup set with
the specified sources, destination, and optional exclusion patterns.`,
	Example: `  # Add a backup set
  m_backuper config add photos --sources ~/Pictures --destination /mnt/backup/photos

  # Add with multiple sources and excludes
  m_backuper config add docs \
    --sources ~/Documents,~/Downloads \
    --destination /mnt/backup/docs \
    --excludes "*.tmp,*.bak"`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigAdd,
}

var configRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a backup set from configuration",
	Long:  `Remove a backup set from the configuration file.`,
	Example: `  # Remove a backup set
  m_backuper config remove photos`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigRemove,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List backup sets",
	Long:  `List all configured backup sets.`,
	RunE: runConfigList,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configAddCmd)
	configCmd.AddCommand(configRemoveCmd)
	configCmd.AddCommand(configListCmd)

	// Flags for add command
	configAddCmd.Flags().StringSliceVar(&addSources, "sources", nil, "source directories (comma-separated)")
	configAddCmd.Flags().StringVar(&addDestination, "destination", "", "destination directory")
	configAddCmd.Flags().StringSliceVar(&addExcludes, "excludes", nil, "exclusion patterns (comma-separated)")
	configAddCmd.MarkFlagRequired("sources")
	configAddCmd.MarkFlagRequired("destination")
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	// Create manager
	manager, err := configutil.NewManager(cfgFile)
	if err != nil {
		return err
	}

	configPath := manager.GetConfigPath()

	// Check if config already exists
	overwrite := false
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Configuration file already exists at: %s\n", configPath)
		fmt.Print("Overwrite? (y/n): ")

		var response string
		fmt.Scanln(&response)

		if response != "y" && response != "yes" {
			fmt.Println("Aborted")
			return nil
		}
		overwrite = true
	}

	// Initialize config
	if err := manager.InitConfig(overwrite); err != nil {
		return err
	}

	fmt.Printf("Configuration file created at: %s\n", configPath)
	fmt.Println("\nNext steps:")
	fmt.Println("1. Edit the configuration file to match your backup sources and destinations")
	fmt.Println("2. Run 'm_backuper backup --dry-run <name>' to preview a backup")
	fmt.Println("3. Run 'm_backuper backup <name>' to perform the backup")

	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	if cfg == nil {
		return fmt.Errorf("no configuration loaded")
	}

	fmt.Println("Current Configuration")
	fmt.Println("====================")
	fmt.Println()

	fmt.Println("Backup Sets:")
	for i, bs := range cfg.BackupSets {
		fmt.Printf("\n%d. %s\n", i+1, bs.Name)
		fmt.Printf("   Sources:\n")
		for _, src := range bs.Sources {
			fmt.Printf("     - %s\n", src)
		}
		fmt.Printf("   Destination: %s\n", bs.Destination)
		if len(bs.Excludes) > 0 {
			fmt.Printf("   Excludes:\n")
			for _, exc := range bs.Excludes {
				fmt.Printf("     - %s\n", exc)
			}
		}
	}

	fmt.Println("\nDuplicates:")
	fmt.Printf("  Min file size: %d bytes\n", cfg.Duplicates.MinFileSize)
	if len(cfg.Duplicates.ScanPaths) > 0 {
		fmt.Printf("  Scan paths:\n")
		for _, path := range cfg.Duplicates.ScanPaths {
			fmt.Printf("    - %s\n", path)
		}
	}

	fmt.Println("\nConcurrency:")
	fmt.Printf("  Hash workers: %d\n", cfg.Concurrency.HashWorkers)
	fmt.Printf("  Copy workers: %d\n", cfg.Concurrency.CopyWorkers)

	fmt.Println("\nDatabase:")
	fmt.Printf("  Path: %s\n", cfg.Database.Path)

	return nil
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	if cfg == nil {
		return fmt.Errorf("no configuration loaded")
	}

	if err := cfg.Validate(); err != nil {
		fmt.Printf("❌ Configuration is invalid: %v\n", err)
		return err
	}

	fmt.Println("✅ Configuration is valid")
	fmt.Printf("   Backup sets: %d\n", len(cfg.BackupSets))
	fmt.Printf("   Database: %s\n", cfg.Database.Path)

	return nil
}

func runConfigAdd(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Create manager
	manager, err := configutil.NewManager(cfgFile)
	if err != nil {
		return err
	}

	// Add backup set
	input := configutil.BackupSetInput{
		Name:        name,
		Sources:     addSources,
		Destination: addDestination,
		Excludes:    addExcludes,
	}

	if err := manager.AddBackupSet(input); err != nil {
		return err
	}

	fmt.Printf("✅ Added backup set '%s'\n", name)
	fmt.Printf("   Sources: %s\n", strings.Join(addSources, ", "))
	fmt.Printf("   Destination: %s\n", addDestination)
	if len(addExcludes) > 0 {
		fmt.Printf("   Excludes: %s\n", strings.Join(addExcludes, ", "))
	}

	return nil
}

func runConfigRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Create manager
	manager, err := configutil.NewManager(cfgFile)
	if err != nil {
		return err
	}

	// Remove backup set
	if err := manager.RemoveBackupSet(name); err != nil {
		return err
	}

	fmt.Printf("✅ Removed backup set '%s'\n", name)

	return nil
}

func runConfigList(cmd *cobra.Command, args []string) error {
	// Create manager
	manager, err := configutil.NewManager(cfgFile)
	if err != nil {
		return err
	}

	// List backup sets
	backupSets, err := manager.ListBackupSets()
	if err != nil {
		return err
	}

	if len(backupSets) == 0 {
		fmt.Println("No backup sets configured")
		fmt.Println("\nTo add a backup set, run:")
		fmt.Println("  m_backuper config add <name> --sources <paths> --destination <path>")
		return nil
	}

	fmt.Println("Backup Sets:")
	fmt.Println("============")
	for i, bs := range backupSets {
		fmt.Printf("\n%d. %s\n", i+1, bs.Name)
		if len(bs.Sources) > 0 {
			fmt.Printf("   Sources: %s\n", strings.Join(bs.Sources, ", "))
		}
		fmt.Printf("   Destination: %s\n", bs.Destination)
		if len(bs.Excludes) > 0 {
			fmt.Printf("   Excludes: %s\n", strings.Join(bs.Excludes, ", "))
		}
	}

	return nil
}
