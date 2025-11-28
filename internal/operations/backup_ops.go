package operations

import (
	"context"
	"log/slog"

	"github.com/mackeper/m_backuper/internal/backup"
	"github.com/mackeper/m_backuper/internal/config"
	"github.com/mackeper/m_backuper/internal/index"
)

// BackupOperation handles backup operations with progress tracking
type BackupOperation struct {
	engine   *backup.Engine
	cfg      *config.Config
	logger   *slog.Logger
	progress ProgressCallback
}

// BackupOptions configures backup execution
type BackupOptions struct {
	BackupSetName string           // Name of backup set to run
	DryRun        bool             // Preview without copying
	Progress      ProgressCallback // Optional progress callback
}

// NewBackupOperation creates a new backup operation
func NewBackupOperation(cfg *config.Config, db *index.DB, logger *slog.Logger) *BackupOperation {
	return &BackupOperation{
		engine: backup.NewEngine(cfg, db, logger),
		cfg:    cfg,
		logger: logger,
	}
}

// Run executes a backup operation for a single backup set
func (op *BackupOperation) Run(ctx context.Context, opts BackupOptions) (*backup.BackupResult, error) {
	op.progress = opts.Progress

	// Run backup using the engine
	result, err := op.engine.Backup(ctx, opts.BackupSetName, opts.DryRun)
	if err != nil {
		return nil, err
	}

	// Report completion if progress callback provided
	if op.progress != nil {
		op.progress(OperationProgress{
			Stage:      "complete",
			Percentage: 100,
		})
	}

	return result, nil
}

// BackupAll executes backup for all configured backup sets
func (op *BackupOperation) BackupAll(ctx context.Context, dryRun bool, progress ProgressCallback) ([]*backup.BackupResult, error) {
	op.progress = progress

	// Run backup for all sets using the engine
	results, err := op.engine.BackupAll(ctx, dryRun)
	if err != nil {
		return nil, err
	}

	// Report completion if progress callback provided
	if op.progress != nil {
		op.progress(OperationProgress{
			Stage:      "complete",
			Percentage: 100,
		})
	}

	return results, nil
}

// Validate checks if a backup set configuration is valid
func (op *BackupOperation) Validate(backupSetName string) error {
	backupSet := op.cfg.GetBackupSet(backupSetName)
	if backupSet == nil {
		return ErrBackupSetNotFound{Name: backupSetName}
	}
	return nil
}

// ErrBackupSetNotFound is returned when a backup set doesn't exist
type ErrBackupSetNotFound struct {
	Name string
}

func (e ErrBackupSetNotFound) Error() string {
	return "backup set not found: " + e.Name
}
