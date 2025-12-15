ckuper

Incremental backup tool for backing up selected drives and files to network
storage.

## Features
• Cross-platform
  • Windows
  • Linux
  • Android (via Termux)
  • iPhone (future)
• Incremental backups (size-based change detection)
• Manually triggered execution
• Network share support (SMB)
• Configurable ignore patterns
• Per-device state tracking

## Technical

### Stack

• golang
• `github.com/hirochachacha/go-smb2` - SMB client library
• makefile with relevant commands
  • build
  • build-android
  • build-windows
  • test
  • run
  • install-termux

### Architecture

**Deployment Model**: Single executable per platform, manually triggered, direct
push to network share

**Components**:
  - ```
  ┌─────────┐     ┌──────────┐     ┌────────────┐     ┌─────────┐
│ Scanner │────▶│Comparator│────▶│   Copier   │────▶│  State  │
└─────────┘     └──────────┘     └────────────┘     └─────────┘
                     ▲
                     │ (pluggable strategy)
```

• **Config Loader**: Load from JSON file + environment variable overrides
• **Scanner**: Walk configured paths, apply ignore patterns
• **Comparator**: Pluggable interface for change detection (size-based
initially)
• **Copier**: Transfer files to network share via SMB
• **State Manager**: Persist/load backup state locally

**Change Detection Interface** (pluggable):
  - ```go
  - type ChangeDetector interface {
  -     HasChanged(path string, info fs.FileInfo, state FileState) bool
  -     }
  -     ```

Initial implementation: `SizeDetector` (compare file size only)
Future: `ModTimeDetector`, `HashDetector`, or composite strategies

**Directory Structure**:
  - ```
  <backup_root>/
├── <device_id_1>/
│   └── <mirrored paths>
└── <device_id_2>/
    └── <mirrored paths>
```

### Design/code principles

• Test all non-trivial public functions in go packages
• If fixing a bug, add a test first that fails, then add the fix to pass the
test
• Less is more. Try to minimize boilerplate
• KISS, keep it simple
• Design for extensibility at strategy points (change detection, copying)

## Deployable

### Executable

Single executable that directly copies files to network share via SMB protocol.
No server process required.

**CLI Commands**:
  - ```bash
  - m_backuper backup    # Run backup
  - m_backuper status    # Show last backup time, file count
  - m_backuper config    # Show current config (merged file + env)
  - m_backuper init      # Generate default config file
  - ```

### Configuration

**Config File Location**: `~/.config/m_backuper/config.json`

**State File Location**: `~/.config/m_backuper/state.json` (local only)

#### Client
```json
{
    "backup_root": "//192.168.1.100/backups/m_backuper",
  "device_id": "marcus-phone",
  "paths_to_backup": [
        "/storage/emulated/0/DCIM",
        "/storage/emulated/0/Documents"
      ],
  "files_to_ignore_patterns": [
        "*.tmp",
        ".cache/*",
        "**/node_modules/**"
      ]
}
  ```

#### State File
```json
{
    "last_run": "2025-12-15T10:30:00Z",
  "files": {
        "/storage/emulated/0/DCIM/photo.jpg": {
      "size": 2048576,
      "backed_up": "2025-12-15T10:30:00Z"
          }
        }
}
  ```

#### Environment Variables (override config)
```bash
M_BACKUPER_SMB_USER=username
M_BACKUPER_SMB_PASS=password
M_BACKUPER_BACKUP_ROOT=//nas/backups
```

## Implementation Plan

Each task results in a buildable executable with passing tests. **Logging (via
slog) and error handling implemented from step 1 onwards.**

### 1. Project Structure + CLI Skeleton + Logging
**Deliverable**: Executable with subcommands, slog logging, error handling
*```
*m_backuper/
*├── cmd/m_backuper/
*│   └── main.go
*├── go.mod
*├── Makefile
*└── README.md
*```
***Tests**: None (structure + slog setup only)
***Build**: `make build` produces executable
***Run**: `./m_backuper backup` prints "backup: not implemented" with slog
output

