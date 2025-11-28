# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**m_backuper** is a Go CLI and TUI tool for backing up files and detecting duplicates using SHA256 content hashing. It provides both command-line and interactive terminal interfaces, uses SQLite for indexing, and supports parallel processing. The codebase features a clean, layered architecture with reusable packages that separate business logic from presentation, enabling both CLI and TUI to share the same core operations.

## Build & Test Commands

```bash
# Build
go build -o m_backuper

# Run tests
go test ./...                    # All tests
go test ./internal/hash/ -v      # Specific package with verbose output

# Run the CLI
./m_backuper --help
./m_backuper backup photos --dry-run
./m_backuper list --format json
./m_backuper config list

# Run the TUI (interactive mode)
./m_backuper tui
```

## Architecture

### Layered Design

```
cmd/                    → CLI commands (Cobra)
  ├── root.go          → Shared state (cfg, db, log), PersistentPreRunE
  ├── tui.go           → TUI launcher (uses internal/tui)
  ├── backup.go        → Backup command (uses operations.BackupOperation)
  ├── scan.go          → Scan command (uses operations.ScanOperation)
  ├── list.go          → List duplicates (uses operations.DuplicateOperation + display.Formatter)
  ├── clean.go         → Clean duplicates (uses operations.DuplicateOperation)
  ├── stats.go         → Database stats (uses stats.Calculator + display.Formatter)
  └── config.go        → Config management (uses configutil.Manager)

internal/
  ├── tui/             → Terminal User Interface (NEW - Bubbletea)
  │   ├── model.go            → Main TUI model and navigation
  │   ├── styles.go           → Lipgloss styles and theming
  │   └── screens/            → Individual TUI screens
  │       ├── messages.go     → Common message types (NavigateMsg, ErrorMsg, SuccessMsg)
  │       ├── menu.go         → Main menu screen
  │       ├── config.go       → Config manager screen (add/edit/delete backup sets)
  │       ├── stats.go        → Stats viewer screen
  │       ├── duplicates.go   → Duplicate browser and cleaner screen
  │       └── backup.go       → Backup runner with progress tracking
  │
  ├── operations/      → High-level operations (NEW - for CLI and TUI)
  │   ├── models.go           → OperationProgress, OperationResult, ProgressCallback
  │   ├── duplicate_ops.go    → DuplicateOperation: FindDuplicates, CleanDuplicates
  │   ├── scan_ops.go         → ScanOperation: Run with progress callbacks
  │   └── backup_ops.go       → BackupOperation: Run, BackupAll, Validate
  │
  ├── display/         → Output formatting (NEW - reusable formatters)
  │   ├── formatter.go        → Formatter interface (table, json)
  │   ├── table.go            → TableFormatter for human-readable output
  │   ├── json.go             → JSONFormatter for machine-readable output
  │   └── summary.go          → CalculateDuplicateSummary helper
  │
  ├── stats/           → Statistics calculation (NEW)
  │   └── calculator.go       → Calculator: Calculate() returns DatabaseStats
  │
  ├── configutil/      → Config YAML operations (NEW)
  │   ├── manager.go          → Manager: CRUD for backup sets (Add, Remove, List, Update)
  │   └── validator.go        → ValidateBackupSet, ValidateConfig
  │
  ├── backup/          → Backup orchestration (existing)
  │   ├── engine.go           → Coordinates walker, hasher, copier
  │   ├── walker.go           → Directory traversal with exclusions
  │   └── copier.go           → Parallel file copying
  │
  ├── hash/            → SHA256 hashing (existing)
  │   └── calculator.go       → Worker pool pattern for parallel hashing
  │
  ├── duplicate/       → Duplicate detection (existing)
  │   ├── detector.go         → Multi-stage: size filter → hash grouping
  │   └── cleaner.go          → File deletion with strategies
  │
  ├── index/           → SQLite database (existing)
  │   ├── db.go               → CRUD operations, migrations
  │   └── models.go           → FileRecord, DuplicateGroup
  │
  └── config/          → Viper-based configuration (existing)
      └── config.go           → Load, validate, defaults

pkg/logger/            → slog wrapper
```

### Key Architectural Patterns

**1. Shared State in root.go**
- `cfg`, `db`, `log` are package-level vars in `cmd/`
- `PersistentPreRunE` initializes these for all commands
- Some commands (config add/remove/list) skip config validation

**2. Operations Layer (NEW)**
- High-level operations that encapsulate business logic
- Support progress callbacks for real-time updates
- Reusable by both CLI and TUI
- Each operation returns structured results (not formatted strings)

