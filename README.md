# m_backuper

CLI tool for backing up files and detecting duplicates using SHA256 content hashing.

## Features

- Full copy backups with parallel SHA256 hashing • SQLite-indexed duplicate detection • Multiple keep strategies (oldest/newest/first/shortest) • CLI config management • Database statistics • Dry-run mode • Exclusion patterns • Progress tracking

## Installation

**Prerequisites:** Go 1.20+, GCC (for SQLite)

```bash
git clone https://github.com/mackeper/m_backuper.git
cd m_backuper
go build -o m_backuper

# Optional: Install to PATH
go install github.com/mackeper/m_backuper@latest
```

## Quick Start

```bash
# 1. Add backup sets (no manual YAML editing!)
m_backuper config add photos --sources ~/Pictures --destination /mnt/backup/photos
m_backuper config add docs --sources ~/Documents --destination /mnt/backup/docs

# 2. Run backup
m_backuper backup photos --dry-run  # Preview
m_backuper backup photos            # Execute

# 3. Check what's indexed
m_backuper stats                    # View database stats

# 4. Find duplicates
m_backuper list --sort wasted       # Show duplicates
m_backuper clean --auto --keep oldest --dry-run  # Clean up
```

## Commands

| Command | Description | Key Flags |
|---------|-------------|-----------|
| `backup [name]` | Run backup operation | `--all`, `--dry-run` |
| `scan [path...]` | Index files and calculate hashes | `--min-size`, `--update-index` |
| `list` | Show duplicate groups | `--sort` (size/count/wasted), `--format` (table/json), `--min-wasted` |
| `clean` | Delete duplicates | `--auto`, `--keep` (oldest/newest/first/shortest), `--dry-run` |
| `stats` | Show database statistics | - |
| `config add <name>` | Add backup set to config | `--sources`, `--destination`, `--excludes` |
| `config remove <name>` | Remove backup set from config | - |
| `config list` | List all backup sets | - |
| `config show` | Display full config | - |
| `config validate` | Validate config file | - |
| `config init` | Create example config | - |

**Examples:**
```bash
m_backuper config add photos --sources ~/Pictures --destination /mnt/backup
m_backuper backup photos --dry-run           # Preview backup
m_backuper backup --all -v                   # Backup all with verbose output
m_backuper stats                             # View indexed files stats
m_backuper list --sort count --format json   # List as JSON sorted by count
m_backuper clean --auto --keep oldest        # Auto-delete, keep oldest files
m_backuper config remove photos              # Remove backup set
```

## Configuration

**Manage via CLI** (no manual editing needed):
```bash
m_backuper config add photos --sources ~/Pictures --destination /mnt/backup/photos
m_backuper config add docs --sources ~/Documents,~/Downloads --destination /mnt/backup/docs --excludes "*.tmp"
m_backuper config list              # View all backup sets
m_backuper config remove photos     # Remove a backup set
```

**Or edit YAML** at: `--config` flag → `~/.m_backuper/config.yaml` → `./config.yaml`

```yaml
backup_sets:
  - name: photos
    sources: [/path/to/source1, /path/to/source2]
    destination: /path/to/dest
    excludes: ["*.tmp", ".cache/**"]

duplicates:
  min_file_size: 1048576        # Bytes (1MB)
  scan_paths: [/mnt/backup]

concurrency:
  hash_workers: 0               # 0 = CPU count
  copy_workers: 2

database:
  path: ~/.m_backuper/index.db
```

## How It Works

**Backup:** Walk sources → Hash files (SHA256, parallel) → Copy to destination → Update SQLite index

**Duplicate Detection:** Load from DB → Group by size (pre-filter) → Group by hash → Sort by wasted space → Report

Multi-stage approach: Size filtering (no I/O) → Hash grouping (only potential duplicates) → Cached results

## Performance Tips

- Adjust `hash_workers` (higher for CPU-bound, mindful of I/O) • Set `min_file_size` to skip small files • Use exclusion patterns • More workers on SSD than HDD

## Use Cases

**Backup:**
```bash
m_backuper backup personal  # Config: ~/Documents, ~/Pictures → /mnt/backup
```

**Find duplicates across backups:**
```bash
m_backuper scan /mnt/backup1 /mnt/backup2
m_backuper list --sort wasted
m_backuper clean --auto --keep newest
```

**Cron scheduled backup:**
```bash
0 2 * * * /usr/local/bin/m_backuper backup --all -v >> /var/log/m_backuper.log 2>&1
```

**Database:** SQLite with `files` (path/hash/size/modtime), `backup_runs` (history), indexed on hash+size

## Troubleshooting

| Issue | Solution |
|-------|----------|
| "No configuration file found" | Run `m_backuper config init` or use `--config` flag |
| "Permission denied" | Check read permissions (sources) and write permissions (destination) |
| Slow hashing | Reduce `hash_workers` (HDD), increase `min_file_size`, check CPU usage |
| Database locked | Only one instance can write at a time - wait for other operations |

## Development

```bash
go test ./...                    # Run tests
go build -o m_backuper          # Build
./m_backuper backup photos -v   # Run with verbose logging
```

## Roadmap

- [ ] TUI (Bubble Tea) • [ ] Incremental backups • [ ] Compression/encryption • [ ] Cloud backends (S3, GDrive) • [ ] Backup scheduling • [ ] Web interface

## Credits

[Cobra](https://github.com/spf13/cobra) • [Viper](https://github.com/spf13/viper) • [go-sqlite3](https://github.com/mattn/go-sqlite3) • [go-humanize](https://github.com/dustin/go-humanize)

**Author:** Marcus Östling ([@mackeper](https://github.com/mackeper)) • **License:** MIT