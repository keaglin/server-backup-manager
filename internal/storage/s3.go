package storage

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/user/server-backup-manager/internal/config"
)

// S3Client handles operations with S3-compatible storage
type S3Client struct {
	client *minio.Client
	config *config.Config
}

// NewS3Client initializes a new S3 client
func NewS3Client(cfg *config.Config) (*S3Client, error) {
	client, err := minio.New(cfg.BucketEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize S3 client: %w", err)
	}

	return &S3Client{
		client: client,
		config: cfg,
	}, nil
}

// EnsureBucketExists creates the bucket if it doesn't exist
func (s *S3Client) EnsureBucketExists(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.config.BucketName)
	if err != nil {
		return fmt.Errorf("failed to check if bucket exists: %w", err)
	}

	if !exists {
		err = s.client.MakeBucket(ctx, s.config.BucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		log.Printf("Bucket '%s' created successfully", s.config.BucketName)
	}

	return nil
}

// UploadFile uploads a single file to the bucket
func (s *S3Client) UploadFile(ctx context.Context, localPath, objectName string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", localPath, err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file stats: %w", err)
	}

	_, err = s.client.PutObject(ctx, s.config.BucketName, objectName, file, stat.Size(),
		minio.PutObjectOptions{
			ContentType: "application/octet-stream",
			UserMetadata: map[string]string{
				"upload-date": time.Now().Format(time.RFC3339),
			},
		})

	if err != nil {
		return fmt.Errorf("failed to upload file %s: %w", localPath, err)
	}

	log.Printf("Uploaded %s to bucket as %s", localPath, objectName)
	return nil
}

// UploadDirectory uploads all files in a directory to the bucket
func (s *S3Client) UploadDirectory(ctx context.Context, dirPath string) error {
	if err := s.EnsureBucketExists(ctx); err != nil {
		return err
	}

	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path for object name
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Replace backslashes with forward slashes for S3 object paths
		objectName := strings.ReplaceAll(relPath, "\\", "/")

		return s.UploadFile(ctx, path, objectName)
	})
}

// CleanupOldBackups removes backups older than the retention period
func (s *S3Client) CleanupOldBackups(ctx context.Context) error {
	cutoffTime := time.Now().AddDate(0, 0, -s.config.RetentionDays)

	// List all objects in the bucket
	objectCh := s.client.ListObjects(ctx, s.config.BucketName, minio.ListObjectsOptions{
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			log.Printf("Error listing object: %v", object.Err)
			continue
		}

		// Check if object is older than retention period
		if object.LastModified.Before(cutoffTime) {
			err := s.client.RemoveObject(ctx, s.config.BucketName, object.Key, minio.RemoveObjectOptions{})
			if err != nil {
				log.Printf("Failed to remove old backup %s: %v", object.Key, err)
				continue
			}
			log.Printf("Removed old backup: %s (last modified: %s)",
				object.Key, object.LastModified.Format(time.RFC3339))
		}
	}

	return nil
}
