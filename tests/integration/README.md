# Integration Tests

This directory contains integration tests that verify m_backuper works correctly with real SMB/CIFS network shares.

## Overview

The integration tests validate:
- Full backup flow to SMB shares
- Incremental backups over the network
- Ignore patterns on network storage
- Mount point validation
- Cross-platform path handling

## Running Tests

### Option 1: Docker (Recommended)

The easiest way to run integration tests is using Docker Compose, which sets up a Samba server automatically:

```bash
make test-integration-docker
```

This will:
1. Start a Samba server in Docker
2. Build a test container with Go and CIFS utilities
3. Mount the SMB share inside the container
4. Run all integration tests
5. Clean up containers

**Requirements:**
- Docker and Docker Compose installed

### Option 2: Local SMB Share

If you have an existing SMB share mounted:

```bash
SMB_MOUNT=/mnt/my-smb-share make test-integration
```

**Requirements:**
- SMB share mounted and accessible
- Write permissions on the mount
- CIFS utilities installed (for mounting)

### Option 3: Manual Setup

1. **Start a Samba server** (if you don't have one):
   ```bash
   docker run -d --name samba-test \
     -p 445:445 \
     -e USER="testuser;testpass" \
     -e SHARE="backup;/backup;yes;no;no;testuser" \
     dperson/samba:latest
   ```

2. **Mount the share**:

   Linux:
   ```bash
   sudo mkdir -p /mnt/smb-test
   sudo mount -t cifs //localhost/backup /mnt/smb-test \
     -o user=testuser,password=testpass,vers=3.0
   ```

   macOS:
   ```bash
   sudo mkdir -p /Volumes/smb-test
   sudo mount_smbfs //testuser:testpass@localhost/backup /Volumes/smb-test
   ```

   Windows:
   ```powershell
   net use Z: \\localhost\backup /user:testuser testpass
   ```

3. **Run tests**:
   ```bash
   SMB_MOUNT=/mnt/smb-test make test-integration
   ```

4. **Cleanup**:
   ```bash
   sudo umount /mnt/smb-test
   docker stop samba-test && docker rm samba-test
   ```

## Test Structure

### Tests Included

| Test | Description |
|------|-------------|
| `TestSMBBackupFullFlow` | Complete backup workflow to SMB share |
| `TestSMBBackupIncremental` | Verify only changed files are re-backed up |
| `TestSMBBackupWithIgnorePatterns` | Test pattern matching on network storage |
| `TestSMBMountValidation` | Verify mount accessibility and permissions |

### Build Tags

Integration tests use the `integration` build tag:

```go
//go:build integration
```

This prevents them from running during normal `go test ./...` to avoid requiring SMB mounts for development.

## CI/CD

Integration tests run automatically in GitHub Actions:

- **Docker workflow**: Uses Docker Compose (runs on all PRs)
- **Multi-platform workflow**: Tests on Ubuntu and macOS with real mounts

See `.github/workflows/integration-tests.yml` for details.

## Troubleshooting

### "SMB_MOUNT environment variable not set"
Set the environment variable to point to your mounted SMB share:
```bash
export SMB_MOUNT=/mnt/smb-test
make test-integration
```

### "path does not exist (is network drive mounted?)"
Verify the mount exists and is accessible:
```bash
ls -la $SMB_MOUNT
mount | grep $SMB_MOUNT
```

### "path is not writable"
Check permissions on the SMB share:
```bash
touch $SMB_MOUNT/test.txt
```

If this fails, verify:
- SMB user has write permissions
- Mount options include write access (not `ro`)
- SELinux/AppArmor isn't blocking writes

### Docker Compose hangs
Check if containers are healthy:
```bash
docker-compose -f docker-compose.test.yml ps
docker-compose -f docker-compose.test.yml logs samba
```

Clean up and retry:
```bash
make clean-test
make test-integration-docker
```

### macOS "Operation not permitted"
macOS may require additional permissions:
```bash
# Grant Terminal full disk access in System Preferences > Security & Privacy
# Or run with sudo (not recommended)
```

## Platform-Specific Notes

### Linux
- Uses `cifs-utils` for mounting
- Requires root/sudo for mounting
- SMB version 3.0 is used by default

### macOS
- Built-in SMB support via `mount_smbfs`
- Requires admin password for mounting
- Mount paths typically under `/Volumes/`

### Windows
- Use `net use` for mapping drives
- UNC paths: `\\server\share`
- PowerShell may require admin rights

### Android (Termux)
- Limited SMB mounting support
- May need additional packages
- Recommend using Docker method instead

## Development

To add new integration tests:

1. Create test file in `tests/integration/`
2. Add `//go:build integration` build tag
3. Use `os.Getenv("SMB_MOUNT")` for mount path
4. Include cleanup with `defer` or `t.Cleanup()`
5. Test locally with `make test-integration-docker`

Example:
```go
//go:build integration

package integration

import (
    "os"
    "testing"
)

func TestNewFeature(t *testing.T) {
    smbMount := os.Getenv("SMB_MOUNT")
    if smbMount == "" {
        t.Fatal("SMB_MOUNT not set")
    }

    // Your test here
}
```

## Resources

- [Samba Docker Image](https://hub.docker.com/r/dperson/samba)
- [CIFS Kernel Module](https://www.kernel.org/doc/html/latest/admin-guide/cifs/usage.html)
- [SMB Protocol](https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-smb)
