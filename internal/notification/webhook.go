package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WebhookClient handles sending notifications via webhooks
type WebhookClient struct {
	URLs     []string
	Timeout  time.Duration
	client   *http.Client
	lastSent map[string]time.Time // Track last sent time per URL to avoid flooding
}

// WebhookPayload is the structure of the data sent to webhooks
type WebhookPayload struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Source    string      `json:"source"`
	Data      interface{} `json:"data"`
	Message   string      `json:"message"`
}

// NewWebhookClient creates a new webhook client
func NewWebhookClient(urls []string, timeout time.Duration) *WebhookClient {
	return &WebhookClient{
		URLs:    urls,
		Timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
		lastSent: make(map[string]time.Time),
	}
}

// Send sends a notification to all configured webhook URLs
func (w *WebhookClient) Send(alertType string, message string, data interface{}) error {
	if len(w.URLs) == 0 {
		return fmt.Errorf("no webhook URLs configured")
	}

	payload := WebhookPayload{
		Type:      alertType,
		Timestamp: time.Now(),
		Source:    "server-backup-manager",
		Data:      data,
		Message:   message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	var lastErr error
	for _, url := range w.URLs {
		// Check if we've sent to this URL recently (within 5 minutes)
		if lastSent, ok := w.lastSent[url]; ok {
			if time.Since(lastSent) < 5*time.Minute {
				continue // Skip this URL to avoid flooding
			}
		}

		resp, err := w.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			lastErr = fmt.Errorf("failed to send webhook to %s: %w", url, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("webhook to %s returned non-success status: %d", url, resp.StatusCode)
			continue
		}

		// Update last sent time
		w.lastSent[url] = time.Now()
	}

	return lastErr
}
