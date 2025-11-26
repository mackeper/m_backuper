package index

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB represents the file index database
type DB struct {
	db *sql.DB
}

// Open opens or creates the database at the specified path
func Open(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Initialize schema
	idx := &DB{db: db}
	if err := idx.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	return idx, nil
}

// Close closes the database connection
func (d *DB) Close() error {
	return d.db.Close()
}

// migrate creates or updates the database schema
func (d *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT NOT NULL UNIQUE,
		hash TEXT NOT NULL,
		size INTEGER NOT NULL,
		mod_time DATETIME NOT NULL,
		backup_set TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_files_hash ON files(hash);
	CREATE INDEX IF NOT EXISTS idx_files_size ON files(size);
	CREATE INDEX IF NOT EXISTS idx_files_backup_set ON files(backup_set);

	CREATE TABLE IF NOT EXISTS backup_runs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		start_time DATETIME NOT NULL,
		end_time DATETIME,
		files_copied INTEGER DEFAULT 0,
		bytes_copied INTEGER DEFAULT 0,
		status TEXT NOT NULL,
		error TEXT
	);

	CREATE VIEW IF NOT EXISTS duplicate_groups AS
	SELECT
		hash,
		COUNT(*) as file_count,
		MIN(size) as file_size,
		(COUNT(*) - 1) * MIN(size) as wasted_space
	FROM files
	GROUP BY hash
	HAVING COUNT(*) > 1;
	`

	_, err := d.db.Exec(schema)
	return err
}

// UpsertFile inserts or updates a file record
func (d *DB) UpsertFile(record *FileRecord) error {
	query := `
	INSERT INTO files (path, hash, size, mod_time, backup_set, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(path) DO UPDATE SET
		hash = excluded.hash,
		size = excluded.size,
		mod_time = excluded.mod_time,
		backup_set = excluded.backup_set,
		updated_at = excluded.updated_at
	`

	now := time.Now()
	_, err := d.db.Exec(query,
		record.Path,
		record.Hash,
		record.Size,
		record.ModTime,
		record.BackupSet,
		now,
		now,
	)
	return err
}

// GetFile retrieves a file by path
func (d *DB) GetFile(path string) (*FileRecord, error) {
	query := `SELECT id, path, hash, size, mod_time, backup_set, created_at, updated_at FROM files WHERE path = ?`

	var record FileRecord
	err := d.db.QueryRow(query, path).Scan(
		&record.ID,
		&record.Path,
		&record.Hash,
		&record.Size,
		&record.ModTime,
		&record.BackupSet,
		&record.CreatedAt,
		&record.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &record, nil
}

// GetAllFiles retrieves all file records
func (d *DB) GetAllFiles() ([]FileRecord, error) {
	query := `SELECT id, path, hash, size, mod_time, backup_set, created_at, updated_at FROM files`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []FileRecord
	for rows.Next() {
		var record FileRecord
		err := rows.Scan(
			&record.ID,
			&record.Path,
			&record.Hash,
			&record.Size,
			&record.ModTime,
			&record.BackupSet,
			&record.CreatedAt,
			&record.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, record)
	}

	return files, rows.Err()
}

// GetFilesByBackupSet retrieves all files for a specific backup set
func (d *DB) GetFilesByBackupSet(backupSet string) ([]FileRecord, error) {
	query := `SELECT id, path, hash, size, mod_time, backup_set, created_at, updated_at FROM files WHERE backup_set = ?`

	rows, err := d.db.Query(query, backupSet)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []FileRecord
	for rows.Next() {
		var record FileRecord
		err := rows.Scan(
			&record.ID,
			&record.Path,
			&record.Hash,
			&record.Size,
			&record.ModTime,
			&record.BackupSet,
			&record.CreatedAt,
			&record.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, record)
	}

	return files, rows.Err()
}

// CreateBackupRun creates a new backup run record
func (d *DB) CreateBackupRun(name string) (int64, error) {
	query := `INSERT INTO backup_runs (name, start_time, status) VALUES (?, ?, ?)`
	result, err := d.db.Exec(query, name, time.Now(), "running")
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// UpdateBackupRun updates a backup run record
func (d *DB) UpdateBackupRun(id int64, filesCopied, bytesCopied int64) error {
	query := `UPDATE backup_runs SET files_copied = ?, bytes_copied = ? WHERE id = ?`
	_, err := d.db.Exec(query, filesCopied, bytesCopied, id)
	return err
}

// CompleteBackupRun marks a backup run as completed
func (d *DB) CompleteBackupRun(id int64, status string, errMsg *string) error {
	query := `UPDATE backup_runs SET end_time = ?, status = ?, error = ? WHERE id = ?`
	_, err := d.db.Exec(query, time.Now(), status, errMsg, id)
	return err
}

// DeleteFile removes a file record from the database
func (d *DB) DeleteFile(path string) error {
	query := `DELETE FROM files WHERE path = ?`
	_, err := d.db.Exec(query, path)
	return err
}
