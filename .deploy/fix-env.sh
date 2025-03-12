#!/bin/bash

# Check if script is run as root
if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root" 
   exit 1
fi

CONFIG_DIR="/etc/default"
CONFIG_FILE="server-backup-manager"
ENV_FILE="$CONFIG_DIR/$CONFIG_FILE"

echo "Checking environment configuration..."

if [[ ! -f "$ENV_FILE" ]]; then
    echo "ERROR: Configuration file $ENV_FILE does not exist!"
    echo "Please run the setup script first."
    exit 1
fi

echo "Current configuration in $ENV_FILE:"
echo "-----------------------------------"
cat "$ENV_FILE"
echo "-----------------------------------"

# Check for required variables
ENDPOINT=$(grep -E "^BUCKET_ENDPOINT=" "$ENV_FILE" | cut -d= -f2)
ENDPOINT=$(echo "$ENDPOINT" | tr -d '"' | tr -d "'")

if [[ -z "$ENDPOINT" ]]; then
    echo "ERROR: BUCKET_ENDPOINT is not set in $ENV_FILE"
    
    # Prompt for endpoint
    read -p "Enter R2 endpoint (e.g., accountid.r2.cloudflarestorage.com): " NEW_ENDPOINT
    
    if [[ -z "$NEW_ENDPOINT" ]]; then
        echo "No endpoint provided. Exiting."
        exit 1
    fi
    
    # Remove any protocol prefix if entered
    NEW_ENDPOINT=$(echo "$NEW_ENDPOINT" | sed -E 's|^(https?://)||')
    
    # Update the configuration file
    if grep -q "^BUCKET_ENDPOINT=" "$ENV_FILE"; then
        # Replace existing line
        sed -i "s|^BUCKET_ENDPOINT=.*|BUCKET_ENDPOINT=$NEW_ENDPOINT|" "$ENV_FILE"
    else
        # Add new line
        echo "BUCKET_ENDPOINT=$NEW_ENDPOINT" >> "$ENV_FILE"
    fi
    
    echo "Updated $ENV_FILE with BUCKET_ENDPOINT=$NEW_ENDPOINT"
else
    echo "BUCKET_ENDPOINT is set to: $ENDPOINT"
    
    # Check if endpoint has protocol prefix
    if [[ "$ENDPOINT" =~ ^(https?://) ]]; then
        echo "WARNING: Endpoint has protocol prefix, which may cause issues."
        echo "Current value: $ENDPOINT"
        
        # Remove protocol prefix
        NEW_ENDPOINT=$(echo "$ENDPOINT" | sed -E 's|^(https?://)||')
        
        read -p "Do you want to update it to $NEW_ENDPOINT? (y/n): " UPDATE
        if [[ "$UPDATE" =~ ^[Yy]$ ]]; then
            sed -i "s|^BUCKET_ENDPOINT=.*|BUCKET_ENDPOINT=$NEW_ENDPOINT|" "$ENV_FILE"
            echo "Updated $ENV_FILE with BUCKET_ENDPOINT=$NEW_ENDPOINT"
        fi
    fi
fi

# Check for other required variables
ACCESS_KEY=$(grep -E "^ACCESS_KEY_ID=" "$ENV_FILE" | cut -d= -f2)
ACCESS_KEY=$(echo "$ACCESS_KEY" | tr -d '"' | tr -d "'")

if [[ -z "$ACCESS_KEY" ]]; then
    echo "ERROR: ACCESS_KEY_ID is not set in $ENV_FILE"
    
    read -p "Enter R2 access key ID: " NEW_ACCESS_KEY
    
    if [[ -z "$NEW_ACCESS_KEY" ]]; then
        echo "No access key provided. Exiting."
        exit 1
    fi
    
    if grep -q "^ACCESS_KEY_ID=" "$ENV_FILE"; then
        sed -i "s|^ACCESS_KEY_ID=.*|ACCESS_KEY_ID=$NEW_ACCESS_KEY|" "$ENV_FILE"
    else
        echo "ACCESS_KEY_ID=$NEW_ACCESS_KEY" >> "$ENV_FILE"
    fi
    
    echo "Updated $ENV_FILE with ACCESS_KEY_ID=[MASKED]"
else
    echo "ACCESS_KEY_ID is set (value masked for security)"
fi

SECRET_KEY=$(grep -E "^SECRET_ACCESS_KEY=" "$ENV_FILE" | cut -d= -f2)
SECRET_KEY=$(echo "$SECRET_KEY" | tr -d '"' | tr -d "'")

if [[ -z "$SECRET_KEY" ]]; then
    echo "ERROR: SECRET_ACCESS_KEY is not set in $ENV_FILE"
    
    read -s -p "Enter R2 secret access key: " NEW_SECRET_KEY
    echo ""
    
    if [[ -z "$NEW_SECRET_KEY" ]]; then
        echo "No secret key provided. Exiting."
        exit 1
    fi
    
    if grep -q "^SECRET_ACCESS_KEY=" "$ENV_FILE"; then
        sed -i "s|^SECRET_ACCESS_KEY=.*|SECRET_ACCESS_KEY=$NEW_SECRET_KEY|" "$ENV_FILE"
    else
        echo "SECRET_ACCESS_KEY=$NEW_SECRET_KEY" >> "$ENV_FILE"
    fi
    
    echo "Updated $ENV_FILE with SECRET_ACCESS_KEY=[MASKED]"
else
    echo "SECRET_ACCESS_KEY is set (value masked for security)"
fi

BUCKET_NAME=$(grep -E "^BUCKET_NAME=" "$ENV_FILE" | cut -d= -f2)
BUCKET_NAME=$(echo "$BUCKET_NAME" | tr -d '"' | tr -d "'")

if [[ -z "$BUCKET_NAME" ]]; then
    echo "ERROR: BUCKET_NAME is not set in $ENV_FILE"
    
    read -p "Enter R2 bucket name: " NEW_BUCKET_NAME
    
    if [[ -z "$NEW_BUCKET_NAME" ]]; then
        echo "No bucket name provided. Exiting."
        exit 1
    fi
    
    if grep -q "^BUCKET_NAME=" "$ENV_FILE"; then
        sed -i "s|^BUCKET_NAME=.*|BUCKET_NAME=$NEW_BUCKET_NAME|" "$ENV_FILE"
    else
        echo "BUCKET_NAME=$NEW_BUCKET_NAME" >> "$ENV_FILE"
    fi
    
    echo "Updated $ENV_FILE with BUCKET_NAME=$NEW_BUCKET_NAME"
else
    echo "BUCKET_NAME is set to: $BUCKET_NAME"
fi

echo ""
echo "Configuration check complete."
echo "To restart the service, run: systemctl restart server-backup-manager"
echo "To check service status, run: systemctl status server-backup-manager" 