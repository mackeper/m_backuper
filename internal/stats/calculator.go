package stats

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mackeper/m_backuper/internal/display"
	"github.com/mackeper/m_backuper/internal/index"
)

// Calculator computes statistics from the database
type Calculator struct {
	db         *index.DB
	dbPath     string
}

// NewCalculator creates a new statistics calculator
func NewCalculator(db *index.DB, dbPath string) *Calculator {
	return &Calculator{
		db:     db,
		dbPath: dbPath,
	}
}

// Calculate computes comprehensive database statistics
func (c *Calculator) Calculate() (*display.DatabaseStats, error) {
	// Get all files from database
	files, err := c.db.GetAllFiles()
	if err != nil {
		return nil, fmt.Errorf("get files: %w", err)
	}

	// Calculate statistics
	var totalSize int64
	backupSetsMap := make(map[string]struct {
		count int64
		size  int64
	})
	rootDirsMap := make(map[string]int)

	for _, file := range files {
		totalSize += file.Size

		// Track by backup set
		setName := file.BackupSet
		if setName == "" {
			setName = "(no backup set)"
		}
		stats := backupSetsMap[setName]
		stats.count++
		stats.size += file.Size
		backupSetsMap[setName] = stats

		// Detect root directories (top 2 levels of path)
		parts := strings.Split(filepath.Clean(file.Path), string(filepath.Separator))
		if len(parts) >= 3 {
			root := filepath.Join(string(filepath.Separator), parts[1], parts[2])
			rootDirsMap[root]++
		} else if len(parts) >= 2 {
			root := filepath.Join(string(filepath.Separator), parts[1])
			rootDirsMap[root]++
		}
	}

	// Get database file size
	dbInfo, err := os.Stat(c.dbPath)
	var dbSize int64
	if err == nil {
		dbSize = dbInfo.Size()
	}

	// Build DatabaseStats structure
	dbStats := &display.DatabaseStats{
		TotalFiles:   int64(len(files)),
		TotalSize:    totalSize,
		DatabaseSize: dbSize,
		DatabasePath: c.dbPath,
	}

	// Convert backup sets map to slice
	for name, stats := range backupSetsMap {
		dbStats.BackupSets = append(dbStats.BackupSets, display.BackupSetStats{
			Name:  name,
			Count: stats.count,
			Size:  stats.size,
		})
	}

	// Sort and convert root directories
	type rootDirStat struct {
		path  string
		count int
	}
	var sortedRoots []rootDirStat
	for path, count := range rootDirsMap {
		sortedRoots = append(sortedRoots, rootDirStat{path, count})
	}

	// Simple bubble sort by count (descending)
	for i := 0; i < len(sortedRoots); i++ {
		for j := i + 1; j < len(sortedRoots); j++ {
			if sortedRoots[j].count > sortedRoots[i].count {
				sortedRoots[i], sortedRoots[j] = sortedRoots[j], sortedRoots[i]
			}
		}
	}

	// Convert to display format
	for _, root := range sortedRoots {
		dbStats.RootDirectories = append(dbStats.RootDirectories, display.RootDirStats{
			Path:  root.path,
			Count: root.count,
		})
	}

	return dbStats, nil
}
