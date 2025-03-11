package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
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
	// Initialize mode - scan existing backups and apply retention policies
	InitializeMode bool
	// Monitoring configuration
	EnableMonitoring     bool
	MonitorPaths         []string
	DiskThresholdPercent float64
	MonitorInterval      time.Duration
	WebhookURLs          []string
	WebhookTimeout       time.Duration
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
	flag.BoolVar(&config.InitializeMode, "initialize", getEnvBool("INITIALIZE", false), "Initialize mode: scan existing backups and apply retention policies")

	// Monitoring flags
	flag.BoolVar(&config.EnableMonitoring, "enable-monitoring", getEnvBool("ENABLE_MONITORING", true), "Enable disk space monitoring")
	monitorPathsStr := flag.String("monitor-paths", getEnv("MONITOR_PATHS", ""), "Comma-separated list of paths to monitor for disk space")
	flag.Float64Var(&config.DiskThresholdPercent, "disk-threshold", getEnvFloat("DISK_THRESHOLD", 75.0), "Disk usage threshold percentage for alerts")
	monitorIntervalStr := flag.String("monitor-interval", getEnv("MONITOR_INTERVAL", "30m"), "Interval for disk space checks")
	webhookURLsStr := flag.String("webhook-urls", getEnv("WEBHOOK_URLS", ""), "Comma-separated list of webhook URLs for alerts")
	webhookTimeoutStr := flag.String("webhook-timeout", getEnv("WEBHOOK_TIMEOUT", "10s"), "Timeout for webhook requests")

	// Parse command-line flags
	flag.Parse()

	// Process monitoring paths
	if *monitorPathsStr != "" {
		config.MonitorPaths = strings.Split(*monitorPathsStr, ",")
		// Trim spaces
		for i, path := range config.MonitorPaths {
			config.MonitorPaths[i] = strings.TrimSpace(path)
		}
	} else {
		// Default to monitoring the backup directory
		config.MonitorPaths = []string{config.BackupDir}
	}

	// Process webhook URLs
	if *webhookURLsStr != "" {
		config.WebhookURLs = strings.Split(*webhookURLsStr, ",")
		// Trim spaces
		for i, url := range config.WebhookURLs {
			config.WebhookURLs[i] = strings.TrimSpace(url)
		}
	}

	// Parse durations
	var err error
	config.MonitorInterval, err = time.ParseDuration(*monitorIntervalStr)
	if err != nil {
		config.MonitorInterval = 30 * time.Minute // Default to 30 minutes
	}

	config.WebhookTimeout, err = time.ParseDuration(*webhookTimeoutStr)
	if err != nil {
		config.WebhookTimeout = 10 * time.Second // Default to 10 seconds
	}

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

	// Validate monitoring configuration if enabled
	if c.EnableMonitoring {
		if len(c.MonitorPaths) == 0 {
			return fmt.Errorf("at least one monitoring path is required when monitoring is enabled")
		}

		if c.DiskThresholdPercent <= 0 || c.DiskThresholdPercent > 100 {
			return fmt.Errorf("disk threshold percentage must be between 0 and 100")
		}

		if c.MonitorInterval < time.Minute {
			return fmt.Errorf("monitor interval must be at least 1 minute")
		}

		if len(c.WebhookURLs) == 0 {
			return fmt.Errorf("at least one webhook URL is required when monitoring is enabled")
		}
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

func getEnvFloat(key string, fallback float64) float64 {
	if value, exists := os.LookupEnv(key); exists {
		var result float64
		_, err := fmt.Sscanf(value, "%f", &result)
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
