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
	config    *config.Config
	s3Client  *storage.S3Client
	scheduler *cron.Cron
}

// NewManager creates a new backup manager
func NewManager(cfg *config.Config, s3Client *storage.S3Client) *Manager {
	return &Manager{
		config:    cfg,
		s3Client:  s3Client,
		scheduler: cron.New(),
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

	// Upload backups
	if err := m.s3Client.UploadDirectory(ctx, m.config.BackupDir); err != nil {
		return fmt.Errorf("backup upload failed: %w", err)
	}
	log.Println("Backup upload completed successfully")

	// Clean up old backups
	if err := m.s3Client.CleanupOldBackups(ctx); err != nil {
		return fmt.Errorf("old backup cleanup failed: %w", err)
	}
	log.Println("Old backup cleanup completed successfully")

	duration := time.Since(startTime)
	log.Printf("Backup operation completed in %s", duration)
	return nil
}

// InitializeBackups scans existing backups and applies retention policies
// It will:
// 1. Delete local backups older than RetentionDays
// 2. Upload all remaining backups to S3
func (m *Manager) InitializeBackups() error {
	ctx := context.Background()
	startTime := time.Now()
	log.Println("Starting initialization of existing backups...")

	// Calculate cutoff time for retention
	cutoffTime := time.Now().AddDate(0, 0, -m.config.RetentionDays)
	log.Printf("Retention policy: keeping backups newer than %s", cutoffTime.Format("2006-01-02"))

	// Track statistics
	var (
		totalFiles    int
		deletedFiles  int
		uploadedFiles int
		errorCount    int
	)

	// Ensure the bucket exists
	if err := m.s3Client.EnsureBucketExists(ctx); err != nil {
		return fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	// Walk through the backup directory
	err := filepath.Walk(m.config.BackupDir, func(path string, info os.FileInfo, err error) error {
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

		// Check if file is older than retention period
		if info.ModTime().Before(cutoffTime) {
			log.Printf("Deleting old backup: %s (modified: %s)", path, info.ModTime().Format("2006-01-02 15:04:05"))
			if err := os.Remove(path); err != nil {
				log.Printf("Error deleting file %s: %v", path, err)
				errorCount++
			} else {
				deletedFiles++
			}
			return nil
		}

		// Get relative path for object name
		relPath, err := filepath.Rel(m.config.BackupDir, path)
		if err != nil {
			log.Printf("Error getting relative path for %s: %v", path, err)
			errorCount++
			return nil
		}

		// Replace backslashes with forward slashes for S3 object paths
		objectName := strings.ReplaceAll(relPath, "\\", "/")

		// Upload file to S3
		if err := m.s3Client.UploadFile(ctx, path, objectName); err != nil {
			log.Printf("Error uploading file %s: %v", path, err)
			errorCount++
		} else {
			uploadedFiles++
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking backup directory: %w", err)
	}

	// Clean up old backups in S3
	if err := m.s3Client.CleanupOldBackups(ctx); err != nil {
		log.Printf("Warning: error cleaning up old backups in S3: %v", err)
		errorCount++
	}

	duration := time.Since(startTime)
	log.Printf("Initialization completed in %s", duration)
	log.Printf("Summary: %d total files, %d deleted, %d uploaded, %d errors",
		totalFiles, deletedFiles, uploadedFiles, errorCount)

	return nil
}
