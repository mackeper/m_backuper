package state

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

//nolint:govet // fieldalignment: field order optimized for JSON readability
type FileState struct {
	Size     int64  `json:"size"`
	BackedUp string `json:"backed_up"` // ISO 8601 timestamp
}

type State struct {
	LastRun time.Time            `json:"last_run"`
	Files   map[string]FileState `json:"files"`
}

func New() *State {
	return &State{
		Files: make(map[string]FileState),
	}
}

func StatePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "m_backuper", "state.json"), nil
}

func Load() (*State, error) {
	statePath, err := StatePath()
	if err != nil {
		return New(), err
	}
	return LoadFrom(statePath)
}

func LoadFrom(statePath string) (*State, error) {
	state := New()

	// Try to load from file
	data, err := os.ReadFile(statePath) //nolint:gosec // State path is from trusted source
	if err != nil {
		if os.IsNotExist(err) {
			slog.Info("state file not found, starting fresh", "path", statePath)
			return state, nil // First run, return empty state
		}
		slog.Error("failed to read state file", "path", statePath, "error", err)
		return state, fmt.Errorf("failed to read state file: %w", err)
	}

	if err := json.Unmarshal(data, state); err != nil {
		slog.Error("failed to parse state file (corrupted)", "path", statePath, "error", err)
		return state, fmt.Errorf("corrupted state file: %w", err)
	}

	slog.Info("loaded state from file", "path", statePath, "file_count", len(state.Files))
	return state, nil
}

func (s *State) Save() error {
	statePath, err := StatePath()
	if err != nil {
		return err
	}
	return s.SaveTo(statePath)
}

func (s *State) SaveTo(statePath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(statePath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		slog.Error("failed to create state directory", "path", dir, "error", err)
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Update last run time
	s.LastRun = time.Now()

	// Marshal state to JSON with indentation
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to file
	if err := os.WriteFile(statePath, data, 0o600); err != nil {
		slog.Error("failed to write state file", "path", statePath, "error", err)
		return fmt.Errorf("failed to write state file: %w", err)
	}

	slog.Info("saved state to file", "path", statePath, "file_count", len(s.Files))
	return nil
}

func (s *State) GetFileState(path string) (FileState, bool) {
	state, exists := s.Files[path]
	return state, exists
}

func (s *State) SetFileState(path string, size int64) {
	s.Files[path] = FileState{
		Size:     size,
		BackedUp: time.Now().Format(time.RFC3339),
	}
}

func (s *State) RemoveFileState(path string) {
	delete(s.Files, path)
}

func (s *State) FileCount() int {
	return len(s.Files)
}
