package backup

import (
	"context"
	"fmt"
	"log"
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
