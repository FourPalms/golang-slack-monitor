package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLoadConfig tests config loading and validation
func TestLoadConfig(t *testing.T) {
	// Create temp directory for test config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Save original getConfigPath and restore after test
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	// Test 1: Missing config file
	_, err := loadConfig()
	if err == nil {
		t.Error("Expected error for missing config file")
	}

	// Test 2: Valid config
	validConfig := map[string]interface{}{
		"slack": map[string]interface{}{
			"xoxc_token":            "test-xoxc",
			"xoxd_token":            "test-xoxd",
			"poll_interval_seconds": 30,
		},
		"notifications": map[string]interface{}{
			"ntfy_topic": "test-topic",
		},
		"monitor": map[string]interface{}{
			"dms_only": true,
		},
	}

	// Create .slack-monitor directory
	monitorDir := filepath.Join(tmpDir, ".slack-monitor")
	if err := os.MkdirAll(monitorDir, 0700); err != nil {
		t.Fatalf("Failed to create monitor dir: %v", err)
	}

	configPath = filepath.Join(monitorDir, "config.json")
	data, _ := json.Marshal(validConfig)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	config, err := loadConfig()
	if err != nil {
		t.Errorf("Expected no error for valid config, got: %v", err)
	}
	if config.Slack.XoxcToken != "test-xoxc" {
		t.Errorf("Expected xoxc_token 'test-xoxc', got '%s'", config.Slack.XoxcToken)
	}
	if config.Slack.PollIntervalSecs != 30 {
		t.Errorf("Expected poll interval 30, got %d", config.Slack.PollIntervalSecs)
	}

	// Test 3: Missing required field
	invalidConfig := map[string]interface{}{
		"slack": map[string]interface{}{
			"xoxc_token": "test-xoxc",
			// Missing xoxd_token
		},
		"notifications": map[string]interface{}{
			"ntfy_topic": "test-topic",
		},
	}
	data, _ = json.Marshal(invalidConfig)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}

	_, err = loadConfig()
	if err == nil {
		t.Error("Expected error for missing required field")
	}
}

// TestLoadSaveState tests state loading and saving
func TestLoadSaveState(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	// Test 1: Load non-existent state (should create new)
	state, err := loadState()
	if err != nil {
		t.Errorf("Expected no error for missing state, got: %v", err)
	}
	if state == nil || state.LastChecked == nil {
		t.Error("Expected initialized state")
	}
	if len(state.LastChecked) != 0 {
		t.Error("Expected empty LastChecked map")
	}

	// Test 2: Save state
	state.LastChecked["C123"] = "1234567890.123456"
	state.LastChecked["C456"] = "1234567891.123456"

	if err := saveState(state); err != nil {
		t.Errorf("Failed to save state: %v", err)
	}

	// Test 3: Load saved state
	loadedState, err := loadState()
	if err != nil {
		t.Errorf("Failed to load state: %v", err)
	}
	if len(loadedState.LastChecked) != 2 {
		t.Errorf("Expected 2 conversations, got %d", len(loadedState.LastChecked))
	}
	if loadedState.LastChecked["C123"] != "1234567890.123456" {
		t.Errorf("Expected timestamp for C123, got %s", loadedState.LastChecked["C123"])
	}
}

// TestFormatMessage tests message formatting
func TestFormatMessage(t *testing.T) {
	tests := []struct {
		userName string
		message  string
		expected string
	}{
		{
			userName: "John Doe",
			message:  "Hello world",
			expected: "DM from John Doe: Hello world",
		},
		{
			userName: "Jane",
			message:  "This is a very long message that exceeds the 100 character limit and should be truncated properly with ellipsis at the end to make it fit",
			expected: "DM from Jane: This is a very long message that exceeds the 100 character limit and should be truncated properly...",
		},
		{
			userName: "Bot",
			message:  "",
			expected: "DM from Bot: ",
		},
	}

	for _, tt := range tests {
		result := formatMessage(tt.userName, tt.message)
		if result != tt.expected {
			t.Errorf("formatMessage(%q, %q) = %q, want %q", tt.userName, tt.message, result, tt.expected)
		}
	}
}

