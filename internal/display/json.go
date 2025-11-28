package display

import (
	"bytes"
	"encoding/json"

	"github.com/mackeper/m_backuper/internal/config"
	"github.com/mackeper/m_backuper/internal/index"
)

// JSONFormatter formats output as JSON
type JSONFormatter struct{}

// FormatDuplicateGroups formats duplicate groups as JSON
func (j *JSONFormatter) FormatDuplicateGroups(groups []index.DuplicateGroup) (string, error) {
	type jsonOutput struct {
		Groups  []index.DuplicateGroup `json:"groups"`
		Summary DuplicateSummary       `json:"summary"`
	}

	summary := CalculateDuplicateSummary(groups)

	output := jsonOutput{
		Groups:  groups,
		Summary: summary,
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// FormatStats formats database statistics as JSON
func (j *JSONFormatter) FormatStats(stats *DatabaseStats) (string, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(stats); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// FormatBackupSets formats backup set configurations as JSON
func (j *JSONFormatter) FormatBackupSets(sets []config.BackupSet) (string, error) {
	type jsonOutput struct {
		BackupSets []config.BackupSet `json:"backup_sets"`
		Count      int                `json:"count"`
	}

	output := jsonOutput{
		BackupSets: sets,
		Count:      len(sets),
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return "", err
	}

	return buf.String(), nil
}
