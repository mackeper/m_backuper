package display

import "github.com/mackeper/m_backuper/internal/index"

// CalculateDuplicateSummary calculates summary statistics for duplicate groups
func CalculateDuplicateSummary(groups []index.DuplicateGroup) DuplicateSummary {
	var summary DuplicateSummary
	summary.GroupCount = len(groups)

	for _, group := range groups {
		summary.TotalWasted += group.WastedSpace
		summary.TotalFiles += group.FileCount
	}

	return summary
}
