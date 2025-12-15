# m_backuper

Incremental backup tool for backing up selected drives and files to network storage.

## Features

- Cross-platform (Windows, Linux, Android via Termux)
- Incremental backups (size-based change detection)
- Manually triggered execution
- Network share support (SMB)
- Configurable ignore patterns
- Per-device state tracking

## Building

```bash
# Build for current platform
make build

# Build for Android
make build-android

# Build for Windows
make build-windows

# Build for Linux
make build-linux

# Install on Termux
make install-termux
```

## Testing

```bash
make test
```

## Usage

```bash
# Run backup
m_backuper backup

# Show status
m_backuper status

# Show current config
m_backuper config

# Generate default config
m_backuper init
```

## Configuration

Config file location: `~/.config/m_backuper/config.json`

See spec.md for detailed configuration options.
