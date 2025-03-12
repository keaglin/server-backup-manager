#!/bin/bash

# Load environment variables
if [ -f .env ]; then
    source .env
fi

# Default settings
DRY_RUN=false

# Parse command line arguments
while [[ "$#" -gt 0 ]]; do
    case $1 in
        -d|--dry-run) DRY_RUN=true ;;
        -h|--help)
            echo "Usage: $0 [options]"
            echo "Options:"
            echo "  -d, --dry-run    Show what would be backed up without actually copying"
            echo "  -h, --help       Show this help message"
            exit 0
            ;;
        *) echo "Unknown parameter: $1"; exit 1 ;;
    esac
    shift
done

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Error: Please run as root or with sudo"
    exit 1
fi

# Set backup directory
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/home/ghost/backups/${TIMESTAMP}"
LATEST_LINK="/home/ghost/backups/latest"

# Check if backup directory already exists
if [ -d "${BACKUP_DIR}" ]; then
    echo "Error: Backup directory already exists: ${BACKUP_DIR}"
    exit 1
fi

# Check available disk space
REQUIRED_SPACE=$(du -sb /var/lib/docker/volumes/{app_db_data,ghost_data,minio_data} 2>/dev/null | awk '{total += $1} END {print total}')
AVAILABLE_SPACE=$(df -B1 /home/ghost/backups | awk 'NR==2 {print $4}')

if [ $AVAILABLE_SPACE -lt $REQUIRED_SPACE ]; then
    echo "Error: Insufficient disk space"
    echo "Required: $(numfmt --to=iec-i $REQUIRED_SPACE)"
    echo "Available: $(numfmt --to=iec-i $AVAILABLE_SPACE)"
    exit 1
fi

# Create backup directory
if [ "$DRY_RUN" = false ]; then
    mkdir -p "${BACKUP_DIR}"
    mkdir -p "${BACKUP_DIR}/databases"
fi

# List of databases to backup
declare -A DATABASES=(
    # Format for MySQL: "mysql:database_name:user:password"
    ["ghost-db-1"]="mysql:ghost:root:${MYSQL_ROOT_PASSWORD}"
    # Format for PostgreSQL: "postgres:database_name:user"
    ["tenshu-db-1"]="postgres:tenshu:tenshu"
)

# Backup databases
echo "Creating database dumps..."
if [ "$DRY_RUN" = true ]; then
    echo "[DRY RUN] Would dump databases"
else
    for container in "${!DATABASES[@]}"; do
        # ... existing code ...
    done
fi

# List of volumes to backup
VOLUMES=(
    "app_db_data"
    "app_ghost_data"
    "app_minio_data"
    "ghost_app_db_data"
    "ghost_db_data"
)

# Backup each volume
for volume in "${VOLUMES[@]}"; do
    # Check if volume exists
    if [ ! -d "/var/lib/docker/volumes/${volume}" ]; then
        echo "Error: Volume not found: ${volume}"
        exit 1
    fi

    echo "Backing up ${volume}..."
    if [ "$DRY_RUN" = true ]; then
        echo "[DRY RUN] Would rsync /var/lib/docker/volumes/${volume} to ${BACKUP_DIR}/volumes/"
        continue
    fi

    # Create volumes directory if it doesn't exist
    mkdir -p "${BACKUP_DIR}/volumes"

    # Use rsync with hard links to previous backup if it exists
    if [ -L "${LATEST_LINK}" ] && [ -d "${LATEST_LINK}/volumes/${volume}" ]; then
        rsync -a --link-dest="${LATEST_LINK}/volumes/${volume}" "/var/lib/docker/volumes/${volume}" "${BACKUP_DIR}/volumes/"
    else
        # If no previous backup exists, do a full copy
        cp -r "/var/lib/docker/volumes/${volume}" "${BACKUP_DIR}/volumes/"
    fi
done

if [ "$DRY_RUN" = true ]; then
    echo "Dry run completed. No files were copied."
else
    echo "Backup completed successfully in ${BACKUP_DIR}"
    echo "Backup size: $(du -sh ${BACKUP_DIR} | cut -f1)"
    
    # Update the "latest" symlink to point to this backup
    rm -f "${LATEST_LINK}"
    ln -s "${BACKUP_DIR}" "${LATEST_LINK}"
fi