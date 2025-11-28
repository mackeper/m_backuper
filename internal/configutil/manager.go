package configutil

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mackeper/m_backuper/internal/config"
	"gopkg.in/yaml.v3"
)

// Manager handles configuration file operations
type Manager struct {
	configPath string
}

// BackupSetInput represents input for creating/updating a backup set
type BackupSetInput struct {
	Name        string
	Sources     []string
	Destination string
	Excludes    []string
}

// BackupSetDisplay represents a backup set for display purposes
type BackupSetDisplay struct {
	Name        string
	Sources     []string
	Destination string
	Excludes    []string
}

// NewManager creates a new configuration manager
func NewManager(configPath string) (*Manager, error) {
	// If no config path provided, use default
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home directory: %w", err)
		}
		configPath = filepath.Join(home, ".m_backuper", "config.yaml")
	}

	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("create config directory: %w", err)
	}

	return &Manager{configPath: configPath}, nil
}

// AddBackupSet adds a new backup set to the configuration
func (m *Manager) AddBackupSet(input BackupSetInput) error {
	// Read existing config or create new one
	configData, err := m.readOrCreateConfig()
	if err != nil {
		return err
	}

	// Get backup_sets array
	backupSets, ok := configData["backup_sets"].([]interface{})
	if !ok {
		backupSets = []interface{}{}
	}

	// Check if backup set already exists
	for _, bs := range backupSets {
		bsMap, ok := bs.(map[string]interface{})
		if ok && bsMap["name"] == input.Name {
			return fmt.Errorf("backup set '%s' already exists", input.Name)
		}
	}

	// Create new backup set
	newBackupSet := map[string]interface{}{
		"name":        input.Name,
		"sources":     input.Sources,
		"destination": input.Destination,
	}

	if len(input.Excludes) > 0 {
		newBackupSet["excludes"] = input.Excludes
	}

	// Add to backup sets
	backupSets = append(backupSets, newBackupSet)
	configData["backup_sets"] = backupSets

	// Write back to file
	return m.writeConfig(configData)
}

// RemoveBackupSet removes a backup set from the configuration
func (m *Manager) RemoveBackupSet(name string) error {
	// Read existing config
	data, err := os.ReadFile(m.configPath)
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
	return m.writeConfig(configData)
}

// ListBackupSets returns all configured backup sets
func (m *Manager) ListBackupSets() ([]BackupSetDisplay, error) {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No config file yet
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var configData map[string]interface{}
	if err := yaml.Unmarshal(data, &configData); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	backupSets, ok := configData["backup_sets"].([]interface{})
	if !ok || len(backupSets) == 0 {
		return nil, nil
	}

	var result []BackupSetDisplay
	for _, bs := range backupSets {
		bsMap, ok := bs.(map[string]interface{})
		if !ok {
			continue
		}

		display := BackupSetDisplay{
			Name:        getString(bsMap, "name"),
			Destination: getString(bsMap, "destination"),
			Sources:     getStringSlice(bsMap, "sources"),
			Excludes:    getStringSlice(bsMap, "excludes"),
		}

		result = append(result, display)
	}

	return result, nil
}

// UpdateBackupSet updates an existing backup set
func (m *Manager) UpdateBackupSet(name string, input BackupSetInput) error {
	// Read existing config
	data, err := os.ReadFile(m.configPath)
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

	// Find and update the backup set
	found := false
	for i, bs := range backupSets {
		bsMap, ok := bs.(map[string]interface{})
		if ok && bsMap["name"] == name {
			found = true

			// Update fields
			updatedSet := map[string]interface{}{
				"name":        input.Name,
				"sources":     input.Sources,
				"destination": input.Destination,
			}

			if len(input.Excludes) > 0 {
				updatedSet["excludes"] = input.Excludes
			}

			backupSets[i] = updatedSet
			break
		}
	}

	if !found {
		return fmt.Errorf("backup set '%s' not found", name)
	}

	configData["backup_sets"] = backupSets

	// Write back to file
	return m.writeConfig(configData)
}

// InitConfig creates an example configuration file
func (m *Manager) InitConfig(overwrite bool) error {
	// Check if config already exists
	if _, err := os.Stat(m.configPath); err == nil && !overwrite {
		return fmt.Errorf("configuration file already exists at: %s", m.configPath)
	}

	// Ensure directory exists
	configDir := filepath.Dir(m.configPath)
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
	if err := os.WriteFile(m.configPath, []byte(exampleConfig), 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

// GetConfigPath returns the configuration file path
func (m *Manager) GetConfigPath() string {
	return m.configPath
}

// Helper functions

func (m *Manager) readOrCreateConfig() (map[string]interface{}, error) {
	if data, err := os.ReadFile(m.configPath); err == nil {
		var configData map[string]interface{}
		if err := yaml.Unmarshal(data, &configData); err != nil {
			return nil, fmt.Errorf("parse config: %w", err)
		}
		return configData, nil
	}

	// Initialize empty config structure
	return map[string]interface{}{
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
	}, nil
}

func (m *Manager) writeConfig(data map[string]interface{}) error {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(m.configPath, yamlData, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getStringSlice(m map[string]interface{}, key string) []string {
	var result []string
	if list, ok := m[key].([]interface{}); ok {
		for _, item := range list {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
	}
	return result
}

// LoadConfig loads and validates the configuration file
func LoadConfig(configPath string) (*config.Config, error) {
	return config.Load(configPath)
}
