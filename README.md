# Server Backup Manager

A Go application for managing server backups to S3-compatible storage.

## Features

- Automated backup scheduling using cron expressions
- S3-compatible storage backend (AWS S3, Minio, Hetzner Storage Box, etc.)
- Configurable retention policy for old backups
- File logging
- Command-line flags and environment variables for configuration
- One-time backup mode

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

- `BACKUP_DIR`: Directory containing backup files
- `BUCKET_NAME`: S3 bucket name
- `BUCKET_ENDPOINT`: S3 endpoint URL
- `ACCESS_KEY_ID`: S3 access key ID
- `SECRET_ACCESS_KEY`: S3 secret access key
- `USE_SSL`: Use SSL for S3 connections (true/false)
- `RETENTION_DAYS`: Number of days to retain backups
- `UPLOAD_SCHEDULE`: Cron schedule for backups
- `RUN_ONCE`: Run backup once and exit (true/false)

## Project Structure

```
server-backup-manager/
├── cmd/                    # Command-line applications
│   └── server-backup-manager/  # Main application
├── internal/               # Private application code
│   ├── backup/             # Backup management
│   ├── config/             # Configuration handling
│   └── storage/            # Storage backends
├── pkg/                    # Public libraries
│   └── utils/              # Utility functions
├── go.mod                  # Go module definition
├── main.go                 # Application entry point
└── README.md               # This file
```

## License

This project is licensed under the MIT License - see the LICENSE file for details. 