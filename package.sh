#!/bin/bash

# Package script for server-backup-manager
# This script creates a deployment package with all necessary files

set -e

# Configuration
VERSION=$(date +%Y%m%d)
PACKAGE_NAME="server-backup-manager-$VERSION"
BUILD_DIR="build"
PACKAGE_DIR="$BUILD_DIR/$PACKAGE_NAME"
BINARY_NAME="server-backup-manager"
ARCH="amd64"
OS="linux"

echo "Packaging server-backup-manager version $VERSION"

# Create build directory
mkdir -p "$PACKAGE_DIR"

# Build for Linux
echo "Building for Linux..."
GOOS=$OS GOARCH=$ARCH go build -o "$PACKAGE_DIR/$BINARY_NAME" .

# Copy deployment files
echo "Copying deployment files..."
cp setup-r2.sh "$PACKAGE_DIR/"
chmod +x "$PACKAGE_DIR/setup-r2.sh"

# Create sample environment file
echo "Creating sample environment file..."
cat > "$PACKAGE_DIR/sample-env" << EOF
# Server Backup Manager Environment Configuration
# Copy this file to /etc/default/server-backup-manager

# Backup Configuration
BACKUP_DIR=/home/ghost/backups
BUCKET_NAME=your-bucket-name
BUCKET_ENDPOINT=your-endpoint.r2.cloudflarestorage.com
ACCESS_KEY_ID=your-access-key
SECRET_ACCESS_KEY=your-secret-key
USE_SSL=true
RETENTION_DAYS=14
UPLOAD_SCHEDULE="0 2 * * *"

# Monitoring Configuration (optional)
ENABLE_MONITORING=false
# DISK_THRESHOLD=85
# WEBHOOK_URLS="https://example.com/webhook"
EOF

# Copy documentation
echo "Copying documentation..."
cp README.md "$PACKAGE_DIR/"
cp DEPLOY.md "$PACKAGE_DIR/"
cp TROUBLESHOOTING.md "$PACKAGE_DIR/" 2>/dev/null || echo "No troubleshooting guide found, skipping..."

# Create tarball
echo "Creating tarball..."
cd "$BUILD_DIR"
tar -czf "$PACKAGE_NAME.tar.gz" "$PACKAGE_NAME"
cd ..

# Clean up
echo "Cleaning up..."
rm -rf "$PACKAGE_DIR"

echo "Package $BUILD_DIR/$PACKAGE_NAME.tar.gz created successfully!"
echo "Transfer this package to your server and follow the instructions in DEPLOY.md" 