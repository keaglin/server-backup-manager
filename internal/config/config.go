package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// Config holds application configuration
type Config struct {
	// Local backup directory to monitor
	BackupDir string
	// S3/Minio configuration
	BucketName      string
	BucketEndpoint  string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	// Retention settings
	RetentionDays int
	// Upload frequency (cron expression)
	UploadSchedule string
	// Run once and exit (for manual trigger)
	RunOnce bool
}

// LoadConfig loads configuration from environment variables and command-line flags
func LoadConfig() *Config {
	config := &Config{}

	// Define command-line flags
	flag.StringVar(&config.BackupDir, "backup-dir", getEnv("BACKUP_DIR", "/path/to/backups"), "Directory containing backup files")
	flag.StringVar(&config.BucketName, "bucket-name", getEnv("BUCKET_NAME", "backups"), "S3 bucket name")
	flag.StringVar(&config.BucketEndpoint, "bucket-endpoint", getEnv("BUCKET_ENDPOINT", "s3.hetzner.cloud"), "S3 endpoint URL")
	flag.StringVar(&config.AccessKeyID, "access-key", getEnv("ACCESS_KEY_ID", ""), "S3 access key ID")
	flag.StringVar(&config.SecretAccessKey, "secret-key", getEnv("SECRET_ACCESS_KEY", ""), "S3 secret access key")
	flag.BoolVar(&config.UseSSL, "use-ssl", getEnvBool("USE_SSL", true), "Use SSL for S3 connections")
	flag.IntVar(&config.RetentionDays, "retention-days", getEnvInt("RETENTION_DAYS", 90), "Number of days to retain backups")
	flag.StringVar(&config.UploadSchedule, "schedule", getEnv("UPLOAD_SCHEDULE", "0 0 */14 * *"), "Cron schedule for backups")
	flag.BoolVar(&config.RunOnce, "run-once", getEnvBool("RUN_ONCE", false), "Run backup once and exit")

	// Parse command-line flags
	flag.Parse()

	return config
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.BackupDir == "" {
		return fmt.Errorf("backup directory is required")
	}

	if c.BucketName == "" {
		return fmt.Errorf("bucket name is required")
	}

	if c.BucketEndpoint == "" {
		return fmt.Errorf("bucket endpoint is required")
	}

	if c.AccessKeyID == "" || c.SecretAccessKey == "" {
		return fmt.Errorf("S3 credentials are required")
	}

	return nil
}

// Helper functions for environment variables
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		var result int
		_, err := fmt.Sscanf(value, "%d", &result)
		if err == nil {
			return result
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		return strings.ToLower(value) == "true"
	}
	return fallback
}
