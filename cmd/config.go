package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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
	// Determine config path
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".m_backuper")
	configPath := filepath.Join(configDir, "config.yaml")

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Configuration file already exists at: %s\n", configPath)
		fmt.Print("Overwrite? (y/n): ")

		var response string
		fmt.Scanln(&response)

		if response != "y" && response != "yes" {
			fmt.Println("Aborted")
			return nil
		}
	}

	// Create config directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	// Example configuration content
	exampleConfig := `# m_backuper configuration file

# Backup sets define source-destination pairs for backups
backup_sets:
  - name: documents
    sources:
      - ~/Documents
      - ~/Downloads
    destination: /mnt/external/backup/documents
    excludes:
      - "*.tmp"
      - "*.bak"
      - ".cache/**"

  # Add more backup sets as needed
  # - name: photos
  #   sources:
  #     - ~/Pictures
  #     - ~/DCIM
  #   destination: /mnt/external/backup/photos
  #   excludes:
  #     - ".thumbnails/**"

# Duplicate detection settings
duplicates:
  # Minimum file size to consider for duplicate detection (in bytes)
  # Files smaller than this are skipped for performance
  min_file_size: 1048576  # 1MB

  # Paths to scan for duplicates
  scan_paths:
    - /mnt/external/backup

# Performance tuning
concurrency:
  # Number of parallel workers for hash calculation
  # 0 = use number of CPU cores (recommended)
  hash_workers: 0

  # Number of parallel workers for file copying
  # Usually 2-4 is optimal due to I/O limits
  copy_workers: 2

# Database settings
database:
  # Path to SQLite database file
  path: ~/.m_backuper/index.db
`

	// Write config file
	if err := os.WriteFile(configPath, []byte(exampleConfig), 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
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

	// Get config file path
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	// Read existing config or create new one
	var configData map[string]interface{}

	if data, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(data, &configData); err != nil {
			return fmt.Errorf("parse config: %w", err)
		}
	} else {
		// Initialize empty config structure
		configData = map[string]interface{}{
			"backup_sets": []interface{}{},
			"duplicates": map[string]interface{}{
				"min_file_size": 1048576,
				"scan_paths":    []string{},
			},
			"concurrency": map[string]interface{}{
				"hash_workers": 0,
				"copy_workers": 2,
			},
			"database": map[string]interface{}{
				"path": "~/.m_backuper/index.db",
			},
		}
	}

	// Get backup_sets array
	backupSets, ok := configData["backup_sets"].([]interface{})
	if !ok {
		backupSets = []interface{}{}
	}

	// Check if backup set already exists
	for _, bs := range backupSets {
		bsMap, ok := bs.(map[string]interface{})
		if ok && bsMap["name"] == name {
			return fmt.Errorf("backup set '%s' already exists", name)
		}
	}

	// Create new backup set
	newBackupSet := map[string]interface{}{
		"name":        name,
		"sources":     addSources,
		"destination": addDestination,
	}

	if len(addExcludes) > 0 {
		newBackupSet["excludes"] = addExcludes
	}

	// Add to backup sets
	backupSets = append(backupSets, newBackupSet)
	configData["backup_sets"] = backupSets

	// Write back to file
	if err := writeConfigFile(configPath, configData); err != nil {
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

	// Get config file path
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	// Read existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	var configData map[string]interface{}
	if err := yaml.Unmarshal(data, &configData); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	// Get backup_sets array
	backupSets, ok := configData["backup_sets"].([]interface{})
	if !ok {
		return fmt.Errorf("no backup sets found in config")
	}

	// Find and remove the backup set
	found := false
	newBackupSets := make([]interface{}, 0, len(backupSets))

	for _, bs := range backupSets {
		bsMap, ok := bs.(map[string]interface{})
		if ok && bsMap["name"] == name {
			found = true
			continue // Skip this one (remove it)
		}
		newBackupSets = append(newBackupSets, bs)
	}

	if !found {
		return fmt.Errorf("backup set '%s' not found", name)
	}

	configData["backup_sets"] = newBackupSets

	// Write back to file
	if err := writeConfigFile(configPath, configData); err != nil {
		return err
	}

	fmt.Printf("✅ Removed backup set '%s'\n", name)

	return nil
}

func runConfigList(cmd *cobra.Command, args []string) error {
	// Load config file without validation
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Println("No backup sets configured")
		fmt.Println("\nTo add a backup set, run:")
		fmt.Println("  m_backuper config add <name> --sources <paths> --destination <path>")
		return nil
	}

	var configData map[string]interface{}
	if err := yaml.Unmarshal(data, &configData); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	backupSets, ok := configData["backup_sets"].([]interface{})
	if !ok || len(backupSets) == 0 {
		fmt.Println("No backup sets configured")
		fmt.Println("\nTo add a backup set, run:")
		fmt.Println("  m_backuper config add <name> --sources <paths> --destination <path>")
		return nil
	}

	fmt.Println("Backup Sets:")
	fmt.Println("============")
	for i, bs := range backupSets {
		bsMap, ok := bs.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := bsMap["name"].(string)
		destination, _ := bsMap["destination"].(string)

		var sources []string
		if srcList, ok := bsMap["sources"].([]interface{}); ok {
			for _, src := range srcList {
				if srcStr, ok := src.(string); ok {
					sources = append(sources, srcStr)
				}
			}
		}

		var excludes []string
		if excList, ok := bsMap["excludes"].([]interface{}); ok {
			for _, exc := range excList {
				if excStr, ok := exc.(string); ok {
					excludes = append(excludes, excStr)
				}
			}
		}

		fmt.Printf("\n%d. %s\n", i+1, name)
		if len(sources) > 0 {
			fmt.Printf("   Sources: %s\n", strings.Join(sources, ", "))
		}
		fmt.Printf("   Destination: %s\n", destination)
		if len(excludes) > 0 {
			fmt.Printf("   Excludes: %s\n", strings.Join(excludes, ", "))
		}
	}

	return nil
}

// Helper functions

func getConfigPath() (string, error) {
	if cfgFile != "" {
		return cfgFile, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".m_backuper")
	configPath := filepath.Join(configDir, "config.yaml")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("create config directory: %w", err)
	}

	return configPath, nil
}

func writeConfigFile(path string, data map[string]interface{}) error {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, yamlData, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}
