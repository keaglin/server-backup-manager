#!/bin/bash

# Load environment variables
if [ -f .env ]; then
    source .env
fi

# Log function to add timestamps
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Default settings
DRY_RUN=false
MAX_BACKUPS=5  # Default number of backups to keep

# Parse command line arguments
while [[ "$#" -gt 0 ]]; do
    case $1 in
        -d|--dry-run) DRY_RUN=true ;;
        -m|--max-backups=*) MAX_BACKUPS="${1#*=}" ;;
        -h|--help)
            echo "Usage: $0 [options]"
            echo "Options:"
            echo "  -d, --dry-run                Show what would be backed up without actually copying"
            echo "  -m, --max-backups=N          Keep only N most recent backups (default: 5)"
            echo "  -h, --help                   Show this help message"
            exit 0
            ;;
        *) echo "Unknown parameter: $1"; exit 1 ;;
    esac
    shift
done

# Check if MAX_BACKUPS is a positive integer
if ! [[ "$MAX_BACKUPS" =~ ^[0-9]+$ ]] || [ "$MAX_BACKUPS" -lt 1 ]; then
    log "Error: MAX_BACKUPS must be a positive integer"
    exit 1
fi

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    log "Error: Please run as root or with sudo"
    exit 1
fi

# Set backup directory
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/home/ghost/backups/${TIMESTAMP}"
LATEST_LINK="/home/ghost/backups/latest"

# Check if backup directory already exists
if [ -d "${BACKUP_DIR}" ]; then
    log "Error: Backup directory already exists: ${BACKUP_DIR}"
    exit 1
fi

# Check available disk space
REQUIRED_SPACE=$(du -sb /var/lib/docker/volumes/{app_db_data,ghost_data,minio_data} 2>/dev/null | awk '{total += $1} END {print total}')
AVAILABLE_SPACE=$(df -B1 /home/ghost/backups | awk 'NR==2 {print $4}')

if [ $AVAILABLE_SPACE -lt $REQUIRED_SPACE ]; then
    log "Error: Insufficient disk space"
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
log "Creating database dumps..."
if [ "$DRY_RUN" = true ]; then
    log "[DRY RUN] Would dump databases"
else
    for container in "${!DATABASES[@]}"; do
        DB_INFO="${DATABASES[$container]}"
        DB_TYPE="${DB_INFO%%:*}"
        DB_NAME=$(echo "$DB_INFO" | cut -d: -f2)
        DB_USER=$(echo "$DB_INFO" | cut -d: -f3)
        DB_PASS=$(echo "$DB_INFO" | cut -d: -f4)
        
        DUMP_FILE="${BACKUP_DIR}/databases/${container}-${DB_NAME}.sql"
        
        log "Dumping ${DB_TYPE} database ${DB_NAME} from ${container}..."
        
        case "$DB_TYPE" in
            "mysql")
                docker exec "$container" mysqldump \
                    -u "${DB_USER}" \
                    -p"${DB_PASS}" \
                    --single-transaction \
                    --quick \
                    --lock-tables=false \
                    "$DB_NAME" > "$DUMP_FILE"
                ;;
            
            "postgres")
                docker exec "$container" pg_dump \
                    -U "${DB_USER}" \
                    -d "${DB_NAME}" \
                    -c > "$DUMP_FILE"
                ;;
            
            *)
                echo "Unknown database type: ${DB_TYPE}"
                exit 1
                ;;
        esac
        
        # Check if dump was successful
        if [ ! -s "$DUMP_FILE" ]; then
            log "Error: Failed to create database dump for ${DB_NAME}"
            exit 1
        fi
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
        log "Error: Volume not found: ${volume}"
        exit 1
    fi

    log "Backing up ${volume}..."
    if [ "$DRY_RUN" = true ]; then
        log "[DRY RUN] Would rsync /var/lib/docker/volumes/${volume} to ${BACKUP_DIR}/volumes/"
        continue
    fi

    # Create volumes directory if it doesn't exist
    mkdir -p "${BACKUP_DIR}/volumes"

    # Use rsync with hard links to previous backup if it exists
    if [ -L "${LATEST_LINK}" ] && [ -d "${LATEST_LINK}/volumes/${volume}" ]; then
        rsync -a --link-dest="${LATEST_LINK}/volumes/${volume}" "/var/lib/docker/volumes/${volume}" "${BACKUP_DIR}/volumes/"
        
        # Check if backup was successful
        if [ $? -ne 0 ]; then
            log "Error: Failed to backup volume ${volume} using rsync"
            exit 1
        fi
    else
        # If no previous backup exists, do a full copy
        cp -r "/var/lib/docker/volumes/${volume}" "${BACKUP_DIR}/volumes/"
        
        # Check if backup was successful
        if [ $? -ne 0 ]; then
            log "Error: Failed to backup volume ${volume}"
            exit 1
        fi
    fi
done

if [ "$DRY_RUN" = true ]; then
    log "Dry run completed. No files were copied."
else
    log "Backup completed successfully in ${BACKUP_DIR}"
    echo "Backup size: $(du -sh ${BACKUP_DIR} | cut -f1)"
    
    # Update the "latest" symlink to point to this backup
    rm -f "${LATEST_LINK}"
    ln -s "${BACKUP_DIR}" "${LATEST_LINK}"
    
    # Rotate old backups
    log "Checking for old backups to remove..."
    if [ "$MAX_BACKUPS" -gt 0 ]; then
        # List all backups sorted by date (oldest first)
        BACKUPS=$(find /home/ghost/backups -maxdepth 1 -type d -name "20*_*" | sort)
        
        # Count the number of backups
        BACKUP_COUNT=$(echo "$BACKUPS" | wc -l)
        
        # Remove oldest backups if we have more than MAX_BACKUPS
        if [ "$BACKUP_COUNT" -gt "$MAX_BACKUPS" ]; then
            REMOVE_COUNT=$((BACKUP_COUNT - MAX_BACKUPS))
            log "Removing $REMOVE_COUNT old backup(s)..."
            
            # Get the list of backups to remove
            BACKUPS_TO_REMOVE=$(echo "$BACKUPS" | head -n "$REMOVE_COUNT")
            
            # Remove each backup
            echo "$BACKUPS_TO_REMOVE" | while read -r backup; do
                log "Removing old backup: $backup"
                rm -rf "$backup"
            done
        else
            log "No old backups to remove. Current count: $BACKUP_COUNT, max: $MAX_BACKUPS"
        fi
    fi
fi