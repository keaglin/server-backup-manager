package monitoring

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
)

// DiskStatus represents the current disk space status
type DiskStatus struct {
	Path        string    `json:"path"`
	Total       uint64    `json:"total_bytes"`
	Used        uint64    `json:"used_bytes"`
	Free        uint64    `json:"free_bytes"`
	UsedPercent float64   `json:"used_percent"`
	Timestamp   time.Time `json:"timestamp"`
	IsAlert     bool      `json:"is_alert"`
}

// FormatBytes converts bytes to a human-readable string
func (ds *DiskStatus) FormatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// Summary returns a human-readable summary of disk status
func (ds *DiskStatus) Summary() string {
	return fmt.Sprintf(
		"Disk space for %s: %s used out of %s (%.2f%%) - %s free",
		ds.Path,
		ds.FormatBytes(ds.Used),
		ds.FormatBytes(ds.Total),
		ds.UsedPercent,
		ds.FormatBytes(ds.Free),
	)
}

// CheckDiskSpace checks disk space usage for a given path
func CheckDiskSpace(path string, thresholdPercent float64) (*DiskStatus, error) {
	usage, err := disk.Usage(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk usage for %s: %w", path, err)
	}

	status := &DiskStatus{
		Path:        path,
		Total:       usage.Total,
		Used:        usage.Used,
		Free:        usage.Free,
		UsedPercent: usage.UsedPercent,
		Timestamp:   time.Now(),
		IsAlert:     usage.UsedPercent > thresholdPercent,
	}

	return status, nil
}
