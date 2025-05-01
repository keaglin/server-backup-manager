package backup

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/user/server-backup-manager/internal/config"
	"github.com/user/server-backup-manager/internal/storage"
)

// Manager handles backup operations
type Manager struct {
	config        *config.Config
	s3Client      *storage.S3Client
	scheduler     *cron.Cron
	uploadTracker *storage.UploadTracker
}

// NewManager creates a new backup manager
func NewManager(cfg *config.Config, s3Client *storage.S3Client) *Manager {
	// Initialize upload tracker
	tracker, err := storage.NewUploadTracker(cfg.BackupDir)
	if err != nil {
		log.Printf("Warning: Failed to initialize upload tracker: %v", err)
		log.Printf("Resumable uploads will not be available")
	}

	return &Manager{
		config:        cfg,
		s3Client:      s3Client,
		scheduler:     cron.New(),
		uploadTracker: tracker,
	}
}

// Start starts the backup manager
func (m *Manager) Start() error {
	// If initialize mode is enabled, run initialization and exit
	if m.config.InitializeMode {
		log.Println("Running in initialization mode")
		return m.InitializeBackups()
	}

	// If run-once mode is enabled, run backup immediately and exit
	if m.config.RunOnce {
		log.Println("Running in one-time backup mode")
		return m.RunBackup()
	}

	// Schedule backup uploads
	_, err := m.scheduler.AddFunc(m.config.UploadSchedule, func() {
		if err := m.RunBackup(); err != nil {
			log.Printf("Scheduled backup failed: %v", err)
		}
	})

	if err != nil {
		return fmt.Errorf("failed to schedule backup job: %w", err)
	}

	// Start cron scheduler
	m.scheduler.Start()
	log.Printf("Backup manager started. Will upload backups every %s and retain for %d days",
		m.config.UploadSchedule, m.config.RetentionDays)

	return nil
}

// Stop stops the backup manager
func (m *Manager) Stop() {
	if m.scheduler != nil {
		m.scheduler.Stop()
	}
}

// RunBackup performs a backup operation
func (m *Manager) RunBackup() error {
	ctx := context.Background()
	startTime := time.Now()
	log.Println("Starting backup operation...")

	// Get the latest backup directory name (format: YYYYMMDD_HHMMSS)
	latestBackup, err := m.findLatestBackup()
	if err != nil {
		return fmt.Errorf("failed to find latest backup: %w", err)
	}

	if latestBackup == "" {
		log.Println("No backups found to upload")
		return nil
	}

	log.Printf("Found latest backup: %s", latestBackup)
	backupPath := filepath.Join(m.config.BackupDir, latestBackup)

	// Upload the latest backup
	if err := m.uploadBackup(ctx, backupPath, latestBackup); err != nil {
		return fmt.Errorf("backup upload failed: %w", err)
	}
	log.Println("Backup upload completed successfully")

	// Call CleanupOldBackups for backward compatibility
	// This function no longer deletes objects but logs a message about R2 lifecycle policies
	if err := m.s3Client.CleanupOldBackups(ctx); err != nil {
		return fmt.Errorf("old backup cleanup failed: %w", err)
	}

	duration := time.Since(startTime)
	log.Printf("Backup operation completed in %s", duration)
	return nil
}

// findLatestBackup finds the most recent backup directory
func (m *Manager) findLatestBackup() (string, error) {
	// List all backup directories (format: YYYYMMDD_HHMMSS)
	entries, err := os.ReadDir(m.config.BackupDir)
	if err != nil {
		return "", fmt.Errorf("failed to read backup directory: %w", err)
	}

	var latestBackup string
	var latestTime time.Time

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if the directory name matches the backup format (YYYYMMDD_HHMMSS)
		name := entry.Name()
		if len(name) != 15 || !strings.Contains(name, "_") {
			continue
		}

		// Try to parse the directory name as a timestamp
		t, err := time.Parse("20060102_150405", name)
		if err != nil {
			continue
		}

		// If this is the first valid backup or newer than the current latest
		if latestBackup == "" || t.After(latestTime) {
			latestBackup = name
			latestTime = t
		}
	}

	return latestBackup, nil
}

