package index

import "time"

// FileRecord represents a file entry in the database
type FileRecord struct {
	ID         int64     `db:"id"`
	Path       string    `db:"path"`
	Hash       string    `db:"hash"`
	Size       int64     `db:"size"`
	ModTime    time.Time `db:"mod_time"`
	BackupSet  string    `db:"backup_set"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

// BackupRun represents a backup operation
type BackupRun struct {
	ID          int64     `db:"id"`
	Name        string    `db:"name"`
	StartTime   time.Time `db:"start_time"`
	EndTime     *time.Time `db:"end_time"`
	FilesCopied int64     `db:"files_copied"`
	BytesCopied int64     `db:"bytes_copied"`
	Status      string    `db:"status"` // 'running', 'completed', 'failed'
	Error       *string   `db:"error"`
}

// DuplicateGroup represents a group of duplicate files
type DuplicateGroup struct {
	Hash        string
	FileCount   int64
	FileSize    int64
	WastedSpace int64
	Files       []FileRecord
}