**3. Display Layer (NEW)**
- Separates formatting from business logic
- Formatter interface supports multiple output formats (table, json)
- Reusable across commands (list, stats)
- Commands work with structured data, formatters handle presentation

**4. Parallel Processing**
- Hash calculator: Worker pool with channels (`HashFiles` returns `<-chan HashResult`)
- File copier: Similar worker pool pattern
- Both use `runtime.NumCPU()` as default worker count
- Operations layer supports progress callbacks during parallel work

**5. Multi-Stage Duplicate Detection**
- Stage 1: Load all files from DB
- Stage 2: Group by size (in-memory, fast pre-filter)
- Stage 3: For duplicate sizes, group by hash
- Stage 4: Build DuplicateGroup structs
- Stage 5: Sort by wasted space
- Now wrapped in `operations.DuplicateOperation.FindDuplicates()`

**6. Database Design**
- `files` table: path (unique), hash, size, mod_time, backup_set
- Indexed on: hash, size, backup_set
- `duplicate_groups` view for fast queries
- `list` command queries DB only (no filesystem access)
- `scan` command walks filesystem and updates DB

**7. Config Management**
- Two approaches: CLI (`config add/remove`) or YAML editing
- Now uses `configutil.Manager` for all YAML operations
- `config add/remove` work without validation (paths may not exist yet)
- `config list` loads YAML directly, skips validation
- Other commands require valid config (paths must exist)

## Package Details

### internal/operations/

**Purpose**: High-level operations layer that encapsulates business logic with progress tracking.

**Key Features**:
- Progress callbacks for real-time updates
- Context support for cancellation
- Structured results (not formatted output)
- Reusable by CLI and TUI

**Types**:
```go
// models.go
type OperationProgress struct {
    Stage         string  // "scanning", "hashing", "copying", "indexing", "complete"
    FilesTotal    int64
    FilesComplete int64
    BytesTotal    int64
    BytesComplete int64
    CurrentFile   string
    Percentage    float64
}

type ProgressCallback func(progress OperationProgress)

// duplicate_ops.go
type DuplicateOperation struct { ... }
func (op *DuplicateOperation) FindDuplicates(ctx, opts FindOptions) ([]DuplicateGroup, error)
func (op *DuplicateOperation) CleanDuplicates(ctx, opts CleanOptions) (*CleanResult, error)

// scan_ops.go
type ScanOperation struct { ... }
func (op *ScanOperation) Run(ctx, opts ScanOptions) (*ScanResult, error)

// backup_ops.go
type BackupOperation struct { ... }
func (op *BackupOperation) Run(ctx, opts BackupOptions) (*BackupResult, error)
func (op *BackupOperation) BackupAll(ctx, dryRun, progress) ([]*BackupResult, error)
```

### internal/display/

**Purpose**: Output formatting abstraction - separates presentation from business logic.

**Key Features**:
- Formatter interface for multiple output formats
- TableFormatter for human-readable output
- JSONFormatter for machine-readable output
- Reusable across commands

**Types**:
```go
type Formatter interface {
    FormatDuplicateGroups(groups []DuplicateGroup) (string, error)
    FormatStats(stats *DatabaseStats) (string, error)
    FormatBackupSets(sets []BackupSet) (string, error)
}

func NewFormatter(format string) (Formatter, error) // "table" or "json"

type DatabaseStats struct {
    TotalFiles      int64
    TotalSize       int64
    DatabaseSize    int64
    DatabasePath    string
    BackupSets      []BackupSetStats
    RootDirectories []RootDirStats
}
```

**Usage Example**:
```go
// Get data from operations
groups, _ := dupOp.FindDuplicates(ctx, opts)

// Format for output
formatter, _ := display.NewFormatter("table")
output, _ := formatter.FormatDuplicateGroups(groups)
fmt.Println(output)
```

### internal/stats/

**Purpose**: Statistics calculation extracted from cmd/stats.go.

**Types**:
```go
type Calculator struct { db *index.DB }
func (c *Calculator) Calculate() (*display.DatabaseStats, error)
```

**Usage**:
```go
calc := stats.NewCalculator(db, dbPath)
stats, _ := calc.Calculate()
```

### internal/configutil/

**Purpose**: Configuration file management - YAML CRUD operations.