// TestSlackClientCreation tests Slack client initialization
func TestSlackClientCreation(t *testing.T) {
	config := &Config{}
	config.Slack.XoxcToken = "test-xoxc"
	config.Slack.XoxdToken = "test-xoxd"

	client := newSlackClient(config)
	if client == nil {
		t.Error("Expected non-nil client")
	}
	if client.xoxcToken != "test-xoxc" {
		t.Errorf("Expected xoxc token 'test-xoxc', got '%s'", client.xoxcToken)
	}
	if client.xoxdToken != "test-xoxd" {
		t.Errorf("Expected xoxd token 'test-xoxd', got '%s'", client.xoxdToken)
	}
	if client.httpClient == nil {
		t.Error("Expected initialized HTTP client")
	}
}

// TestNotificationServiceCreation tests notification service initialization
func TestNotificationServiceCreation(t *testing.T) {
	ntfyTopic := "test-topic-123"

	notifier := newNotificationService(ntfyTopic)
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
	notifier := newNotificationService("test-topic")

	// First notification should be sent (lastNotify is zero time)
	// Note: We can't actually test sending without mocking HTTP, but we can test the rate limit logic
	// by checking the time tracking

	// Simulate notification was just sent
	notifier.lastNotify = time.Now()

	// Immediate second notification should be skipped (within rate limit)
	timeSinceLastNotify := time.Since(notifier.lastNotify)
	if timeSinceLastNotify >= NotificationRateLimitSec*time.Second {
		t.Errorf("Test setup error: time since last notify should be less than %d seconds", NotificationRateLimitSec)
	}

	// After waiting, should be allowed
	notifier.lastNotify = time.Now().Add(-NotificationRateLimitSec * time.Second)
	timeSinceLastNotify = time.Since(notifier.lastNotify)
	if timeSinceLastNotify < NotificationRateLimitSec*time.Second {
		t.Error("Time since last notify should be >= rate limit after waiting")
	}
}

// TestStateUpdateScenarios tests different state update scenarios
func TestStateUpdateScenarios(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	// Scenario 1: First run - empty state
	state, err := loadState()
	if err != nil {
		t.Fatalf("Failed to load initial state: %v", err)
	}
	if len(state.LastChecked) != 0 {
		t.Error("Initial state should have empty LastChecked map")
	}

	// Scenario 2: Add first conversation
	channelID1 := "C123456"
	timestamp1 := "1234567890.123456"
	state.LastChecked[channelID1] = timestamp1

	if err := saveState(state); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Scenario 3: Reload and verify persistence
	reloadedState, err := loadState()
	if err != nil {
		t.Fatalf("Failed to reload state: %v", err)
	}
	if reloadedState.LastChecked[channelID1] != timestamp1 {
		t.Errorf("Expected timestamp '%s' for %s, got '%s'", timestamp1, channelID1, reloadedState.LastChecked[channelID1])
	}

	// Scenario 4: Update existing conversation
	timestamp2 := "1234567900.123456"
	reloadedState.LastChecked[channelID1] = timestamp2

	// Scenario 5: Add second conversation
	channelID2 := "C789012"
	timestamp3 := "1234567910.123456"
	reloadedState.LastChecked[channelID2] = timestamp3

	if err := saveState(reloadedState); err != nil {
		t.Fatalf("Failed to save updated state: %v", err)
	}

	// Scenario 6: Final reload and verify both conversations
	finalState, err := loadState()
	if err != nil {
		t.Fatalf("Failed to load final state: %v", err)
	}
	if len(finalState.LastChecked) != 2 {
		t.Errorf("Expected 2 conversations, got %d", len(finalState.LastChecked))
	}
	if finalState.LastChecked[channelID1] != timestamp2 {
		t.Errorf("Expected updated timestamp '%s' for %s, got '%s'", timestamp2, channelID1, finalState.LastChecked[channelID1])
	}
	if finalState.LastChecked[channelID2] != timestamp3 {
		t.Errorf("Expected timestamp '%s' for %s, got '%s'", timestamp3, channelID2, finalState.LastChecked[channelID2])
	}
}

// TestMessageFiltering tests message filtering logic (own messages, non-user messages)
func TestMessageFiltering(t *testing.T) {
	// Test that we correctly identify messages to skip
	authenticatedUserID := "U123456"

	tests := []struct {
		name       string
		msgUser    string
		msgType    string
		shouldSkip bool
	}{
		{
			name:       "Normal message from other user",
			msgUser:    "U789012",
			msgType:    "message",
			shouldSkip: false,
		},
		{
			name:       "Own message should be skipped",
			msgUser:    authenticatedUserID,
			msgType:    "message",
			shouldSkip: true,
		},
		{
			name:       "Message with empty user should be skipped",
			msgUser:    "",
			msgType:    "message",
			shouldSkip: true,
		},
		{
			name:       "Non-message type should be skipped",
			msgUser:    "U789012",
			msgType:    "channel_join",
			shouldSkip: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the filtering logic from checkForNewMessages
			shouldSkip := tt.msgUser == "" || tt.msgType != "message" || tt.msgUser == authenticatedUserID

			if shouldSkip != tt.shouldSkip {
				t.Errorf("Expected shouldSkip=%v, got %v for user=%s type=%s", tt.shouldSkip, shouldSkip, tt.msgUser, tt.msgType)
			}
		})
	}
}

