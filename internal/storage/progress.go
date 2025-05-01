package storage

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// UploadTracker tracks which files have been uploaded
type UploadTracker struct {
	progressDir string
	progressMap map[string]struct{}
	mutex       sync.RWMutex
}

// NewUploadTracker creates a new upload tracker
func NewUploadTracker(backupDir string) (*UploadTracker, error) {
	// Create progress directory inside the backup directory
	progressDir := filepath.Join(backupDir, ".progress")
	if err := os.MkdirAll(progressDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create progress directory: %w", err)
	}

	tracker := &UploadTracker{
		progressDir: progressDir,
		progressMap: make(map[string]struct{}),
	}

	// Load existing progress
	if err := tracker.loadProgress(); err != nil {
		return nil, fmt.Errorf("failed to load progress: %w", err)
	}

	return tracker, nil
}

// loadProgress loads the progress from the progress file
func (t *UploadTracker) loadProgress() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Clear the map
	t.progressMap = make(map[string]struct{})

	// Get all progress files
	progressFiles, err := filepath.Glob(filepath.Join(t.progressDir, "*.progress"))
	if err != nil {
		return fmt.Errorf("failed to list progress files: %w", err)
	}

	// Load each progress file
	for _, progressFile := range progressFiles {
		file, err := os.Open(progressFile)
		if err != nil {
			// Skip files we can't open
			continue
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			filePath := scanner.Text()
			t.progressMap[filePath] = struct{}{}
		}

		if err := scanner.Err(); err != nil {
			return fmt.Errorf("error reading progress file: %w", err)
		}
	}

	return nil
}

// MarkUploaded marks a file as uploaded
func (t *UploadTracker) MarkUploaded(filePath, backupName string) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Add to memory map
	t.progressMap[filePath] = struct{}{}

	// Create progress file for this backup if it doesn't exist
	progressFile := filepath.Join(t.progressDir, backupName+".progress")
	file, err := os.OpenFile(progressFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open progress file: %w", err)
	}
	defer file.Close()

	// Write the file path to the progress file
	if _, err := file.WriteString(filePath + "\n"); err != nil {
		return fmt.Errorf("failed to write to progress file: %w", err)
	}

	return nil
}

// IsUploaded checks if a file has been uploaded
func (t *UploadTracker) IsUploaded(filePath string) bool {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	_, exists := t.progressMap[filePath]
	return exists
}

// GetUploadedCount returns the number of uploaded files
func (t *UploadTracker) GetUploadedCount() int {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return len(t.progressMap)
}
