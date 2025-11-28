package configutil

import (
	"fmt"
	"os"

	"github.com/mackeper/m_backuper/internal/config"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	BackupSet string
	Field     string
	Message   string
}

// ValidationResult holds the results of configuration validation
type ValidationResult struct {
	Valid  bool
	Errors []ValidationError
}

// ValidateBackupSet validates a single backup set configuration
func ValidateBackupSet(input BackupSetInput, checkPaths bool) error {
	if input.Name == "" {
		return fmt.Errorf("backup set name cannot be empty")
	}

	if len(input.Sources) == 0 {
		return fmt.Errorf("backup set must have at least one source")
	}

	if input.Destination == "" {
		return fmt.Errorf("backup set must have a destination")
	}

	// Optionally check if paths exist
	if checkPaths {
		for _, source := range input.Sources {
			if _, err := os.Stat(source); err != nil {
				return fmt.Errorf("source path does not exist: %s", source)
			}
		}

		if _, err := os.Stat(input.Destination); err != nil {
			return fmt.Errorf("destination path does not exist: %s", input.Destination)
		}
	}

	return nil
}

// ValidateConfig validates the entire configuration file
func ValidateConfig(cfg *config.Config) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:  true,
		Errors: []ValidationError{},
	}

	// Use the config's built-in validation
	if err := cfg.Validate(); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Message: err.Error(),
		})
		return result, nil
	}

	// Additional validation can be added here
	for _, bs := range cfg.BackupSets {
		if bs.Name == "" {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				BackupSet: bs.Name,
				Field:     "name",
				Message:   "backup set name cannot be empty",
			})
		}

		if len(bs.Sources) == 0 {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				BackupSet: bs.Name,
				Field:     "sources",
				Message:   "backup set must have at least one source",
			})
		}

		if bs.Destination == "" {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				BackupSet: bs.Name,
				Field:     "destination",
				Message:   "backup set must have a destination",
			})
		}
	}

	return result, nil
}
