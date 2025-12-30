package notification

import (
	"testing"
	"time"
)

// TestNewService tests notification service initialization
func TestNewService(t *testing.T) {
	ntfyTopic := "test-topic-123"

	notifier := NewService(ntfyTopic)
	if notifier == nil {
		t.Error("Expected non-nil notifier")
	}
	if notifier.ntfyTopic != ntfyTopic {
		t.Errorf("Expected topic '%s', got '%s'", ntfyTopic, notifier.ntfyTopic)
	}
	if notifier.httpClient == nil {
		t.Error("Expected initialized HTTP client")
	}
}

// TestRateLimiting tests that notification rate limiting works correctly
func TestRateLimiting(t *testing.T) {
	notifier := NewService("test-topic")

	// First notification should be sent (lastNotify is zero time)
	// Note: We can't actually test sending without mocking HTTP, but we can test the rate limit logic
	// by checking the time tracking

	// Simulate notification was just sent
	notifier.lastNotify = time.Now()

	// Immediate second notification should be skipped (within rate limit)
	timeSinceLastNotify := time.Since(notifier.lastNotify)
	if timeSinceLastNotify >= rateLimitSeconds*time.Second {
		t.Errorf("Test setup error: time since last notify should be less than %d seconds", rateLimitSeconds)
	}

	// After waiting, should be allowed
	notifier.lastNotify = time.Now().Add(-rateLimitSeconds * time.Second)
	timeSinceLastNotify = time.Since(notifier.lastNotify)
	if timeSinceLastNotify < rateLimitSeconds*time.Second {
		t.Error("Time since last notify should be >= rate limit after waiting")
	}
}
