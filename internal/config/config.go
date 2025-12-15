package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

//nolint:govet // fieldalignment: field order optimized for JSON readability
type Config struct {
	BackupRoot            string   `json:"backup_root"`
	DeviceID              string   `json:"device_id"`
	PathsToBackup         []string `json:"paths_to_backup"`
	FilesToIgnorePatterns []string `json:"files_to_ignore_patterns"`
	SMBUser               string   `json:"smb_user,omitempty"`
	SMBPassword           string   `json:"smb_password,omitempty"`
}

func Default() Config {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown-device"
	}

	return Config{
		BackupRoot:            "//192.168.1.100/backups/m_backuper",
		DeviceID:              hostname,
		PathsToBackup:         []string{},
		FilesToIgnorePatterns: []string{"*.tmp", ".cache/*"},
	}
}

func ConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "m_backuper", "config.json"), nil
}

func Load() (Config, error) {
	configPath, err := ConfigPath()
	if err != nil {
		return Default(), err
	}
	return LoadFrom(configPath)
}

func LoadFrom(configPath string) (Config, error) {
	cfg := Default()

	// Try to load from file
	data, err := os.ReadFile(configPath) //nolint:gosec // Config path is from trusted source
	if err != nil {
		if os.IsNotExist(err) {
			slog.Info("config file not found, using defaults", "path", configPath)
		} else {
			slog.Error("failed to read config file", "path", configPath, "error", err)
			return cfg, fmt.Errorf("failed to read config file: %w", err)
		}
	} else {
		if err := json.Unmarshal(data, &cfg); err != nil {
			slog.Error("failed to parse config file", "path", configPath, "error", err)
			return cfg, fmt.Errorf("invalid JSON in config file: %w", err)
		}
		slog.Info("loaded config from file", "path", configPath)
	}

	// Apply environment variable overrides
	if v := os.Getenv("M_BACKUPER_SMB_USER"); v != "" {
		cfg.SMBUser = v
		slog.Debug("overriding SMB user from environment")
	}
	if v := os.Getenv("M_BACKUPER_SMB_PASS"); v != "" {
		cfg.SMBPassword = v
		slog.Debug("overriding SMB password from environment")
	}
	if v := os.Getenv("M_BACKUPER_BACKUP_ROOT"); v != "" {
		cfg.BackupRoot = v
		slog.Debug("overriding backup root from environment", "backup_root", v)
	}

	return cfg, nil
}

func Save(cfg *Config) error {
	configPath, err := ConfigPath()
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal config to JSON with indentation
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	slog.Info("saved config to file", "path", configPath)
	return nil
}

// Passwords are redacted for security
func (c Config) String() string {
	password := c.SMBPassword
	if password != "" {
		password = "***REDACTED***"
	}

	return fmt.Sprintf(`Configuration:
  Backup Root: %s
  Device ID: %s
  Paths to Backup: %v
  Ignore Patterns: %v
  SMB User: %s
  SMB Password: %s`,
		c.BackupRoot,
		c.DeviceID,
		c.PathsToBackup,
		c.FilesToIgnorePatterns,
		c.SMBUser,
		password,
	)
}