// uploadBackup uploads a specific backup directory
func (m *Manager) uploadBackup(ctx context.Context, backupPath, backupName string) error {
	log.Printf("Uploading backup %s to R2 bucket", backupName)

	// Calculate cutoff time for old logs (14 days)
	oldLogsCutoff := time.Now().AddDate(0, 0, -14)
	log.Printf("Files older than %s will be uploaded as 'archive/' objects", oldLogsCutoff.Format("2006-01-02"))

	// Track statistics
	var totalFiles, uploadedFiles, skippedFiles, errorCount int

	// Ensure the bucket exists
	if err := m.s3Client.EnsureBucketExists(ctx); err != nil {
		return fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	// Walk through the backup directory
	err := filepath.Walk(backupPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error accessing path %s: %v", path, err)
			errorCount++
			return nil // Continue despite errors
		}

		// Skip directories and non-regular files
		if info.IsDir() || !info.Mode().IsRegular() {
			return nil
		}

		totalFiles++

		// Skip if already uploaded (if tracker is available)
		if m.uploadTracker != nil && m.uploadTracker.IsUploaded(path) {
			log.Printf("File %s already uploaded, skipping", path)
			skippedFiles++
			return nil
		}

		// Get relative path for object name
		relPath, err := filepath.Rel(backupPath, path)
		if err != nil {
			log.Printf("Error getting relative path for %s: %v", path, err)
			errorCount++
			return nil
		}

		// Replace backslashes with forward slashes for S3 object paths
		objectName := backupName + "/" + strings.ReplaceAll(relPath, "\\", "/")

		// If the file is older than 14 days, put it in an "archive" folder
		if info.ModTime().Before(oldLogsCutoff) {
			objectName = "archive/" + objectName
			log.Printf("File %s is older than 14 days, uploading to archive folder", path)
		}

		// Upload file to S3
		if err := m.s3Client.UploadFile(ctx, path, objectName); err != nil {
			log.Printf("Error uploading file %s: %v", path, err)
			errorCount++
			return nil
		}

		uploadedFiles++

		// Mark as uploaded if tracker is available
		if m.uploadTracker != nil {
			if err := m.uploadTracker.MarkUploaded(path, backupName); err != nil {
				log.Printf("Warning: Failed to mark file as uploaded: %v", err)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking backup directory: %w", err)
	}

	log.Printf("Upload summary: %d total files, %d uploaded, %d skipped, %d errors",
		totalFiles, uploadedFiles, skippedFiles, errorCount)

	return nil
}

// InitializeBackups scans existing backups and applies retention policies
// It will:
// 1. Upload all backups to S3, with files older than 14 days going to the archive/ prefix
func (m *Manager) InitializeBackups() error {
	ctx := context.Background()
	startTime := time.Now()
	log.Println("Starting initialization of existing backups...")

	// Calculate cutoff time for old logs (14 days)
	oldLogsCutoff := time.Now().AddDate(0, 0, -14)
	log.Printf("Files older than %s will be uploaded to the archive/ prefix", oldLogsCutoff.Format("2006-01-02"))

	// Track statistics
	var totalBackups, totalFiles, uploadedFiles, skippedFiles, errorCount int

	// Ensure the bucket exists
	if err := m.s3Client.EnsureBucketExists(ctx); err != nil {
		return fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	// List all backup directories (format: YYYYMMDD_HHMMSS)
	entries, err := os.ReadDir(m.config.BackupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	// Process each backup directory
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if the directory name matches the backup format (YYYYMMDD_HHMMSS)
		backupName := entry.Name()
		if len(backupName) != 15 || !strings.Contains(backupName, "_") {
			continue
		}

		// Try to parse the directory name as a timestamp
		_, err := time.Parse("20060102_150405", backupName)
		if err != nil {
			continue
		}

		totalBackups++
		backupPath := filepath.Join(m.config.BackupDir, backupName)
		log.Printf("Processing backup %d: %s", totalBackups, backupName)

		// Upload this backup
		err = filepath.Walk(backupPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Printf("Error accessing path %s: %v", path, err)
				errorCount++
				return nil // Continue despite errors
			}

			// Skip directories and non-regular files
			if info.IsDir() || !info.Mode().IsRegular() {
				return nil
			}

			totalFiles++

			// Skip if already uploaded (if tracker is available)
			if m.uploadTracker != nil && m.uploadTracker.IsUploaded(path) {
				log.Printf("File %s already uploaded, skipping", path)
				skippedFiles++
				return nil
			}

			// Get relative path for object name
			relPath, err := filepath.Rel(backupPath, path)
			if err != nil {
				log.Printf("Error getting relative path for %s: %v", path, err)
				errorCount++
				return nil
			}

			// Replace backslashes with forward slashes for S3 object paths
			objectName := backupName + "/" + strings.ReplaceAll(relPath, "\\", "/")

			// If the file is older than 14 days, put it in an "archive" folder
			if info.ModTime().Before(oldLogsCutoff) {
				objectName = "archive/" + objectName
				log.Printf("File %s is older than 14 days, uploading to archive folder", path)
			}

			// Upload file to S3
			if err := m.s3Client.UploadFile(ctx, path, objectName); err != nil {
				log.Printf("Error uploading file %s: %v", path, err)
				errorCount++
				return nil
			}

			uploadedFiles++

			// Mark as uploaded if tracker is available
			if m.uploadTracker != nil {
				if err := m.uploadTracker.MarkUploaded(path, backupName); err != nil {
					log.Printf("Warning: Failed to mark file as uploaded: %v", err)
				}
			}

			return nil
		})

		if err != nil {
			log.Printf("Error processing backup %s: %v", backupName, err)
			errorCount++
		}

		// Log progress after each backup
		log.Printf("Completed backup %s: %d files processed", backupName, totalFiles)
	}

	duration := time.Since(startTime)
	log.Printf("Initialization completed in %s", duration)
	log.Printf("Summary: %d backups, %d total files, %d uploaded, %d skipped, %d errors",
		totalBackups, totalFiles, uploadedFiles, skippedFiles, errorCount)

	return nil
}
