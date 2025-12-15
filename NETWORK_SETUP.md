# Network Share Setup Guide

m_backuper uses your operating system's native SMB/CIFS support by mounting network shares as local directories. This approach avoids dependencies and provides better performance.

## Linux

### Install CIFS utilities (if not already installed)
```bash
# Debian/Ubuntu
sudo apt-get install cifs-utils

# Fedora/RHEL
sudo dnf install cifs-utils

# Arch
sudo pacman -S cifs-utils
```

### Mount SMB Share
```bash
# Create mount point
sudo mkdir -p /mnt/backup

# Mount the share
sudo mount -t cifs //192.168.1.100/backups /mnt/backup \
  -o username=YOUR_USERNAME,password=YOUR_PASSWORD,uid=$(id -u),gid=$(id -g)
```

### Permanent Mount (Optional)
Add to `/etc/fstab`:
```bash
//192.168.1.100/backups /mnt/backup cifs username=YOUR_USERNAME,password=YOUR_PASSWORD,uid=1000,gid=1000,_netdev 0 0
```

For better security, use a credentials file:
```bash
# Create credentials file
sudo nano /etc/samba-credentials
```

Add:
```
username=YOUR_USERNAME
password=YOUR_PASSWORD
```

Secure it:
```bash
sudo chmod 600 /etc/samba-credentials
```

Update `/etc/fstab`:
```bash
//192.168.1.100/backups /mnt/backup cifs credentials=/etc/samba-credentials,uid=1000,gid=1000,_netdev 0 0
```

### Configure m_backuper
```json
{
  "backup_root": "/mnt/backup/m_backuper",
  "device_id": "my-linux-pc",
  "paths_to_backup": ["/home/user/Documents", "/home/user/Photos"]
}
```

## Android (Termux)

### Prerequisites
```bash
# Install required packages
pkg install root-repo
pkg install tsu cifs-utils
```

### Mount SMB Share
**Note**: Mounting network shares on Android typically requires root access.

#### Method 1: With Root Access
```bash
# Create mount point
mkdir -p ~/mnt/backup

# Mount using tsu (Termux su)
tsu
mount -t cifs //192.168.1.100/backups /data/data/com.termux/files/home/mnt/backup \
  -o username=YOUR_USERNAME,password=YOUR_PASSWORD,uid=$(id -u),gid=$(id -g)
exit
```

#### Method 2: Without Root (Use App)
If you don't have root access, use an SMB file manager app like:
- **Solid Explorer** - Mount SMB shares as accessible directories
- **CX File Explorer** - Built-in network support
- **MiXplorer** - Supports SMB/CIFS mounting

Then configure m_backuper to backup to the app's mounted directory (usually under `/storage/emulated/0/...`).

#### Method 3: Use Android Storage Access Framework
Some apps (like Solid Explorer) expose mounted network shares through Android's Storage Access Framework at paths like:
```
/storage/emulated/0/Android/data/pl.solidexplorer2/files/
```

### Configure m_backuper
```json
{
  "backup_root": "/data/data/com.termux/files/home/mnt/backup/m_backuper",
  "device_id": "my-android-phone",
  "paths_to_backup": [
    "/storage/emulated/0/DCIM",
    "/storage/emulated/0/Documents"
  ]
}
```

## Windows

### Method 1: Map Network Drive (GUI)
1. Open File Explorer
2. Click "This PC" in the left sidebar
3. Click "Map network drive" in the toolbar
4. Choose a drive letter (e.g., Z:)
5. Enter the network path: `\\192.168.1.100\backups`
6. Check "Reconnect at sign-in" for persistence
7. Enter credentials if prompted
8. Click "Finish"

### Method 2: Command Line
```cmd
REM Map network drive
net use Z: \\192.168.1.100\backups /user:YOUR_USERNAME YOUR_PASSWORD

REM Make it persistent (reconnect at login)
net use Z: \\192.168.1.100\backups /user:YOUR_USERNAME YOUR_PASSWORD /persistent:yes
```

### Method 3: PowerShell
```powershell
# Map network drive
New-PSDrive -Name "Z" -PSProvider FileSystem -Root "\\192.168.1.100\backups" -Persist

# With credentials
$username = "YOUR_USERNAME"
$password = ConvertTo-SecureString "YOUR_PASSWORD" -AsPlainText -Force
$credential = New-Object System.Management.Automation.PSCredential($username, $password)
New-PSDrive -Name "Z" -PSProvider FileSystem -Root "\\192.168.1.100\backups" -Credential $credential -Persist
```

### Configure m_backuper
```json
{
  "backup_root": "Z:\\m_backuper",
  "device_id": "my-windows-pc",
  "paths_to_backup": ["C:\\Users\\YourName\\Documents", "C:\\Users\\YourName\\Pictures"]
}
```

**Note**: Use double backslashes in JSON for Windows paths.

## Troubleshooting

### Connection Issues
```bash
# Test SMB connectivity (Linux/Termux)
smbclient -L //192.168.1.100 -U YOUR_USERNAME

# Check if share is accessible
smbclient //192.168.1.100/backups -U YOUR_USERNAME
```

### Permission Issues
If you get permission errors, check:
- UID/GID in mount options match your user
- SMB share permissions allow write access
- Firewall isn't blocking SMB ports (445, 139)

### Unmounting (Linux/Android)
```bash
sudo umount /mnt/backup
```

## Security Best Practices

1. **Use credentials file** instead of putting passwords in fstab
2. **Restrict credentials file permissions**: `chmod 600`
3. **Use SMB3** protocol when possible (more secure than SMB1)
4. **Consider VPN** for remote access instead of exposing SMB to internet
5. **Use strong passwords** for SMB authentication

## Alternative: Local Backup First, Then Sync

If network mounting is problematic, you can:
1. Backup to a local directory
2. Use rsync, robocopy, or rclone to sync to network share periodically

```bash
# Linux: Backup locally first
m_backuper backup

# Then sync to network share
rsync -av ~/.local/backup/ /mnt/network-backup/
```
