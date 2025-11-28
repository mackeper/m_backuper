# Architecture Overview

Quick reference guide to m_backuper's structure, components, and design decisions.

## Design Philosophy

- **Separation of Concerns**: Business logic (operations) separate from presentation (display/TUI)
- **Shared Core**: Both CLI and TUI use the same operations layer
- **Progress Tracking**: Operations support callbacks for real-time updates
- **Lazy Initialization**: Components created only when needed
- **Parallel Processing**: Worker pools for hashing and copying

## Directory Structure

```
m_backuper/
├── cmd/                    # CLI commands & TUI launcher
├── internal/
│   ├── tui/               # Interactive terminal UI (Bubbletea)
│   ├── operations/        # High-level business logic (shared)
│   ├── display/           # CLI output formatting
│   ├── stats/             # Statistics calculation
│   ├── configutil/        # Config file management
│   ├── backup/            # Backup engine (core)
│   ├── hash/              # SHA256 parallel hashing
│   ├── duplicate/         # Duplicate detection & cleanup
│   ├── index/             # SQLite database layer
│   └── config/            # Viper configuration
└── pkg/logger/            # Logging utilities
```

## Layer Responsibilities

### CLI Layer (`cmd/`)

**Purpose**: User-facing commands and TUI launcher

- `root.go` - Shared state (cfg, db, log), PersistentPreRunE initialization
- `tui.go` - Launch interactive TUI
- `backup.go` - Run backup operations
- `scan.go` - Scan and index files
- `list.go` - List duplicates
- `clean.go` - Delete duplicates
- `stats.go` - Show database statistics
- `config.go` - Manage backup sets

**Dependencies**: operations/, display/, stats/, configutil/

**Design**: Thin orchestration layer, delegates to operations

### TUI Layer (`internal/tui/`)

**Purpose**: Interactive Bubbletea interface

- `model.go` - Main TUI state machine, screen routing, navigation
- `styles.go` - Lipgloss color palette and styles
- `screens/menu.go` - Main navigation menu
- `screens/config.go` - Backup set CRUD (uses configutil.Manager)
- `screens/stats.go` - Statistics viewer (uses stats.Calculator)
- `screens/duplicates.go` - Browse/delete duplicates (uses operations.DuplicateOperation)
- `screens/backup.go` - Run backups with progress (uses operations.BackupOperation)
- `screens/messages.go` - Message types (NavigateMsg, ErrorMsg, SuccessMsg)

**Dependencies**: operations/, stats/, configutil/, config/, index/

**Design**: Elm architecture (Model, Update, View), lazy screen initialization, message passing

### Operations Layer (`internal/operations/`)

**Purpose**: High-level business logic, reusable by CLI and TUI

- `models.go` - Common types:
  - `OperationProgress` - Progress tracking state
  - `ProgressCallback` - Function type for real-time updates
  - `OperationResult` - Generic operation result
- `backup_ops.go` - `BackupOperation`:
  - `Run()` - Single backup set with progress
  - `BackupAll()` - All backup sets
  - `Validate()` - Check backup set exists
- `scan_ops.go` - `ScanOperation`:
  - `Run()` - Scan paths, hash files, update index
- `duplicate_ops.go` - `DuplicateOperation`:
  - `FindDuplicates()` - Multi-stage detection
  - `CleanDuplicates()` - Delete with strategy

**Dependencies**: backup/, hash/, duplicate/, index/, config/

**Design**: Context support, progress callbacks, returns structured data (not strings)

### Display Layer (`internal/display/`)

**Purpose**: CLI output formatting (not used by TUI)

- `formatter.go` - `Formatter` interface:
  - `FormatDuplicateGroups()`
  - `FormatStats()`
  - `FormatBackupSets()`
- `table.go` - `TableFormatter` - Human-readable tables
- `json.go` - `JSONFormatter` - Machine-readable JSON
- `summary.go` - `CalculateDuplicateSummary()` helper

**Dependencies**: index/, config/

**Design**: Strategy pattern for multiple output formats

### Stats Layer (`internal/stats/`)

**Purpose**: Database statistics calculation

