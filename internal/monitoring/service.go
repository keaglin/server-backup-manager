package monitoring

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/user/server-backup-manager/internal/notification"
)

// Config holds monitoring configuration
type Config struct {
	// Paths to monitor
	MonitorPaths []string
	// Threshold percentage (0-100) at which to trigger alerts
	DiskThresholdPercent float64
	// Check interval
	CheckInterval time.Duration
	// Webhook URLs for alerts
	WebhookURLs []string
	// Webhook timeout
	WebhookTimeout time.Duration
}

// Service handles disk space monitoring
type Service struct {
	config        Config
	webhookClient *notification.WebhookClient
	stopCh        chan struct{}
	wg            sync.WaitGroup
	running       bool
	mu            sync.Mutex
}

// NewService creates a new monitoring service
func NewService(config Config) *Service {
	return &Service{
		config:        config,
		webhookClient: notification.NewWebhookClient(config.WebhookURLs, config.WebhookTimeout),
		stopCh:        make(chan struct{}),
	}
}

// Start starts the monitoring service
func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("monitoring service is already running")
	}

	s.running = true
	s.wg.Add(1)

	go s.monitorLoop()

	log.Printf("Disk space monitoring started. Checking %d paths every %s with threshold %.2f%%",
		len(s.config.MonitorPaths), s.config.CheckInterval, s.config.DiskThresholdPercent)

	return nil
}

// Stop stops the monitoring service
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	close(s.stopCh)
	s.wg.Wait()
	s.running = false

	log.Println("Disk space monitoring stopped")
}

// monitorLoop periodically checks disk space
func (s *Service) monitorLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.CheckInterval)
	defer ticker.Stop()

	// Run an initial check
	s.checkAllPaths()

	for {
		select {
		case <-ticker.C:
			s.checkAllPaths()
		case <-s.stopCh:
			return
		}
	}
}

// checkAllPaths checks disk space for all configured paths
func (s *Service) checkAllPaths() {
	for _, path := range s.config.MonitorPaths {
		status, err := CheckDiskSpace(path, s.config.DiskThresholdPercent)
		if err != nil {
			log.Printf("Error checking disk space for %s: %v", path, err)
			continue
		}

		// Log the current status
		log.Println(status.Summary())

		// If disk usage is above threshold, send an alert
		if status.IsAlert {
			message := fmt.Sprintf("Disk space alert: %s is at %.2f%% (threshold: %.2f%%)",
				path, status.UsedPercent, s.config.DiskThresholdPercent)

			if err := s.webhookClient.Send("disk_space_alert", message, status); err != nil {
				log.Printf("Failed to send disk space alert: %v", err)
			} else {
				log.Printf("Sent disk space alert for %s", path)
			}
		}
	}
}
