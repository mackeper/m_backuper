package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.BackupRoot == "" {
		t.Error("default backup root should not be empty")
	}
	if cfg.DeviceID == "" {
		t.Error("default device ID should not be empty")
	}
	if cfg.PathsToBackup == nil {
		t.Error("paths to backup should be initialized")
	}
	if cfg.FilesToIgnorePatterns == nil {
		t.Error("ignore patterns should be initialized")
	}
}

func TestLoadFromValidFile(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	testConfig := Config{
		BackupRoot:            "//test-server/backups",
		DeviceID:              "test-device",
		PathsToBackup:         []string{"/test/path1", "/test/path2"},
		FilesToIgnorePatterns: []string{"*.log", "*.tmp"},
		SMBUser:               "testuser",
	}

	data, err := json.MarshalIndent(testConfig, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal test config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Read the config
	cfg := Default()
	data, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	// Verify
	if cfg.BackupRoot != testConfig.BackupRoot {
		t.Errorf("expected backup root %s, got %s", testConfig.BackupRoot, cfg.BackupRoot)
	}
	if cfg.DeviceID != testConfig.DeviceID {
		t.Errorf("expected device ID %s, got %s", testConfig.DeviceID, cfg.DeviceID)
	}
	if len(cfg.PathsToBackup) != 2 {
		t.Errorf("expected 2 paths to backup, got %d", len(cfg.PathsToBackup))
	}
}

func TestEnvironmentVariableOverride(t *testing.T) {
	// Set environment variables
	if err := os.Setenv("M_BACKUPER_SMB_USER", "envuser"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("M_BACKUPER_SMB_PASS", "envpass"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("M_BACKUPER_BACKUP_ROOT", "//env-server/backups"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Unsetenv("M_BACKUPER_SMB_USER")
		_ = os.Unsetenv("M_BACKUPER_SMB_PASS")
		_ = os.Unsetenv("M_BACKUPER_BACKUP_ROOT")
	}()

	cfg := Default()

	// Apply overrides
	if v := os.Getenv("M_BACKUPER_SMB_USER"); v != "" {
		cfg.SMBUser = v
	}
	if v := os.Getenv("M_BACKUPER_SMB_PASS"); v != "" {
		cfg.SMBPassword = v
	}
	if v := os.Getenv("M_BACKUPER_BACKUP_ROOT"); v != "" {
		cfg.BackupRoot = v
	}

	if cfg.SMBUser != "envuser" {
		t.Errorf("expected SMB user 'envuser', got '%s'", cfg.SMBUser)
	}
	if cfg.SMBPassword != "envpass" {
		t.Errorf("expected SMB password 'envpass', got '%s'", cfg.SMBPassword)
	}
	if cfg.BackupRoot != "//env-server/backups" {
		t.Errorf("expected backup root '//env-server/backups', got '%s'", cfg.BackupRoot)
	}
}

func TestMissingFileReturnsDefaults(t *testing.T) {
	// Try to read from non-existent file
	nonExistentPath := "/tmp/nonexistent-config-12345.json"

	_, err := os.ReadFile(nonExistentPath)
	if !os.IsNotExist(err) {
		t.Fatalf("expected file not found error, got: %v", err)
	}

	// Should fall back to defaults
	cfg := Default()
	if cfg.BackupRoot == "" {
		t.Error("default config should have backup root")
	}
}

func TestInvalidJSONReturnsError(t *testing.T) {
	// Create temporary file with invalid JSON
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	invalidJSON := []byte(`{this is not valid json}`)
	if err := os.WriteFile(configPath, invalidJSON, 0644); err != nil {
		t.Fatalf("failed to write invalid config: %v", err)
	}

	// Try to parse it
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	var cfg Config
	err = json.Unmarshal(data, &cfg)
	if err == nil {
		t.Error("expected error when parsing invalid JSON, got nil")
	}
}

func TestSave(t *testing.T) {
	// Save to the actual config path (will be in user's home directory)
	// We'll read it back to verify
	testConfig := Config{
		BackupRoot:            "//test-server/backups",
		DeviceID:              "test-device",
		PathsToBackup:         []string{"/test/path"},
		FilesToIgnorePatterns: []string{"*.tmp"},
	}

	// Save config
	if err := Save(&testConfig); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify file was created
	configPath, _ := ConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Read and verify content
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read saved config: %v", err)
	}

	var loadedConfig Config
	if err := json.Unmarshal(data, &loadedConfig); err != nil {
		t.Fatalf("failed to unmarshal saved config: %v", err)
	}

	if loadedConfig.BackupRoot != testConfig.BackupRoot {
		t.Errorf("expected backup root %s, got %s", testConfig.BackupRoot, loadedConfig.BackupRoot)
	}

	// Clean up
	_ = os.Remove(configPath)
}

func TestConfigString(t *testing.T) {
	cfg := Config{
		BackupRoot:    "//test/backup",
		DeviceID:      "test-device",
		PathsToBackup: []string{"/test"},
		SMBUser:       "user",
		SMBPassword:   "secret",
	}

	str := cfg.String()

	// Password should be redacted
	if contains(str, "secret") {
		t.Error("password should be redacted in string output")
	}
	if !contains(str, "***REDACTED***") {
		t.Error("redacted placeholder should be present")
	}
	if !contains(str, "test-device") {
		t.Error("device ID should be present")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