- `calculator.go` - `Calculator`:
  - `Calculate()` - Returns `DatabaseStats`
  - Aggregates: total files, size, per-backup-set, top directories

**Dependencies**: display/, index/

**Design**: Extracts stats logic from cmd/stats.go for reusability

### Config Utilities (`internal/configutil/`)

**Purpose**: YAML file manipulation (bypasses validation)

- `manager.go` - `Manager`:
  - `AddBackupSet()` - Add to YAML
  - `RemoveBackupSet()` - Remove from YAML
  - `UpdateBackupSet()` - Modify YAML
  - `ListBackupSets()` - Read YAML directly
  - `InitConfig()` - Create example config
- `validator.go` - Validation helpers:
  - `ValidateBackupSet()` - Check inputs
  - `ValidateConfig()` - Full config validation

**Dependencies**: config/

**Design**: Works with raw YAML, allows adding backup sets before paths exist

### Backup Engine (`internal/backup/`)

**Purpose**: Core backup orchestration

- `engine.go` - `Engine`:
  - `Backup()` - Orchestrates walker → hasher → copier → indexer
  - `BackupAll()` - Loop over all backup sets
- `walker.go` - `Walker`:
  - `Walk()` - Directory traversal with exclusions
  - `WalkMultiple()` - Multiple source paths
- `copier.go` - `Copier`:
  - `CopyFiles()` - Parallel file copying with worker pool

**Dependencies**: hash/, index/, config/

**Design**: Sequential pipeline, each stage feeds the next

### Hash Calculator (`internal/hash/`)

**Purpose**: Parallel SHA256 hashing

- `calculator.go` - `Calculator`:
  - `HashFiles()` - Worker pool pattern
  - Returns `<-chan HashResult`

**Dependencies**: None (pure computation)

**Design**: Worker pool, configurable parallelism, channels for results

### Duplicate Detection (`internal/duplicate/`)

**Purpose**: Find and clean duplicate files

- `detector.go` - `Detector`:
  - Multi-stage: size filter → hash grouping → sort by wasted space
- `cleaner.go` - `Cleaner`:
  - `DeleteFiles()` - Delete with dry-run support
  - `SelectFilesToDelete()` - Strategy selection:
    - KeepFirst, KeepOldest, KeepNewest, KeepShortest

**Dependencies**: index/

**Design**: Multi-stage to minimize I/O, in-memory size grouping before hash checks

### Database Layer (`internal/index/`)

**Purpose**: SQLite persistence

- `db.go` - `DB`:
  - `UpsertFile()` - Add/update file record
  - `GetAllFiles()` - Load all indexed files
  - `FindDuplicates()` - SQL query for duplicates
  - `DeleteFile()` - Remove from index
  - `migrate()` - Schema migrations
- `models.go` - Types:
  - `FileRecord` - path, hash, size, modtime, backup_set
  - `DuplicateGroup` - hash, count, size, wasted space

**Dependencies**: None (uses database/sql)

**Design**: SQLite with indexes on hash+size, auto-migrations on Open()

### Configuration (`internal/config/`)

**Purpose**: Viper-based config loading and validation

- `config.go` - `Config`:
  - `Load()` - Load YAML with validation
  - `Validate()` - Check paths exist
  - `GetBackupSet()` - Lookup by name
  - Types: BackupSet, DuplicateConfig, ConcurrencyConfig, DatabaseConfig

**Dependencies**: None (uses viper)

**Design**: Validates on load (paths must exist), except for config add/remove commands

## Data Flow

### CLI Backup Flow
```
cmd/backup.go
  → operations.BackupOperation.Run()
    → backup.Engine.Backup()
      → walker.Walk() → hasher.HashFiles() → copier.CopyFiles() → db.UpsertFile()
  → display.Formatter.FormatBackupResult()
  → fmt.Println()
```

### TUI Backup Flow
```
screens/backup.go (Update)
  → operations.BackupOperation.Run(progress callback)
    → backup.Engine.Backup()
      → (same as above)
      → Calls progress callback during execution
  → Update model state
  → screens/backup.go (View)
    → Render progress bar and stats
```

