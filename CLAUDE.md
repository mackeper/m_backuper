# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**m_backuper** is a Go CLI tool for backing up files and detecting duplicates using SHA256 content hashing. It uses SQLite for indexing and supports parallel processing.

## Build & Test Commands

```bash
# Build
go build -o m_backuper

# Run tests
go test ./...                    # All tests
go test ./internal/hash/ -v      # Specific package with verbose output

# Run the tool
./m_backuper --help
./m_backuper backup photos --dry-run
```

## Architecture

### Layered Design

```
cmd/                    → CLI commands (Cobra)
  ├── root.go          → Shared state (cfg, db, log), PersistentPreRunE
  ├── backup.go        → Orchestrates backup via engine
  ├── scan.go          → Direct file scanning without config
  ├── list.go          → Query duplicates from DB (no filesystem access)
  ├── clean.go         → Interactive/auto duplicate deletion
  ├── stats.go         → Database statistics
  └── config.go        → CLI config management (add/remove/list)

internal/
  ├── backup/          → Backup orchestration
  │   ├── engine.go    → Coordinates walker, hasher, copier
  │   ├── walker.go    → Directory traversal with exclusions
  │   └── copier.go    → Parallel file copying
  ├── hash/            → SHA256 hashing
  │   └── calculator.go → Worker pool pattern for parallel hashing
  ├── duplicate/       → Duplicate detection
  │   ├── detector.go  → Multi-stage: size filter → hash grouping
  │   └── cleaner.go   → File deletion with strategies
  ├── index/           → SQLite database
  │   ├── db.go        → CRUD operations, migrations
  │   └── models.go    → FileRecord, DuplicateGroup
  └── config/          → Viper-based configuration
      └── config.go    → Load, validate, defaults

pkg/logger/            → slog wrapper
```

### Key Architectural Patterns

**1. Shared State in root.go**
- `cfg`, `db`, `log` are package-level vars in `cmd/`
- `PersistentPreRunE` initializes these for all commands
- Some commands (config add/remove/list) skip config validation

**2. Parallel Processing**
- Hash calculator: Worker pool with channels (`HashFiles` returns `<-chan HashResult`)
- File copier: Similar worker pool pattern
- Both use `runtime.NumCPU()` as default worker count

**3. Multi-Stage Duplicate Detection**
- Stage 1: Load all files from DB
- Stage 2: Group by size (in-memory, fast pre-filter)
- Stage 3: For duplicate sizes, group by hash
- Stage 4: Build DuplicateGroup structs
- Stage 5: Sort by wasted space

**4. Database Design**
- `files` table: path (unique), hash, size, mod_time, backup_set
- Indexed on: hash, size, backup_set
- `duplicate_groups` view for fast queries
- `list` command queries DB only (no filesystem access)
- `scan` command walks filesystem and updates DB

**5. Config Management**
- Two approaches: CLI (`config add/remove`) or YAML editing
- `config add/remove` work without validation (paths may not exist yet)
- `config list` loads YAML directly, skips validation
- Other commands require valid config (paths must exist)

## Important Implementation Details

### Adding New Commands

1. Create file in `cmd/` (e.g., `cmd/newcmd.go`)
2. Use `rootCmd.AddCommand(newCmd)` in `init()`
3. Access shared state: `cfg`, `db`, `log` (initialized by PersistentPreRunE)
4. If command needs to skip config loading, update `root.go` skip logic

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
- `config add/remove/list` bypass this by working with raw YAML
- This allows adding backup sets before creating source directories

## Testing Strategy

- Unit tests in `*_test.go` files (see `internal/hash/calculator_test.go`)
- Use `t.TempDir()` for temporary files
- Test files use table-driven tests where applicable
- Integration tests would need temporary SQLite database

## Configuration Files

- Default: `~/.m_backuper/config.yaml`
- Override: `--config` flag or `./config.yaml`
- Example: `configs/config.example.yaml`
- Database: `~/.m_backuper/index.db` (default)

## Common Gotchas

1. **`list` doesn't scan filesystem** - it only queries the database. Use `scan` first to index files.
2. **Config commands skip validation** - `config add` allows non-existent paths. They're validated when running `backup`.
3. **PersistentPreRunE shared state** - All commands share `cfg`, `db`, `log` initialized in root.go. Don't re-initialize.
4. **Worker pool channels** - Jobs channel is sent to workers, results channel is returned. Close jobs, wait for workers, then close results.
5. **Backup engine flow** - The backup process is sequential: walk → hash → copy → index. Each stage feeds the next.

## Dependencies

Key external packages:
- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration
- `github.com/mattn/go-sqlite3` - SQLite (requires GCC)
- `github.com/dustin/go-humanize` - Human-readable formats
- `gopkg.in/yaml.v3` - YAML parsing (for config CLI commands)

Standard library: `crypto/sha256`, `log/slog`, `database/sql`, `path/filepath`
