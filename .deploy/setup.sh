#!/bin/bash

# Check if script is run as root
if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root" 
   exit 1
fi

# Configuration
INSTALL_DIR="/opt/server-backup-manager"
CONFIG_DIR="/etc/default"
CONFIG_FILE="server-backup-manager"
SERVICE_FILE="/etc/systemd/system/server-backup-manager.service"
TEMP_CONFIG_FILE="/tmp/server-backup-manager.env"

# Create directories if they don't exist
mkdir -p "$INSTALL_DIR"
mkdir -p "$CONFIG_DIR"

# Default values
DEFAULT_BACKUP_DIR="/var/backups"
DEFAULT_BUCKET_NAME="server-backups"
DEFAULT_RETENTION_DAYS="14"
DEFAULT_UPLOAD_SCHEDULE="0 2 * * *"
DEFAULT_DISK_THRESHOLD="85"
DEFAULT_ENDPOINT="accountid.r2.cloudflarestorage.com"
DEFAULT_ACCESS_KEY="secret"
DEFAULT_SECRET_KEY="secret"

# Prompt for configuration values with defaults
read -p "Enter backup directory path [${DEFAULT_BACKUP_DIR}]: " BACKUP_DIR
BACKUP_DIR=${BACKUP_DIR:-$DEFAULT_BACKUP_DIR}

read -p "Enter R2 bucket name [${DEFAULT_BUCKET_NAME}]: " BUCKET_NAME
BUCKET_NAME=${BUCKET_NAME:-$DEFAULT_BUCKET_NAME}

ENDPOINT=${DEFAULT_ENDPOINT}

# Ensure R2 endpoint is properly set and formatted
while true; do
    read -p "Enter R2 endpoint (e.g., accountid.r2.cloudflarestorage.com): " ENDPOINT
    
    if [[ -z "$ENDPOINT" ]]; then
        ENDPOINT=${DEFAULT_ENDPOINT}
    fi
    
    if [[ -z "$ENDPOINT" ]]; then
        echo "ERROR: R2 endpoint is required. Please enter a valid endpoint."
        continue
    fi
    
    # Remove any protocol prefix if entered
    ENDPOINT=$(echo "$ENDPOINT" | sed -E 's|^(https?://)||')
    
    # Validate format - should contain r2.cloudflarestorage.com
    if [[ ! "$ENDPOINT" =~ r2\.cloudflarestorage\.com ]]; then
        echo "WARNING: Endpoint doesn't match expected format (accountid.r2.cloudflarestorage.com)"
        read -p "Are you sure this is correct? (y/n): " CONFIRM
        if [[ "$CONFIRM" =~ ^[Yy]$ ]]; then
            break
        fi
        continue
    fi
    
    break
done

echo "Using R2 endpoint: $ENDPOINT"





read -p "Enter R2 access key ID: " ACCESS_KEY
ACCESS_KEY=${ACCESS_KEY:-$DEFAULT_ACCESS_KEY}
while [[ -z "$ACCESS_KEY" ]]; do
    echo "ERROR: R2 access key ID is required."
    read -p "Enter R2 access key ID: " ACCESS_KEY
done

read -s -p "Enter R2 secret access key: " SECRET_KEY
SECRET_KEY=${SECRET_KEY:-$DEFAULT_SECRET_KEY}
while [[ -z "$SECRET_KEY" ]]; do
    echo "ERROR: R2 secret access key is required."
    read -s -p "Enter R2 secret access key: " SECRET_KEY
    echo ""
done

read -p "Enter backup retention days [${DEFAULT_RETENTION_DAYS}]: " RETENTION_DAYS
RETENTION_DAYS=${RETENTION_DAYS:-$DEFAULT_RETENTION_DAYS}

read -p "Enter backup schedule in cron format [${DEFAULT_UPLOAD_SCHEDULE}]: " UPLOAD_SCHEDULE
UPLOAD_SCHEDULE=${UPLOAD_SCHEDULE:-$DEFAULT_UPLOAD_SCHEDULE}