### Duplicate Detection Flow
```
cmd/list.go or screens/duplicates.go
  → operations.DuplicateOperation.FindDuplicates()
    → duplicate.Detector.FindDuplicates()
      → db.GetAllFiles()
      → Group by size (in-memory)
      → Group by hash (only size duplicates)
      → Build DuplicateGroup structs
      → Sort by wasted space
  → display.Formatter.FormatDuplicateGroups() OR TUI View()
```

## Key Design Decisions

### Why Operations Layer?
- **Reusability**: CLI and TUI share the same business logic
- **Testability**: Can test operations without UI dependencies
- **Progress Tracking**: Unified progress callback interface

### Why Display Separate from Operations?
- **Separation**: Operations return data, display handles formatting
- **Multiple Formats**: table, json without changing operations
- **TUI Independence**: TUI doesn't need display layer

### Why Lazy TUI Screen Initialization?
- **Memory**: Don't create all screens at startup
- **Performance**: Only initialize what user navigates to
- **Flexibility**: Can pass context-specific parameters on creation

### Why Multi-Stage Duplicate Detection?
- **Performance**: Size grouping (fast, in-memory) filters before hash grouping (slow, I/O)
- **Efficiency**: Only hash files with duplicate sizes
- **Caching**: Database stores hashes, no re-computation

### Why Worker Pools for Hashing?
- **Parallelism**: CPU-bound operation benefits from multiple cores
- **Backpressure**: Channel buffering prevents memory explosion
- **Configurability**: User can tune worker count for their system

### Why SQLite?
- **Simplicity**: No server, single file database
- **Performance**: Indexes on hash+size make queries fast
- **Persistence**: Hashes survive between runs
- **Portability**: Works everywhere Go works

## Common Patterns

### Worker Pool
```go
jobs := make(chan Job)
results := make(chan Result)

// Start workers
for i := 0; i < numWorkers; i++ {
    go worker(jobs, results)
}

// Close results after workers finish
go func() {
    wg.Wait()
    close(results)
}()
```

Used by: hash.Calculator, backup.Copier

### Progress Callbacks
```go
type ProgressCallback func(progress OperationProgress)

op.Run(ctx, options, progressCallback)
  // Inside operation:
  if progress != nil {
      progress(OperationProgress{...})
  }
```

Used by: All operations (backup, scan, duplicates)

### Elm Architecture (TUI)
```go
type Model struct { state... }

func (m Model) Init() tea.Cmd { ... }
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { ... }
func (m Model) View() string { ... }
```

Used by: All TUI screens

### Strategy Pattern (Formatters)
```go
type Formatter interface {
    Format(...) (string, error)
}

type TableFormatter struct{}
type JSONFormatter struct{}
```

Used by: display package

## Testing Strategy

- Unit tests in `*_test.go` files
- `t.TempDir()` for temporary files
- Table-driven tests where applicable
- Mock-free (use real SQLite with temp DB)
- Integration: Test operations with real dependencies

## External Dependencies

- **CLI**: cobra (commands), viper (config)
- **TUI**: bubbletea (framework), bubbles (components), lipgloss (styling)
- **Database**: go-sqlite3 (requires GCC)
- **Utilities**: go-humanize (formatting), yaml.v3 (config CLI)
- **Stdlib**: crypto/sha256, database/sql, log/slog, context

## File Counts

```
cmd/           8 files   (CLI commands)
tui/           2 files   (model, styles)
tui/screens/   7 files   (individual screens)
operations/    4 files   (business logic)
display/       4 files   (formatters)
stats/         1 file    (calculator)
configutil/    2 files   (manager, validator)
backup/        3 files   (engine, walker, copier)
hash/          1 file    (calculator)
duplicate/     2 files   (detector, cleaner)
index/         2 files   (db, models)
config/        1 file    (config)
logger/        1 file    (wrapper)

Total: ~38 Go files
```

## Quick Lookup

**Need to add a CLI command?** → Create in `cmd/`, use operations layer

**Need to add a TUI screen?** → Create in `tui/screens/`, implement tea.Model

**Need to change business logic?** → Update `operations/`, affects both CLI and TUI

**Need to add output format?** → Implement `display.Formatter` interface

**Need to change database schema?** → Update `index/db.go` migrate()

**Need to add config option?** → Update `config/config.go` and validation
