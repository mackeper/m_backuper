package config

import (
	"testing"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()
	if cfg.allowDuplicates {
		t.Errorf("Expected allowDuplicates to be false, got true")
	}
	if len(cfg.directories) != 0 {
		t.Errorf("Expected directories to be empty, got %d entries", len(cfg.directories))
	}
}

func TestAllowDuplicates(t *testing.T) {
	config := NewConfig().AllowDuplicates()
	if !config.allowDuplicates {
		t.Errorf("Expected allowDuplicates to be true, got false")
	}
}

func TestDisllowDuplicates(t *testing.T) {
	config := NewConfig().AllowDuplicates()
	if !config.allowDuplicates {
		t.Errorf("Expected allowDuplicates to be true, got false")
	}
	config = config.DisallowDuplicates()
	if config.allowDuplicates {
		t.Errorf("Expected allowDuplicates to be false, got true")
	}
}

func TestIsAllowDuplicates(t *testing.T) {
	config := NewConfig()
	if config.IsAllowDuplicates() {
		t.Errorf("Expected allowDuplicates to be false, got true")
	}
	config = config.AllowDuplicates()
	if !config.IsAllowDuplicates() {
		t.Errorf("Expected allowDuplicates to be true, got false")
	}
}

func TestAddDirectory(t *testing.T) {
	config := NewConfig().AddDirectory("./", "/tmp/backup")
	if config.directories["./"] != "/tmp/backup" {
		t.Errorf("Expected directory './' to map to '/tmp/backup', got '%s'", config.directories["./"])
	}
	config.AddDirectory("./2", "/tmp/backup2")
	if config.directories["./"] != "/tmp/backup" {
		t.Errorf("Expected directory './' to map to '/tmp/backup', got '%s'", config.directories["./"])
	}
	if config.directories["./2"] != "/tmp/backup2" {
		t.Errorf("Expected directory './2' to map to '/tmp/backup2', got '%s'", config.directories["./2"])
	}
}

func TestWriteToFile(t *testing.T) {
	config := NewConfig().AddDirectory("./", "/tmp/backup").AllowDuplicates()
	err := config.WriteToFile()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Read the file back to verify contents
	readConfig, err := ReadConfigToFile()
	if err != nil {
		t.Errorf("Expected no error reading config file, got %v", err)
	}

	if readConfig.allowDuplicates != config.allowDuplicates {
		t.Errorf("Expected allowDuplicates to be %v, got %v", config.allowDuplicates, readConfig.allowDuplicates)
	}
	if len(readConfig.directories) != len(config.directories) {
		t.Errorf("Expected %d directories, got %d", len(config.directories), len(readConfig.directories))
	}
}