# Monitoring configuration
read -p "Enable disk space monitoring? (yes/no) [no]: " ENABLE_MONITORING_INPUT
ENABLE_MONITORING_INPUT=${ENABLE_MONITORING_INPUT:-"no"}
if [[ "$ENABLE_MONITORING_INPUT" =~ ^[Yy][Ee][Ss]$ ]]; then
    ENABLE_MONITORING="true"
    read -p "Enter disk usage threshold percentage [${DEFAULT_DISK_THRESHOLD}]: " DISK_THRESHOLD
    DISK_THRESHOLD=${DISK_THRESHOLD:-$DEFAULT_DISK_THRESHOLD}
    
    # Webhook configuration
    WEBHOOK_URLS=""
    read -p "Enter webhook URL (required for monitoring, leave empty to finish): " WEBHOOK_URL
    
    if [[ -z "$WEBHOOK_URL" ]]; then
        echo "Warning: Monitoring requires at least one webhook URL. Disabling monitoring."
        ENABLE_MONITORING="false"
    else
        WEBHOOK_URLS="$WEBHOOK_URL"
        
        # Allow multiple webhook URLs
        while true; do
            read -p "Enter additional webhook URL (leave empty to finish): " ADDITIONAL_URL
            if [[ -z "$ADDITIONAL_URL" ]]; then
                break
            fi
            WEBHOOK_URLS="$WEBHOOK_URLS,$ADDITIONAL_URL"
        done
    fi
else
    ENABLE_MONITORING="false"
fi

# Create environment file
cat > "$TEMP_CONFIG_FILE" << EOF
# Server Backup Manager Configuration
# Generated on $(date)

# Backup Settings
BACKUP_DIR=$BACKUP_DIR
BUCKET_NAME=$BUCKET_NAME
BUCKET_ENDPOINT=$ENDPOINT
ACCESS_KEY_ID=$ACCESS_KEY
SECRET_ACCESS_KEY=$SECRET_KEY
USE_SSL=true
RETENTION_DAYS=$RETENTION_DAYS
UPLOAD_SCHEDULE="$UPLOAD_SCHEDULE"

# Monitoring Settings
ENABLE_MONITORING=$ENABLE_MONITORING
EOF

if [[ "$ENABLE_MONITORING" == "true" ]]; then
    echo "DISK_THRESHOLD=$DISK_THRESHOLD" >> "$TEMP_CONFIG_FILE"
    echo "WEBHOOK_URLS=\"$WEBHOOK_URLS\"" >> "$TEMP_CONFIG_FILE"
fi

# Copy the environment file to the destination
cp "$TEMP_CONFIG_FILE" "$CONFIG_DIR/$CONFIG_FILE"
chmod 644 "$CONFIG_DIR/$CONFIG_FILE"
rm "$TEMP_CONFIG_FILE"

# Create systemd service file if it doesn't exist
if [[ ! -f "$SERVICE_FILE" ]]; then
    cat > "$SERVICE_FILE" << EOF
[Unit]
Description=Server Backup Manager Service
After=network.target

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=/opt/server-backup-manager
ExecStart=/opt/server-backup-manager/server-backup-manager
Restart=on-failure
RestartSec=30
Environment="GOGC=20"
EnvironmentFile=/etc/default/server-backup-manager
# Memory limits
MemoryLimit=1G
MemorySwapMax=0
# Make the OOM killer less likely to kill this process
OOMScoreAdjust=-100
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

    chmod 644 "$SERVICE_FILE"
    systemctl daemon-reload
    systemctl enable server-backup-manager
fi

echo "Configuration complete. Service will start on next boot."
echo "To start the service now, run: systemctl start server-backup-manager"
echo "To check service status, run: systemctl status server-backup-manager"
echo ""
echo "IMPORTANT: Make sure to set up a lifecycle policy in your R2 bucket for the 'archive/' prefix"
echo "to automatically delete old backups according to your retention requirements." 