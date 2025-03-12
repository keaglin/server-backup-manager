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

# Prompt for configuration values
read -p "Enter backup directory path: " BACKUP_DIR
read -p "Enter R2 bucket name: " BUCKET_NAME
read -p "Enter R2 endpoint (e.g., xxx.r2.cloudflarestorage.com): " ENDPOINT
read -p "Enter R2 access key ID: " ACCESS_KEY
read -s -p "Enter R2 secret access key: " SECRET_KEY
echo ""
read -p "Enter backup retention days (default: 14): " RETENTION_DAYS
RETENTION_DAYS=${RETENTION_DAYS:-14}
read -p "Enter backup schedule in cron format (default: 0 2 * * *): " UPLOAD_SCHEDULE
UPLOAD_SCHEDULE=${UPLOAD_SCHEDULE:-"0 2 * * *"}

# Monitoring configuration
read -p "Enable disk space monitoring? (yes/no, default: no): " ENABLE_MONITORING_INPUT
ENABLE_MONITORING_INPUT=${ENABLE_MONITORING_INPUT:-"no"}
if [[ "$ENABLE_MONITORING_INPUT" =~ ^[Yy][Ee][Ss]$ ]]; then
    ENABLE_MONITORING="true"
    read -p "Enter disk usage threshold percentage (default: 85): " DISK_THRESHOLD
    DISK_THRESHOLD=${DISK_THRESHOLD:-85}
    
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
BACKUP_DIR=$BACKUP_DIR
BUCKET_NAME=$BUCKET_NAME
BUCKET_ENDPOINT=$ENDPOINT
ACCESS_KEY_ID=$ACCESS_KEY
SECRET_ACCESS_KEY=$SECRET_KEY
USE_SSL=true
RETENTION_DAYS=$RETENTION_DAYS
UPLOAD_SCHEDULE="$UPLOAD_SCHEDULE"
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
Description=Server Backup Manager
After=network.target

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/server-backup-manager
EnvironmentFile=$CONFIG_DIR/$CONFIG_FILE
Restart=on-failure
RestartSec=10
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