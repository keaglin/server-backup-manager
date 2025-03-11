# Server Backup Manager

A Go application for managing server backups to S3-compatible storage.

## Features

- Automated backup scheduling using cron expressions
- S3-compatible storage backend (AWS S3, Minio, Hetzner Storage Box, etc.)
- Configurable retention policy for old backups
- File logging
- Command-line flags and environment variables for configuration
- One-time backup mode
- Disk space monitoring with webhook alerts

## Getting Started

### Prerequisites

- Go 1.21 or higher

### Installation

```bash
git clone https://github.com/user/server-backup-manager.git
cd server-backup-manager
go build
```

### Usage

```bash
# Run with default settings
./server-backup-manager

# Run with custom settings
./server-backup-manager --backup-dir=/path/to/backups --bucket-name=my-backups --retention-days=30

# Run once and exit
./server-backup-manager --run-once

# Show help
./server-backup-manager --help
```

### Environment Variables

The application can be configured using environment variables:

#### Backup Configuration
- `BACKUP_DIR`: Directory containing backup files
- `BUCKET_NAME`: S3 bucket name
- `BUCKET_ENDPOINT`: S3 endpoint URL
- `ACCESS_KEY_ID`: S3 access key ID
- `SECRET_ACCESS_KEY`: S3 secret access key
- `USE_SSL`: Use SSL for S3 connections (true/false)
- `RETENTION_DAYS`: Number of days to retain backups
- `UPLOAD_SCHEDULE`: Cron schedule for backups
- `RUN_ONCE`: Run backup once and exit (true/false)

#### Monitoring Configuration
- `ENABLE_MONITORING`: Enable disk space monitoring (true/false)
- `MONITOR_PATHS`: Comma-separated list of paths to monitor for disk space
- `DISK_THRESHOLD`: Disk usage threshold percentage for alerts (default: 75.0)
- `MONITOR_INTERVAL`: Interval for disk space checks (e.g., "30m" for 30 minutes)
- `WEBHOOK_URLS`: Comma-separated list of webhook URLs for alerts
- `WEBHOOK_TIMEOUT`: Timeout for webhook requests (e.g., "10s" for 10 seconds)

## Disk Space Monitoring

The application can monitor disk space usage and send alerts when usage exceeds a configured threshold (default: 75%).

### Webhook Alerts

When disk space usage exceeds the threshold, the application sends a webhook alert with the following JSON payload:

```json
{
  "type": "disk_space_alert",
  "timestamp": "2023-03-11T12:34:56Z",
  "source": "server-backup-manager",
  "message": "Disk space alert: /path/to/backups is at 80.25% (threshold: 75.00%)",
  "data": {
    "path": "/path/to/backups",
    "total_bytes": 1000000000,
    "used_bytes": 802500000,
    "free_bytes": 197500000,
    "used_percent": 80.25,
    "timestamp": "2023-03-11T12:34:56Z",
    "is_alert": true
  }
}
```

You can configure multiple webhook endpoints to receive these alerts.

## Project Structure

```
server-backup-manager/
├── cmd/                    # Command-line applications
│   └── server-backup-manager/  # Main application
├── internal/               # Private application code
│   ├── backup/             # Backup management
│   ├── config/             # Configuration handling
│   ├── monitoring/         # Disk space monitoring
│   ├── notification/       # Alert notifications
│   └── storage/            # Storage backends
├── pkg/                    # Public libraries
│   └── utils/              # Utility functions
├── go.mod                  # Go module definition
├── main.go                 # Application entry point
└── README.md               # This file
```

## License

This project is licensed under the MIT License - see the LICENSE file for details. 