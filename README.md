# Server Backup Manager

A memory-efficient tool for backing up server data to Cloudflare R2 storage with resumable uploads.

## Features

- **Memory-efficient uploads**: Designed to use minimal memory to avoid OOM killer
- **Resumable uploads**: Can resume interrupted uploads without re-uploading already transferred files
- **Automatic archiving**: Files older than 14 days are automatically moved to an `archive/` prefix
- **Scheduled backups**: Configure backup schedules using cron expressions
- **Disk space monitoring**: Optional monitoring of disk space with webhook alerts

## Installation

For detailed deployment instructions, especially for running as a systemd service, see the [Deployment Guide](.deploy/DEPLOY.md).

### Prerequisites

- Go 1.18 or higher
- Access to Cloudflare R2 storage

### Building from source

```bash
git clone https://github.com/keaglin/server-backup-manager.git
cd server-backup-manager
go build -o server-backup-manager
```

### Installing as a service

1. Copy the binary to a system location:
   ```bash
   sudo cp server-backup-manager /usr/local/bin/
   ```

2. Copy the systemd service file:
   ```bash
   sudo cp server-backup-manager.service /etc/systemd/system/
   ```

3. Reload systemd and enable the service:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable server-backup-manager
   ```

## Configuration

The application is primarily configured using environment variables. Command-line flags are also available, which might be more convenient for one-time runs.

When deploying as a service using the method described in the [Deployment Guide](.deploy/DEPLOY.md), configuration should be placed in `/etc/default/server-backup-manager`.

### Required Configuration

| Environment Variable | Command-line Flag | Description |
|---|---|---|
| `BACKUP_DIR` | `--backup-dir` | Directory containing backup files |
| `BUCKET_NAME` | `--bucket-name` | R2 bucket name |
| `BUCKET_ENDPOINT` | `--bucket-endpoint` | R2 endpoint URL (e.g., `accountid.r2.cloudflarestorage.com`) |
| `ACCESS_KEY_ID` | `--access-key` | R2 access key ID |
| `SECRET_ACCESS_KEY` | `--secret-key` | R2 secret access key |
| `USE_SSL` | `--use-ssl` | Use SSL for R2 connection (default: `true`) |


### Optional Configuration

| Environment Variable | Command-line Flag | Description | Default |
|---|---|---|---|
| `UPLOAD_SCHEDULE` | `--schedule` | Cron schedule for backups | `"0 2 * * *"` |
| `RETENTION_DAYS` | `--retention-days` | Days before files are moved to `archive/` prefix | `14` |
| `RUN_ONCE` | `--run-once` | Run backup once and exit | `false` |
| `INITIALIZE` | `--initialize` | Initialize mode for resumable uploads | `false` |
| `ENABLE_MONITORING` | `--enable-monitoring` | Enable disk space monitoring | `false` |
| `DISK_THRESHOLD` | `--disk-threshold` | Disk usage percentage threshold for monitoring alerts | `85` |
| `WEBHOOK_URLS` | `--webhook-urls` | Comma-separated list of webhook URLs for monitoring alerts | `""` |


## Usage

### Running a one-time backup

```bash
server-backup-manager --run-once \
  --backup-dir=/path/to/backups \
  --bucket-name=your-bucket \
  --bucket-endpoint=accountid.r2.cloudflarestorage.com \
  --access-key=your-access-key \
  --secret-key=your-secret-key
```

### Initializing with resumable uploads

If your upload was interrupted or you want to upload all existing backups:

```bash
server-backup-manager --initialize \
  --backup-dir=/path/to/backups \
  --bucket-name=your-bucket \
  --bucket-endpoint=accountid.r2.cloudflarestorage.com \
  --access-key=your-access-key \
  --secret-key=your-secret-key
```

This will:
1. Scan all existing backup directories
2. Upload files that haven't been uploaded yet
3. Track progress so you can resume if interrupted

### Running as a service

Start the service:

```bash
sudo systemctl start server-backup-manager
```

Check status:

```bash
sudo systemctl status server-backup-manager
```

View logs:

```bash
sudo journalctl -u server-backup-manager -f
```

## Memory Management

The application is designed to be memory-efficient:

1. **Built-in memory limits**: The application sets a 1GB memory limit and aggressive garbage collection
2. **Systemd memory limits**: The systemd service file includes a 1GB memory limit
3. **Efficient file handling**: Files are processed in a streaming manner to minimize memory usage
4. **Resumable uploads**: Interrupted uploads can be resumed without re-uploading already transferred files

## Troubleshooting

If you encounter issues, check the application logs and consult the [Troubleshooting Guide](.deploy/TROUBLESHOOTING.md) for common problems and solutions.

### OOM Killer Issues

If the application is still being killed by the OOM killer:

1. Adjust the memory limit in the systemd service file:
   ```
   MemoryLimit=1.5G
   ```

2. Reduce concurrency by setting environment variables:
   ```
   GOMAXPROCS=2
   ```

### Resuming Interrupted Uploads

If an upload is interrupted, simply run the application again with the `--initialize` flag. It will:

1. Check which files have already been uploaded
2. Skip those files and continue with the remaining ones
3. Track progress as it goes

# License

[MIT License](LICENSE) 