**Key Features**:
- Add, Remove, List, Update backup sets
- Works with raw YAML (doesn't require validation)
- Validation utilities

**Types**:
```go
type Manager struct { configPath string }
func (m *Manager) AddBackupSet(input BackupSetInput) error
func (m *Manager) RemoveBackupSet(name string) error
func (m *Manager) ListBackupSets() ([]BackupSetDisplay, error)
func (m *Manager) UpdateBackupSet(name string, input BackupSetInput) error
func (m *Manager) InitConfig(overwrite bool) error

type BackupSetInput struct {
    Name, Destination string
    Sources, Excludes []string
}
```

## Important Implementation Details

### Adding New Commands

1. Create file in `cmd/` (e.g., `cmd/newcmd.go`)
2. Use `rootCmd.AddCommand(newCmd)` in `init()`
3. Access shared state: `cfg`, `db`, `log` (initialized by PersistentPreRunE)
4. Use operations layer for business logic: `operations.NewXOperation()`
5. Use display layer for output: `display.NewFormatter(format)`
6. If command needs to skip config loading, update `root.go` skip logic

**Example Command Structure**:
```go
func runNewCommand(cmd *cobra.Command, args []string) error {
    // 1. Create operation
    op := operations.NewSomeOperation(db, cfg, log)

    // 2. Run operation
    result, err := op.Run(ctx, options)
    if err != nil {
        return err
    }

    // 3. Format output
    formatter, _ := display.NewFormatter("table")
    output, _ := formatter.FormatSomething(result)
    fmt.Println(output)

    return nil
}
```

### Progress Callbacks (for TUI)

Operations support progress callbacks for real-time updates:

```go
progressCallback := func(progress operations.OperationProgress) {
    fmt.Printf("\r%s: %d/%d files (%.1f%%)",
        progress.Stage,
        progress.FilesComplete,
        progress.FilesTotal,
        progress.Percentage)
}

scanOp := operations.NewScanOperation(db, log)
result, _ := scanOp.Run(ctx, operations.ScanOptions{
    Paths:    []string{"/path"},
    Progress: progressCallback,
})
```

### Worker Pool Pattern

Used by hash calculator and copier:

```go
func Process(ctx context.Context, jobs <-chan Job) <-chan Result {
    results := make(chan Result)
    var wg sync.WaitGroup

    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go worker(ctx, jobs, results, &wg)
    }

    go func() {
        wg.Wait()
        close(results)
    }()

    return results
}
```

### Database Migrations

Schema is in `internal/index/db.go` `migrate()` method. Run automatically on `index.Open()`. Uses `CREATE TABLE IF NOT EXISTS` for idempotence.

### Config Validation vs Loading

- `config.Load()` calls `Validate()` which checks paths exist
- `configutil.Manager` works with raw YAML without validation
- This allows adding backup sets before creating source directories
- `config add/remove/list` use Manager (no validation)
- Other commands require validated config

## Testing Strategy

- Unit tests in `*_test.go` files (see `internal/hash/calculator_test.go`)
- Use `t.TempDir()` for temporary files
- Test files use table-driven tests where applicable
- Integration tests would need temporary SQLite database
- Test operations layer separately from CLI commands
- Test formatters with known inputs/outputs

## Configuration Files

- Default: `~/.m_backuper/config.yaml`
- Override: `--config` flag or `./config.yaml`
- Example: `configs/config.example.yaml`
- Database: `~/.m_backuper/index.db` (default)

## Common Gotchas

1. **`list` doesn't scan filesystem** - it only queries the database. Use `scan` first to index files.
2. **Config commands skip validation** - `config add` allows non-existent paths. They're validated when running `backup`.
3. **PersistentPreRunE shared state** - All commands (CLI and TUI) share `cfg`, `db`, `log` initialized in root.go. Don't re-initialize.
4. **Worker pool channels** - Jobs channel is sent to workers, results channel is returned. Close jobs, wait for workers, then close results.
5. **Backup engine flow** - The backup process is sequential: walk → hash → copy → index. Each stage feeds the next.
6. **Operations return data, not formatted strings** - Use display.Formatter to format operation results for output.
7. **Progress callbacks are optional** - CLI may ignore them, TUI uses them for real-time updates.
8. **TUI screens need configPath** - Config screen requires configPath parameter to reload config after edits (not available in config.Config struct).
9. **TUI lazy initialization** - Screens are created on first navigation, not at startup. Check for nil before delegating updates.
10. **Config field names** - Use `cfg.BackupSets` (not `Backups`), `cfg.Database.Path` (config has no `ConfigPath` field).

## Code Organization Principles

**Separation of Concerns**:
- **operations/** - Business logic, no formatting (shared by CLI and TUI)
- **display/** - Formatting logic, no business logic (used by CLI)
- **tui/** - Interactive interface using Bubbletea (uses operations directly)
- **cmd/** - CLI commands and TUI launcher, orchestrates operations + display
- **internal/** - Core functionality, reusable across interfaces

**Data Flow**:
```
CLI Command → Operations (business logic) → Structured Data → Display (formatting) → Output
     ↓              ↓                            ↓                     ↓
  User Input   Progress Callbacks         Domain Models         Formatted Strings

TUI Screen  → Operations (business logic) → Structured Data → TUI View (Bubbletea) → Render
     ↓              ↓                            ↓                     ↓
  User Input   Progress Callbacks         Domain Models         Styled Components
```

**Reusability**:
- Operations layer is CLI/TUI agnostic (core design principle)
- Display layer provides CLI output formatting (table, json)
- TUI layer provides interactive interface (Bubbletea)
- All interfaces share the same business logic

## Git Commit Guidelines

**Before Committing**:
- ALWAYS run `go build -o m_backuper` to ensure code compiles
- ALWAYS run `go test ./...` to ensure all tests pass
- Only create commits when build and tests are successful
- Fix any compilation errors or test failures before committing

**Commit Message Format**:
- NEVER include Claude attribution or AI-generated markers in commit messages
- Do NOT add lines like "🤖 Generated with Claude Code"
- Do NOT add "Co-Authored-By: Claude" tags
- Write clear, descriptive commit messages focused on technical changes
- Use standard commit message format: short summary, blank line, detailed description

**Example Good Commit**:
```
Refactor: Extract operations and display layers

- Create internal/operations/ for business logic with progress tracking
- Create internal/display/ for output formatting (table, json)
- Create internal/stats/ for statistics calculation
- Create internal/configutil/ for config YAML management
- Update all commands to use new packages
- Reduce code duplication by ~460 lines
```

## Dependencies

Key external packages:
- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration
- `github.com/mattn/go-sqlite3` - SQLite (requires GCC)
- `github.com/dustin/go-humanize` - Human-readable formats
- `gopkg.in/yaml.v3` - YAML parsing (for config CLI commands)
- `github.com/charmbracelet/bubbletea` - TUI framework (Elm architecture)
- `github.com/charmbracelet/bubbles` - TUI components (list, textinput, progress)
- `github.com/charmbracelet/lipgloss` - TUI styling and layout

Standard library: `crypto/sha256`, `log/slog`, `database/sql`, `path/filepath`, `context`

## TUI (Terminal User Interface)

The TUI provides an interactive interface built with [Bubbletea](https://github.com/charmbracelet/bubbletea) following the Elm architecture (Model, Update, View).

**Features**:
- **Config Manager**: Add, edit, and delete backup sets interactively
- **Stats Viewer**: Real-time database statistics with refresh capability
- **Duplicate Browser**: Browse duplicate groups, select deletion strategy, and clean files
- **Backup Runner**: Run backups with real-time progress tracking

**Architecture**:
```
internal/tui/
  ├── model.go        → Main TUI state, navigation, screen routing
  ├── styles.go       → Lipgloss styles and color palette
  └── screens/        → Individual screens implementing tea.Model
      ├── menu.go     → Main navigation menu
      ├── config.go   → Backup set management (uses configutil.Manager)
      ├── stats.go    → Statistics display (uses stats.Calculator)
      ├── duplicates.go → Duplicate browsing (uses operations.DuplicateOperation)
      └── backup.go   → Backup execution (uses operations.BackupOperation)
```

**Navigation**:
- Arrow keys or j/k to navigate menus and lists
- Enter to select/confirm
- Esc or q to go back
- Ctrl+C to quit from main menu

**Key Design Patterns**:
1. **Elm Architecture**: Each screen is a tea.Model with Init(), Update(), View()
2. **Message Passing**: NavigateMsg, ErrorMsg, SuccessMsg for screen communication
3. **Shared Operations**: TUI and CLI use the same operations layer
4. **Progress Callbacks**: Real-time updates for long-running operations
5. **Lazy Initialization**: Screens created on first navigation (memory efficient)

**Usage**:
```bash
# Launch TUI
./m_backuper tui

# TUI uses the same config file as CLI
./m_backuper --config custom.yaml tui
```

**Implementation Notes**:
- TUI requires valid config and database (initialized via PersistentPreRunE)
- Config screen needs configPath parameter to reload config after edits
- Progress tracking uses operations.ProgressCallback for real-time updates
- All TUI screens access shared state (cfg, db, logger) from main model