### 2. Config Package
**Deliverable**: Load config from file + environment variable overrides with
error handling
```
internal/config/
├── config.go
└── config_test.go
```
**Tests**:
  - - Load from valid JSON file
  - - Environment variable override
  - - Missing file returns defaults
  - - Invalid JSON returns clear error with slog logging
  - **CLI**: `m_backuper config` prints merged configuration
  - **CLI**: `m_backuper init` creates default config file

### 3. Scanner Package
**Deliverable**: Walk filesystem paths, collect file info, apply ignore patterns
with error handling
```
internal/scanner/
├── scanner.go
└── scanner_test.go
```
**Tests**:
  - - Scan directory returns file list
  - - Ignore patterns work (glob matching)
  - - Nested directories handled
  - - Permission errors logged via slog and handled gracefully
  - - Symlinks handled appropriately
  - **CLI**: `m_backuper backup --dry-run` prints files that would be backed up
  with progress logging

### 4. Detector Package (Change Detection)
**Deliverable**: Pluggable interface + size-based implementation
*```
*internal/detector/
*├── detector.go       # interface
*├── size.go          # SizeDetector implementation
*└── detector_test.go
*```
***Tests**:
  - - SizeDetector.HasChanged returns true when size differs
  - - SizeDetector.HasChanged returns false when size matches
  - - Interface can be swapped
  - **CLI**: No change (used internally in next step)

### 5. State Package
**Deliverable**: Load/save backup state to local JSON file with error handling
*```
*internal/state/
*├── state.go
*└── state_test.go
*```
***Tests**:
  - - Save state to file
  - - Load state from file
  - - Handle missing state file (first run)
  - - Handle corrupted state file with clear error
  - - File write errors handled and logged via slog
  - **CLI**: `m_backuper status` reads and displays state

### 6. Copier Package (Local Filesystem First)
**Deliverable**: Copy files to destination with error handling and progress
logging
```
internal/copier/
├── copier.go       # interface
├── local.go        # LocalCopier for testing
└── copier_test.go
```
**Tests**:
  - - Copy file to destination
  - - Preserve directory structure
  - - Handle copy errors (permissions, disk full, etc.)
  - - Create destination directories as needed
  - - Log copy progress via slog
  - **CLI**: `m_backuper backup` works end-to-end with local destination, shows
  progress

### 7. Wire Everything Together
**Deliverable**: Complete backup flow with error recovery
*```
*internal/backup/
*├── backup.go      # orchestrates scanner, detector, copier, state
*└── backup_test.go
*```
***Tests**:
  - - Full backup flow (integration test with temp directories)
  - - Incremental backup only copies changed files
  - - State updates after successful backup
  - - Partial failures handled gracefully
  - - State not corrupted on interrupted backup
  - **CLI**: `m_backuper backup` performs complete local backup with error
  recovery

### 8. SMB Copier Implementation
**Deliverable**: Replace local copier with SMB network share support
*```
*internal/copier/
*├── smb.go         # SMBCopier using go-smb2
*└── smb_test.go    # may require test SMB server or mocking
*```
***Tests**:
  - - SMB connection with credentials
  - - Copy file over SMB
  - - Handle network errors (timeout, connection lost, auth failure)
  - - Retry logic for transient failures
  - - (May use mocked SMB for tests)
  - **CLI**: `m_backuper backup` works with network share with network error
  handling

### 9. Cross-Platform Builds
**Deliverable**: Build targets for all platforms
*```
*Makefile additions:
  - - build-android
  - - build-windows
  - - build-linux
  - ```
  **Tests**: Existing tests pass on all platforms
  ***Build**: `make build-android` produces ARM64 binary

### Testing Strategy Per Task
- Unit tests for all public functions
- - Integration tests where components interact
- - Manual CLI testing for user-facing changes
- - Error paths tested explicitly
- - `make test` must pass before moving to next task
- - `make build` must produce working executable
