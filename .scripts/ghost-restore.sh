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
FORCE=false

# Parse command line arguments
while [[ "$#" -gt 0 ]]; do
    case $1 in
        --timestamp=*) TIMESTAMP="${1#*=}" ;;
        -d|--dry-run) DRY_RUN=true ;;
        -f|--force) FORCE=true ;;
        -h|--help)
            echo "Usage: $0 --timestamp=YYYYMMDD_HHMMSS [options]"
            echo "Options:"
            echo "  --timestamp=YYYYMMDD_HHMMSS  Backup timestamp to restore"
            echo "  -d, --dry-run                Show what would be restored without making changes"
            echo "  -f, --force                  Skip confirmation prompt"
            echo "  -h, --help                   Show this help message"
            exit 0
            ;;
        *) echo "Unknown parameter: $1"; exit 1 ;;
    esac
    shift
done

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    log "Error: Please run as root or with sudo"
    exit 1
fi

if [ -z "$TIMESTAMP" ]; then
    log "Please provide the backup timestamp"
    echo "Usage: $0 --timestamp=20240320_123456"
    exit 1
fi

BACKUP_DIR="/home/ghost/backups/${TIMESTAMP}"

if [ ! -d "${BACKUP_DIR}" ]; then
    log "Error: Backup directory not found: ${BACKUP_DIR}"
    exit 1
fi

# Check available disk space for restore
BACKUP_SIZE=$(du -sb "${BACKUP_DIR}/volumes" 2>/dev/null | awk '{print $1}')
AVAILABLE_SPACE=$(df -B1 /var/lib/docker/volumes | awk 'NR==2 {print $4}')

if [ $AVAILABLE_SPACE -lt $BACKUP_SIZE ]; then
    log "Error: Insufficient disk space for restore"
    echo "Required: $(numfmt --to=iec-i $BACKUP_SIZE)"
    echo "Available: $(numfmt --to=iec-i $AVAILABLE_SPACE)"
    exit 1
fi

# List of volumes to restore
VOLUMES=(
    "app_db_data"
    "app_ghost_data"
    "app_minio_data"
    "ghost_app_db_data"
    "ghost_db_data"
)

# List of databases to restore
declare -A DATABASES=(
    # Format for MySQL: "mysql:database_name:user:password"
    ["ghost-db-1"]="mysql:ghost:root:${MYSQL_ROOT_PASSWORD}"
    # Format for PostgreSQL: "postgres:database_name:user"
    ["tenshu-db-1"]="postgres:tenshu:tenshu"
)

if [ "$FORCE" = false ] && [ "$DRY_RUN" = false ]; then
    log "WARNING: This will:"
    echo "1. Overwrite the following volumes:"
    for volume in "${VOLUMES[@]}"; do
        echo "   - ${volume}"
    done
    echo "2. Restore the following databases:"
    for container in "${!DATABASES[@]}"; do
        DB_INFO="${DATABASES[$container]}"
        DB_TYPE="${DB_INFO%%:*}"
        DB_NAME=$(echo "$DB_INFO" | cut -d: -f2)
        echo "   - ${DB_TYPE} database '${DB_NAME}' in container '${container}'"
    done
    echo
    read -p "Are you sure you want to continue? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log "Operation cancelled."
        exit 1
    fi
fi

# First restore databases
log "Restoring database dumps..."
if [ "$DRY_RUN" = true ]; then
    log "[DRY RUN] Would restore databases"
else
    for container in "${!DATABASES[@]}"; do
        DB_INFO="${DATABASES[$container]}"
        DB_TYPE="${DB_INFO%%:*}"
        DB_NAME=$(echo "$DB_INFO" | cut -d: -f2)
        DB_USER=$(echo "$DB_INFO" | cut -d: -f3)
        DB_PASS=$(echo "$DB_INFO" | cut -d: -f4)
        
        DUMP_FILE="${BACKUP_DIR}/databases/${container}-${DB_NAME}.sql"
        
        if [ ! -f "$DUMP_FILE" ]; then
            log "Error: Database dump not found: ${DUMP_FILE}"
            exit 1
        fi
        
        log "Restoring ${DB_TYPE} database ${DB_NAME} to ${container}..."
        
        case "$DB_TYPE" in
            "mysql")
                docker exec -i "$container" mysql \
                    -u "${DB_USER}" \
                    -p"${DB_PASS}" \
                    "$DB_NAME" < "$DUMP_FILE"
                
                # Check if restore was successful
                if [ $? -ne 0 ]; then
                    log "Error: Failed to restore MySQL database ${DB_NAME}"
                    exit 1
                fi
                ;;
            
            "postgres")
                # Drop and recreate database
                docker exec "$container" dropdb -U "${DB_USER}" "${DB_NAME}" || true
                docker exec "$container" createdb -U "${DB_USER}" "${DB_NAME}"
                # Restore dump
                docker exec -i "$container" psql \
                    -U "${DB_USER}" \
                    -d "${DB_NAME}" < "$DUMP_FILE"
                
                # Check if restore was successful
                if [ $? -ne 0 ]; then
                    log "Error: Failed to restore PostgreSQL database ${DB_NAME}"
                    exit 1
                fi
                ;;
            
            *)
                log "Unknown database type: ${DB_TYPE}"
                exit 1
                ;;
        esac
    done
fi

# Then restore volumes
log "Restoring volumes..."
for volume in "${VOLUMES[@]}"; do
    if [ ! -d "${BACKUP_DIR}/volumes/${volume}" ]; then
        log "Error: Volume backup not found: ${volume}"
        exit 1
    fi

    log "Restoring ${volume}..."
    if [ "$DRY_RUN" = true ]; then
        log "[DRY RUN] Would restore ${BACKUP_DIR}/volumes/${volume} to /var/lib/docker/volumes/"
        continue
    fi

    rm -rf "/var/lib/docker/volumes/${volume}"
    cp -r "${BACKUP_DIR}/volumes/${volume}" "/var/lib/docker/volumes/"
    
    # Check if restore was successful
    if [ $? -ne 0 ]; then
        log "Error: Failed to restore volume ${volume}"
        exit 1
    fi
done

if [ "$DRY_RUN" = true ]; then
    log "Dry run completed. No changes were made."
else
    log "Restore completed successfully."
    log "NOTE: You should restart your containers for changes to take effect:"
    echo "  docker compose down && docker compose up -d"
fi 