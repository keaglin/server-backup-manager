package storage

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/user/server-backup-manager/internal/config"
)

// S3Client handles operations with S3-compatible storage
type S3Client struct {
	client *s3.Client
	config *config.Config
}

// NewS3Client initializes a new S3 client
func NewS3Client(cfg *config.Config) (*S3Client, error) {
	// Validate required configuration
	if cfg.BucketEndpoint == "" {
		return nil, fmt.Errorf("bucket endpoint is required - please check your configuration")
	}

	if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return nil, fmt.Errorf("R2 credentials (access key ID and secret access key) are required")
	}

	if cfg.BucketName == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	// Log configuration (without exposing secrets)
	log.Printf("Initializing S3 client with the following configuration:")
	log.Printf("- Access Key ID: %s (length: %d)", maskString(cfg.AccessKeyID), len(cfg.AccessKeyID))
	log.Printf("- Secret Access Key length: %d", len(cfg.SecretAccessKey))
	log.Printf("- Bucket Endpoint: %s", cfg.BucketEndpoint)
	log.Printf("- Bucket Name: %s", cfg.BucketName)

	// Format the endpoint correctly for Cloudflare R2
	// Expected format: https://accountid.r2.cloudflarestorage.com
	endpoint := cfg.BucketEndpoint

	// If the endpoint doesn't already have a protocol, add https://
	if !strings.HasPrefix(endpoint, "https://") && !strings.HasPrefix(endpoint, "http://") {
		endpoint = fmt.Sprintf("https://%s", endpoint)
	}

	// Log the endpoint being used for debugging
	log.Printf("Using R2 endpoint: %s", endpoint)

	// Following Cloudflare R2 example: https://developers.cloudflare.com/r2/examples/aws/aws-sdk-go/
	// Create a custom resolver that forces the endpoint
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               endpoint,
			HostnameImmutable: true,
			SigningRegion:     "auto",
		}, nil
	})

	// Create AWS SDK configuration
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		// Use static credentials provider with the access key and secret
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
		// R2 requires "auto" as the region
		awsconfig.WithRegion("auto"),
		// Use our custom endpoint resolver
		awsconfig.WithEndpointResolverWithOptions(customResolver),
		// Enable logging for debugging
		awsconfig.WithClientLogMode(aws.LogRetries|aws.LogRequest|aws.LogResponse),
		// IMPORTANT: Disable checksum calculation and validation for R2 compatibility
		// See: https://developers.cloudflare.com/r2/examples/aws/aws-sdk-go/
		awsconfig.WithRequestChecksumCalculation(0),
		awsconfig.WithResponseChecksumValidation(0),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS SDK config: %w", err)
	}

	// Create S3 client with Cloudflare R2 endpoint
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		// Force path-style addressing (bucket name in the path rather than subdomain)
		// This is important for compatibility with R2
		o.UsePathStyle = true
	})

	// Test the credentials with a simple operation
	log.Printf("Testing S3 client connection...")
	_, err = client.ListBuckets(context.Background(), &s3.ListBucketsInput{})
	if err != nil {
		log.Printf("WARNING: Failed to list buckets during initialization: %v", err)
		// Continue anyway, as the bucket might not exist yet
	} else {
		log.Printf("Successfully connected to S3 endpoint")
	}

	return &S3Client{
		client: client,
		config: cfg,
	}, nil
}

// maskString masks a string for logging, showing only the first and last characters
func maskString(s string) string {
	if len(s) <= 6 {
		return "***" // Don't show anything for short strings
	}
	return s[:3] + "..." + s[len(s)-3:]
}

// EnsureBucketExists creates the bucket if it doesn't exist
func (s *S3Client) EnsureBucketExists(ctx context.Context) error {
	log.Printf("Checking if bucket '%s' exists...", s.config.BucketName)

	// Check if bucket exists
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.config.BucketName),
	})

	if err != nil {
		log.Printf("Bucket '%s' does not exist or cannot be accessed: %v", s.config.BucketName, err)
		log.Printf("Attempting to create bucket '%s'...", s.config.BucketName)

		// If bucket doesn't exist, create it
		_, err = s.client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(s.config.BucketName),
		})
		if err != nil {
			log.Printf("Failed to create bucket: %v", err)
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		log.Printf("Bucket '%s' created successfully", s.config.BucketName)
	} else {
		log.Printf("Bucket '%s' already exists", s.config.BucketName)
	}

	return nil
}

// CheckIfFileExists checks if a file already exists in the bucket with the same size
func (s *S3Client) CheckIfFileExists(ctx context.Context, objectName string, localSize int64) (bool, error) {
	// Try to get object metadata
	resp, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(objectName),
	})

	if err != nil {
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			// Object doesn't exist
			return false, nil
		}
		// Some other error
		return false, fmt.Errorf("error checking if file exists: %w", err)
	}

	// Object exists, check if sizes match
	if resp.ContentLength != nil {
		return *resp.ContentLength == localSize, nil
	}

	// If ContentLength is nil, assume sizes don't match
	return false, nil
}

// UploadFile uploads a single file to the bucket using memory-efficient streaming
func (s *S3Client) UploadFile(ctx context.Context, localPath, objectName string) error {
	// Get file info for size
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("failed to get file info for %s: %w", localPath, err)
	}

	// Check if file already exists with same size
	exists, err := s.CheckIfFileExists(ctx, objectName, fileInfo.Size())
	if err != nil {
		log.Printf("Warning: Failed to check if file exists: %v", err)
	} else if exists {
		log.Printf("File %s already exists in bucket with same size, skipping upload", objectName)
		return nil
	}

	// Open the file for reading
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", localPath, err)
	}
	defer file.Close()

	// Upload the file with memory-efficient streaming
	log.Printf("Uploading %s to bucket as %s (size: %d bytes)", localPath, objectName, fileInfo.Size())

	// Use PutObject directly with the file as a ReadSeeker
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.config.BucketName),
		Key:    aws.String(objectName),
		Body:   file,
		Metadata: map[string]string{
			"upload-date": time.Now().Format(time.RFC3339),
		},
		ContentType: aws.String("application/octet-stream"),
	})

	if err != nil {
		return fmt.Errorf("failed to upload file %s: %w", localPath, err)
	}

	log.Printf("Successfully uploaded %s to bucket as %s", localPath, objectName)
	return nil
}

// UploadDirectory uploads all files in a directory to the bucket
func (s *S3Client) UploadDirectory(ctx context.Context, dirPath string) error {
	if err := s.EnsureBucketExists(ctx); err != nil {
		return err
	}

	// Calculate cutoff time for old logs (14 days)
	oldLogsCutoff := time.Now().AddDate(0, 0, -14)
	log.Printf("Uploading files from %s to R2 bucket %s", dirPath, s.config.BucketName)
	log.Printf("Files older than %s will be uploaded as 'archive/' objects", oldLogsCutoff.Format("2006-01-02"))

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

		// If the file is older than 14 days, put it in an "archive" folder
		if info.ModTime().Before(oldLogsCutoff) {
			objectName = "archive/" + objectName
			log.Printf("File %s is older than 14 days, uploading to archive folder", path)
		}

		return s.UploadFile(ctx, path, objectName)
	})
}

// CleanupOldBackups is now deprecated as we're using R2 lifecycle policies instead
// This function is kept for backward compatibility but doesn't delete objects anymore
func (s *S3Client) CleanupOldBackups(ctx context.Context) error {
	log.Printf("CleanupOldBackups is deprecated - using R2 lifecycle policies instead")
	log.Printf("Objects in the 'archive/' prefix should have a lifecycle policy configured in R2")
	return nil
}
