package display

import (
	"fmt"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/mackeper/m_backuper/internal/config"
	"github.com/mackeper/m_backuper/internal/index"
)

// TableFormatter formats output as human-readable tables
type TableFormatter struct{}

// FormatDuplicateGroups formats duplicate groups as a table
func (t *TableFormatter) FormatDuplicateGroups(groups []index.DuplicateGroup) (string, error) {
	if len(groups) == 0 {
		return "No duplicates found", nil
	}

	var b strings.Builder

	b.WriteString("\nDuplicate Groups\n")
	b.WriteString("================\n")

	summary := CalculateDuplicateSummary(groups)

	for i, group := range groups {
		b.WriteString(fmt.Sprintf("\n%d. Hash: %s\n", i+1, group.Hash[:16]+"..."))
		b.WriteString(fmt.Sprintf("   Files: %d | Size: %s | Wasted: %s\n",
			group.FileCount,
			humanize.Bytes(uint64(group.FileSize)),
			humanize.Bytes(uint64(group.WastedSpace))))

		for j, file := range group.Files {
			b.WriteString(fmt.Sprintf("   [%d] %s\n", j+1, file.Path))
		}
	}

	b.WriteString("\nSummary\n")
	b.WriteString("=======\n")
	b.WriteString(fmt.Sprintf("Duplicate groups: %d\n", summary.GroupCount))
	b.WriteString(fmt.Sprintf("Total wasted space: %s\n", humanize.Bytes(uint64(summary.TotalWasted))))

	return b.String(), nil
}

// FormatStats formats database statistics as a table
func (t *TableFormatter) FormatStats(stats *DatabaseStats) (string, error) {
	var b strings.Builder

	b.WriteString("\nDatabase Statistics\n")
	b.WriteString("===================\n")
	b.WriteString(fmt.Sprintf("Database: %s\n", stats.DatabasePath))
	if stats.DatabaseSize > 0 {
		b.WriteString(fmt.Sprintf("Database size: %s\n", humanize.Bytes(uint64(stats.DatabaseSize))))
	}
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("Total files: %s\n", humanize.Comma(stats.TotalFiles)))
	b.WriteString(fmt.Sprintf("Total size: %s\n", humanize.Bytes(uint64(stats.TotalSize))))
	b.WriteString("\n")

	// Backup sets breakdown
	if len(stats.BackupSets) > 0 {
		b.WriteString("Files by Backup Set:\n")
		b.WriteString("--------------------\n")
		for _, set := range stats.BackupSets {
			b.WriteString(fmt.Sprintf("  %-30s %8s files  %12s\n",
				set.Name,
				humanize.Comma(set.Count),
				humanize.Bytes(uint64(set.Size))))
		}
		b.WriteString("\n")
	}

	// Root directories
	if len(stats.RootDirectories) > 0 {
		b.WriteString("Top Root Directories:\n")
		b.WriteString("---------------------\n")

		maxShow := 10
		if len(stats.RootDirectories) < maxShow {
			maxShow = len(stats.RootDirectories)
		}

		for i := 0; i < maxShow; i++ {
			b.WriteString(fmt.Sprintf("  %-50s %8s files\n",
				stats.RootDirectories[i].Path,
				humanize.Comma(int64(stats.RootDirectories[i].Count))))
		}

		if len(stats.RootDirectories) > maxShow {
			b.WriteString(fmt.Sprintf("  ... and %d more\n", len(stats.RootDirectories)-maxShow))
		}
	}

	return b.String(), nil
}

// FormatBackupSets formats backup set configurations as a table
func (t *TableFormatter) FormatBackupSets(sets []config.BackupSet) (string, error) {
	if len(sets) == 0 {
		return "No backup sets configured", nil
	}

	var b strings.Builder

	b.WriteString("\nConfigured Backup Sets\n")
	b.WriteString("======================\n")

	for i, set := range sets {
		b.WriteString(fmt.Sprintf("\n%d. %s\n", i+1, set.Name))
		b.WriteString(fmt.Sprintf("   Sources: %v\n", set.Sources))
		b.WriteString(fmt.Sprintf("   Destination: %s\n", set.Destination))
		if len(set.Excludes) > 0 {
			b.WriteString(fmt.Sprintf("   Excludes: %v\n", set.Excludes))
		}
	}

	return b.String(), nil
}
