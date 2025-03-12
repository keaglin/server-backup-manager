#!/bin/bash

# Server Backup Manager Deployment Script
# This script installs the server-backup-manager as a systemd service

set -e

# Configuration
INSTALL_DIR="/opt/server-backup-manager"
SERVICE_NAME="server-backup-manager"
EXECUTABLE_NAME="server-backup-manager"
CONFIG_DIR="$INSTALL_DIR/config"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root or with sudo"
  exit 1
fi

# Create installation directory
echo "Creating installation directory..."
mkdir -p "$INSTALL_DIR"
mkdir -p "$CONFIG_DIR"

# Copy executable
echo "Copying executable..."
cp "$EXECUTABLE_NAME" "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/$EXECUTABLE_NAME"

# Copy configuration files if they exist
if [ -d "config" ]; then
  echo "Copying configuration files..."
  cp -r config/* "$CONFIG_DIR/"
fi

# Copy service file
echo "Installing systemd service..."
cp "$SERVICE_NAME.service" "/etc/systemd/system/"

# Reload systemd
echo "Reloading systemd..."
systemctl daemon-reload

# Enable and start service
echo "Enabling and starting service..."
systemctl enable "$SERVICE_NAME"
systemctl start "$SERVICE_NAME"

echo "Checking service status..."
systemctl status "$SERVICE_NAME"

echo "Installation complete!"
echo "You can check the service logs with: journalctl -u $SERVICE_NAME -f" 