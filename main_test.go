package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
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
