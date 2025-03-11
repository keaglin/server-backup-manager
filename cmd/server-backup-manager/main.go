package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/user/server-backup-manager/internal/backup"
	"github.com/user/server-backup-manager/internal/config"
	"github.com/user/server-backup-manager/internal/storage"
	"github.com/user/server-backup-manager/pkg/utils"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Setup logging
	logDir := filepath.Join(cfg.BackupDir, "logs")
	logger, err := utils.NewLogger(logDir)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Close()

	// Initialize S3 client
	s3Client, err := storage.NewS3Client(cfg)
	if err != nil {
		logger.Error("Failed to initialize S3 client: %v", err)
		os.Exit(1)
	}

	// Initialize backup manager
	backupManager := backup.NewManager(cfg, s3Client)

	// Handle one-time backup mode
	if cfg.RunOnce {
		if err := backupManager.RunBackup(); err != nil {
			logger.Error("Backup failed: %v", err)
			os.Exit(1)
		}
		logger.Info("One-time backup completed successfully")
		return
	}

	// Start backup manager
	if err := backupManager.Start(); err != nil {
		logger.Error("Failed to start backup manager: %v", err)
		os.Exit(1)
	}

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	sig := <-sigCh
	logger.Info("Received signal %v, shutting down...", sig)

	// Stop backup manager
	backupManager.Stop()
	logger.Info("Backup manager stopped")
}