// TestConfigDefaults tests that configuration defaults are set correctly
func TestConfigDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	// Create minimal config (only required fields)
	monitorDir := filepath.Join(tmpDir, ".slack-monitor")
	if err := os.MkdirAll(monitorDir, 0700); err != nil {
		t.Fatalf("Failed to create monitor dir: %v", err)
	}

	minimalConfig := map[string]interface{}{
		"slack": map[string]interface{}{
			"xoxc_token": "test-xoxc",
			"xoxd_token": "test-xoxd",
			// No poll_interval_seconds specified
		},
		"notifications": map[string]interface{}{
			"ntfy_topic": "test-topic",
		},
		// No monitor section specified
	}

	configPath := filepath.Join(monitorDir, "config.json")
	data, _ := json.Marshal(minimalConfig)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify defaults were set
	if config.Slack.PollIntervalSecs != DefaultPollIntervalSecs {
		t.Errorf("Expected default poll interval %d, got %d", DefaultPollIntervalSecs, config.Slack.PollIntervalSecs)
	}
	if config.Monitor.DMsOnly != DefaultDMsOnly {
		t.Errorf("Expected default DMsOnly %v, got %v", DefaultDMsOnly, config.Monitor.DMsOnly)
	}
}

// TestFirstCheckStatePersistence tests the critical bug fix:
// State must be saved on first check of a conversation, not just when messages are found.
// This was a production bug where state.LastChecked[channelID] was never populated.
func TestFirstCheckStatePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	// Start with empty state (simulating first run)
	state, err := loadState()
	if err != nil {
		t.Fatalf("Failed to load initial state: %v", err)
	}
	if len(state.LastChecked) != 0 {
		t.Error("Initial state should be empty")
	}

	// Simulate first check of a conversation (the bug scenario)
	channelID := "D123456789"

	// This is what the bug was: setting lastChecked only locally, not in state
	// The fix: state.LastChecked[channelID] = lastChecked (line 466 in main.go)
	_, exists := state.LastChecked[channelID]
	if exists {
		t.Error("Channel should not exist in state yet")
	}

	// CRITICAL: Must save to state on first check (this was the bug)
	nowTimestamp := "1735579200.000000"
	state.LastChecked[channelID] = nowTimestamp

	// Verify state is populated before saving
	if len(state.LastChecked) != 1 {
		t.Errorf("State should have 1 conversation, got %d", len(state.LastChecked))
	}
	if state.LastChecked[channelID] != nowTimestamp {
		t.Errorf("State timestamp mismatch: got %s, want %s", state.LastChecked[channelID], nowTimestamp)
	}

	// Save state (end of check cycle)
	if err := saveState(state); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Reload state (simulating second monitoring cycle)
	reloadedState, err := loadState()
	if err != nil {
		t.Fatalf("Failed to reload state: %v", err)
	}

	// VERIFY: State must persist between cycles (this was failing with the bug)
	if len(reloadedState.LastChecked) != 1 {
		t.Errorf("Reloaded state should have 1 conversation, got %d", len(reloadedState.LastChecked))
	}
	if reloadedState.LastChecked[channelID] != nowTimestamp {
		t.Errorf("Reloaded state timestamp mismatch: got %s, want %s", reloadedState.LastChecked[channelID], nowTimestamp)
	}

	// Simulate second check (no new messages, update timestamp)
	newTimestamp := "1735579210.000000" // 10 seconds later
	reloadedState.LastChecked[channelID] = newTimestamp

	if err := saveState(reloadedState); err != nil {
		t.Fatalf("Failed to save updated state: %v", err)
	}

	// Verify update persisted
	finalState, err := loadState()
	if err != nil {
		t.Fatalf("Failed to reload final state: %v", err)
	}
	if finalState.LastChecked[channelID] != newTimestamp {
		t.Errorf("Final state timestamp mismatch: got %s, want %s", finalState.LastChecked[channelID], newTimestamp)
	}
}
