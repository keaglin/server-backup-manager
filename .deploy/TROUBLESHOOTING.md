# Server Backup Manager Troubleshooting Guide

## Configuration Issues and Resolution

### Problem Summary
The server-backup-manager application was failing to start with the error message:
```
Invalid configuration: at least one webhook URL is required when monitoring is enabled
```

This error persisted even after attempting to disable monitoring in the JSON configuration file.

### Root Cause Analysis
After extensive troubleshooting, we discovered that the application was not reading the JSON configuration file at all. Instead, it was using environment variables for configuration.

**Evidence:**
1. Modifying the JSON configuration file had no effect on the application's behavior
2. Setting environment variables directly resolved the issue immediately
3. The application continued to report that monitoring was enabled despite the JSON configuration explicitly disabling it

### Attempted Solutions

#### 1. Modifying the JSON Configuration File (Failed)
We initially tried to edit the configuration file at `/opt/server-backup-manager/config/config.json` to set `"EnableMonitoring": false`. This did not resolve the issue.

#### 2. Creating a New Configuration File (Failed)
We created a new configuration file with minimal settings and `"EnableMonitoring": false`. This also failed to resolve the issue.

#### 3. Validating JSON Syntax (Inconclusive)
We used `jq` to validate the JSON syntax of the configuration file. The JSON was valid, but this didn't address the underlying issue.

#### 4. Using Environment Variables (Successful)
Setting the environment variables directly, particularly `ENABLE_MONITORING=false`, immediately resolved the issue.

### Final Solution
We implemented a permanent solution using systemd's environment file mechanism:

1. Created an environment file at `/etc/default/server-backup-manager` containing all necessary configuration variables:
   ```
   BACKUP_DIR=/home/ghost/backups
   BUCKET_NAME=blerdimension
   BUCKET_ENDPOINT=765d4225202a79629f59e6d77d61a7ec.r2.cloudflarestorage.com
   ACCESS_KEY_ID=136bc557feed76ad6cb3e6253ae78ece
   SECRET_ACCESS_KEY=4ca4d8bad7d825967bfcd008f8de1631fcf55a21a9c713c205494a3cff464233
   USE_SSL=true
   ENABLE_MONITORING=false
   RETENTION_DAYS=14
   UPLOAD_SCHEDULE="0 2 * * *"
   ```

2. Modified the systemd service file at `/etc/systemd/system/server-backup-manager.service` to use this environment file by adding:
   ```
   EnvironmentFile=/etc/default/server-backup-manager
   ```

3. Reloaded systemd and restarted the service:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl restart server-backup-manager
   ```

### Technical Explanation
The application appears to be designed to read configuration from environment variables rather than from a JSON file. This is a common pattern in Go applications, especially those following the [12-factor app](https://12factor.net/) methodology.

The JSON configuration file that was initially created was likely not part of the original application design, or the code to read it was not properly implemented. When we tried to configure it using a JSON file, the application ignored that file and used default values or environment variables instead.

### Lessons Learned
1. **Configuration Source Matters**: Always verify how an application is designed to read its configuration (environment variables, config files, command-line arguments, etc.)
2. **Test Configuration Changes**: After making configuration changes, verify that the application is actually using the new values
3. **Environment Variables as a Fallback**: For many modern applications, especially those written in Go, environment variables are often the primary or fallback configuration method

### Recommended Updates
1. Update the setup script to create the environment file instead of a JSON configuration
2. Update the deployment documentation to clarify that the application uses environment variables for configuration
3. Consider modifying the application code to properly read from the JSON configuration file if that's the preferred configuration method

### Environment Variable Reference
Here's a list of the environment variables used by the server-backup-manager:

| Environment Variable | Description | Example Value |
|----------------------|-------------|--------------|
| BACKUP_DIR | Directory to back up | /home/ghost/backups |
| BUCKET_NAME | R2 bucket name | blerdimension |
| BUCKET_ENDPOINT | R2 endpoint URL | 765d4225202a79629f59e6d77d61a7ec.r2.cloudflarestorage.com |
| ACCESS_KEY_ID | R2 access key ID | 136bc557feed76ad6cb3e6253ae78ece |
| SECRET_ACCESS_KEY | R2 secret access key | 4ca4d8bad7d825967bfcd008f8de1631fcf55a21a9c713c205494a3cff464233 |
| USE_SSL | Whether to use SSL for R2 connection | true |
| ENABLE_MONITORING | Whether to enable disk space monitoring | false |
| RETENTION_DAYS | Number of days to retain backups | 14 |
| UPLOAD_SCHEDULE | Cron schedule for backups | 0 2 * * * |
| DISK_THRESHOLD | Disk usage threshold percentage | 85 |
| WEBHOOK_URLS | Comma-separated list of webhook URLs | https://example.com/webhook | 