# Server Backup Manager Deployment Guide

This guide will help you deploy the server-backup-manager to your server.

## Prerequisites

- Linux server (Ubuntu/Debian recommended)
- Root or sudo access
- Cloudflare R2 account (or compatible S3 storage)

## Installation Steps

1. Transfer the deployment package to your server:
   ```bash
   scp server-backup-manager-*.tar.gz user@your-server:/tmp/
   ```

2. SSH into your server:
   ```bash
   ssh user@your-server
   ```

3. Extract the package:
   ```bash
   sudo mkdir -p /opt/server-backup-manager
   sudo tar -xzf /tmp/server-backup-manager-*.tar.gz -C /opt/server-backup-manager
   cd /opt/server-backup-manager
   ```

4. Run the setup script:
   ```bash
   sudo ./setup.sh
   ```
   
   This script will:
   - Prompt for your R2 credentials and backup settings
   - Create the environment configuration file at `/etc/default/server-backup-manager`
   - Set up a systemd service for automatic operation

5. Start the service:
   ```bash
   sudo systemctl start server-backup-manager
   ```

6. Verify the service is running:
   ```bash
   sudo systemctl status server-backup-manager
   ```

## Configuration

The server-backup-manager uses environment variables for configuration. The setup script creates these variables in the file `/etc/default/server-backup-manager`.

### Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| BACKUP_DIR | Directory to back up | /home/ghost/backups |
| BUCKET_NAME | R2 bucket name | my-backups |
| BUCKET_ENDPOINT | R2 endpoint URL | xxx.r2.cloudflarestorage.com |
| ACCESS_KEY_ID | R2 access key ID | your-access-key |
| SECRET_ACCESS_KEY | R2 secret access key | your-secret-key |
| USE_SSL | Whether to use SSL | true |
| RETENTION_DAYS | Number of days to keep backups | 14 |
| UPLOAD_SCHEDULE | Cron schedule for backups | 0 2 * * * |
| ENABLE_MONITORING | Enable disk space monitoring | false |
| DISK_THRESHOLD | Disk usage threshold percentage | 85 |
| WEBHOOK_URLS | Comma-separated list of webhook URLs | https://example.com/webhook |

### Manual Configuration

If you need to manually configure the application:

1. Create or edit the environment file:
   ```bash
   sudo nano /etc/default/server-backup-manager
   ```

2. Add your configuration variables:
   ```
   BACKUP_DIR=/path/to/backup/directory
   BUCKET_NAME=your-bucket-name
   BUCKET_ENDPOINT=your-endpoint.r2.cloudflarestorage.com
   ACCESS_KEY_ID=your-access-key
   SECRET_ACCESS_KEY=your-secret-key
   USE_SSL=true
   RETENTION_DAYS=14
   UPLOAD_SCHEDULE="0 2 * * *"
   ENABLE_MONITORING=false
   ```

3. If monitoring is enabled, add:
   ```
   DISK_THRESHOLD=85
   WEBHOOK_URLS="https://webhook1.example.com,https://webhook2.example.com"
   ```

4. Restart the service:
   ```bash
   sudo systemctl restart server-backup-manager
   ```

## Backup Retention and Scheduling

- **Retention**: The `RETENTION_DAYS` setting controls how many days of backups are kept. Older backups are automatically deleted.
- **Scheduling**: The `UPLOAD_SCHEDULE` setting uses cron syntax to determine when backups run.

## Disk Space Monitoring

When enabled, the server-backup-manager will monitor disk space usage and send alerts when usage exceeds the threshold.

### Monitoring Configuration

- Set `ENABLE_MONITORING=true` to enable monitoring
- Set `DISK_THRESHOLD` to the percentage at which alerts should be triggered (e.g., 85)
- Configure at least one webhook URL in `WEBHOOK_URLS` for notifications

### Webhook Configuration

When monitoring is enabled, at least one webhook URL is required. Webhooks receive JSON payloads with disk usage information when thresholds are exceeded.

Example webhook payload:
```json
{
  "event": "disk_space_alert",
  "timestamp": "2023-03-11T15:04:05Z",
  "path": "/home/ghost/backups",
  "usage_percent": 87.5,
  "threshold_percent": 85.0
}
```

## Troubleshooting

If you encounter issues:

1. Check the service status:
   ```bash
   sudo systemctl status server-backup-manager
   ```

2. View logs:
   ```bash
   sudo journalctl -u server-backup-manager
   ```

3. Verify your environment configuration:
   ```bash
   cat /etc/default/server-backup-manager
   ```

4. For more detailed troubleshooting, refer to the [TROUBLESHOOTING.md](TROUBLESHOOTING.md) file.

## Updating

To update the server-backup-manager:

1. Stop the service:
   ```bash
   sudo systemctl stop server-backup-manager
   ```

2. Replace the executable:
   ```bash
   sudo cp new-server-backup-manager /opt/server-backup-manager/server-backup-manager
   ```

3. Restart the service:
   ```bash
   sudo systemctl start server-backup-manager
   ```

## Special Operation Modes

### Initialization Mode

If you have existing backups that you want to upload to R2, you can use the initialization mode:

1. Edit the environment file to enable initialization mode:
   ```bash
   sudo nano /etc/default/server-backup-manager
   ```

2. Add or modify the following line:
   ```
   INITIALIZE=true
   ```

3. Run the application manually:
   ```bash
   sudo /opt/server-backup-manager/server-backup-manager
   ```

4. After initialization is complete, set `INITIALIZE=false` or remove the line to prevent re-initialization on the next run.

### One-Time Backup Mode

If you want to run a single backup and then exit (instead of running as a service):

1. Edit the environment file:
   ```bash
   sudo nano /etc/default/server-backup-manager
   ```

2. Add or modify the following line:
   ```
   RUN_ONCE=true
   ```

3. Run the application manually:
   ```bash
   sudo /opt/server-backup-manager/server-backup-manager
   ``` 