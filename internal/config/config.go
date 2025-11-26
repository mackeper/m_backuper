package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	BackupSets  []BackupSet       `mapstructure:"backup_sets"`
	Duplicates  DuplicateConfig   `mapstructure:"duplicates"`
	Concurrency ConcurrencyConfig `mapstructure:"concurrency"`
	Database    DatabaseConfig    `mapstructure:"database"`
}

// BackupSet defines a source-destination backup configuration
type BackupSet struct {
	Name        string   `mapstructure:"name"`
	Sources     []string `mapstructure:"sources"`
	Destination string   `mapstructure:"destination"`
	Excludes    []string `mapstructure:"excludes"`
}

// DuplicateConfig contains duplicate detection settings
type DuplicateConfig struct {
	MinFileSize int64    `mapstructure:"min_file_size"`
	ScanPaths   []string `mapstructure:"scan_paths"`
}

// ConcurrencyConfig contains performance tuning settings
type ConcurrencyConfig struct {
	HashWorkers int `mapstructure:"hash_workers"`
	CopyWorkers int `mapstructure:"copy_workers"`
}

// DatabaseConfig contains database settings
type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

// Load loads configuration from file or returns defaults
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set config file if provided
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Look for config in standard locations
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home directory: %w", err)
		}

		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(filepath.Join(home, ".m_backuper"))
		v.AddConfigPath(".")
	}

	// Set defaults
	setDefaults(v)

	// Read config file if it exists
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
		// Config file not found, use defaults
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	home, _ := os.UserHomeDir()

	// Database defaults
	v.SetDefault("database.path", filepath.Join(home, ".m_backuper", "index.db"))

	// Concurrency defaults
	v.SetDefault("concurrency.hash_workers", 0) // 0 = use NumCPU
	v.SetDefault("concurrency.copy_workers", 2)

	// Duplicate detection defaults
	v.SetDefault("duplicates.min_file_size", 1048576) // 1MB
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate backup sets
	for i, bs := range c.BackupSets {
		if bs.Name == "" {
			return fmt.Errorf("backup set %d: name is required", i)
		}
		if len(bs.Sources) == 0 {
			return fmt.Errorf("backup set %s: at least one source is required", bs.Name)
		}
		if bs.Destination == "" {
			return fmt.Errorf("backup set %s: destination is required", bs.Name)
		}

		// Validate paths exist
		for _, src := range bs.Sources {
			if _, err := os.Stat(src); err != nil {
				return fmt.Errorf("backup set %s: source path %s: %w", bs.Name, src, err)
			}
		}
	}

	// Validate concurrency settings
	if c.Concurrency.HashWorkers < 0 {
		return fmt.Errorf("hash_workers must be >= 0")
	}
	if c.Concurrency.CopyWorkers < 1 {
		return fmt.Errorf("copy_workers must be >= 1")
	}

	// Validate database path
	if c.Database.Path == "" {
		return fmt.Errorf("database path is required")
	}

	return nil
}

// GetBackupSet returns a backup set by name
func (c *Config) GetBackupSet(name string) *BackupSet {
	for i := range c.BackupSets {
		if c.BackupSets[i].Name == name {
			return &c.BackupSets[i]
		}
	}
	return nil
}
