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

### Quick Start

1. Generate default config:
   ```bash
   m_backuper init
   ```

2. Edit `~/.config/m_backuper/config.json`:
   ```json
   {
     "backup_root": "/path/to/backup/destination",
     "device_id": "my-device",
     "paths_to_backup": ["/path/to/important/files"],
     "files_to_ignore_patterns": ["*.tmp", ".cache/*"]
   }
   ```

3. Run backup:
   ```bash
   m_backuper backup
   ```

### Network Storage (SMB/CIFS)

m_backuper uses your OS's native SMB support by mounting network shares as local directories. This provides better performance and avoids external dependencies.

**See [NETWORK_SETUP.md](NETWORK_SETUP.md) for detailed instructions on:**
- Mounting SMB shares on Linux
- Mounting SMB shares on Android (Termux)
- Mapping network drives on Windows
- Troubleshooting and security best practices

Once mounted, simply configure `backup_root` to point to the mounted location:
- Linux: `/mnt/backup/m_backuper`
- Android: `/data/data/com.termux/files/home/mnt/backup/m_backuper`
- Windows: `Z:\\m_backuper`

See spec.md for detailed configuration options.
