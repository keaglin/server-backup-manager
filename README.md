# Server Backup Manager

A Go application for managing server backups to Cloudflare R2 or compatible S3 storage.

## Features

- Automated backups to Cloudflare R2 (or any S3-compatible storage)
- Configurable backup schedule using cron syntax
- Automatic cleanup of old backups based on retention policy
- Disk space monitoring with webhook notifications
- Systemd integration for reliable operation

## Configuration

The server-backup-manager uses environment variables for configuration. These can be set in the file `/etc/default/server-backup-manager` which is read by the systemd service.

Key environment variables:

- `BACKUP_DIR`: Directory to back up
- `BUCKET_NAME`: R2 bucket name
- `BUCKET_ENDPOINT`: R2 endpoint URL
- `ACCESS_KEY_ID`: R2 access key ID
- `SECRET_ACCESS_KEY`: R2 secret access key
- `ENABLE_MONITORING`: Enable disk space monitoring (true/false)
- `WEBHOOK_URLS`: Comma-separated list of webhook URLs (required if monitoring is enabled)

For a complete list of configuration options, see the [Deployment Guide](DEPLOY.md).

## Deployment

See the [Deployment Guide](DEPLOY.md) for detailed instructions on how to deploy the server-backup-manager to your server.

## Troubleshooting

If you encounter issues with the server-backup-manager, refer to the [Troubleshooting Guide](TROUBLESHOOTING.md) for common problems and solutions.

## Development

### Building from Source

```bash
go build -o server-backup-manager .
```

### Creating a Deployment Package

```bash
./package.sh
```

This will create a tarball containing the executable and all necessary files for deployment.

## License

[MIT License](LICENSE) 