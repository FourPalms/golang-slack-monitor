package notification

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	rateLimitSeconds = 2 // Minimum seconds between notifications
)

// Service implements the monitor.Notifier interface
type Service struct {
	ntfyTopic  string
	httpClient *http.Client
	lastNotify time.Time // For rate limiting
}

// NewService creates a new notification service
func NewService(ntfyTopic string) *Service {
	return &Service{
		ntfyTopic: ntfyTopic,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		lastNotify: time.Time{},
	}
}

// SendNotification sends a notification to ntfy.sh
func (s *Service) SendNotification(message string) error {
	// Rate limiting: prevent notification spam
	if time.Since(s.lastNotify) < rateLimitSeconds*time.Second {
		log.Println("Rate limiting: skipping notification")
		return nil
	}

	ntfyURL := fmt.Sprintf("https://ntfy.sh/%s", s.ntfyTopic)

	req, err := http.NewRequest("POST", ntfyURL, strings.NewReader(message))
	if err != nil {
		return fmt.Errorf("failed to create notification request: %w", err)
	}

	req.Header.Set("Title", "Slack Monitor")
	req.Header.Set("Priority", "default")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ntfy returned status %d: %s", resp.StatusCode, string(body))
	}

	s.lastNotify = time.Now()
	log.Printf("Notification sent: %s", message)
	return nil
}
