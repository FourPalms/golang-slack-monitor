package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/FourPalms/golang-slack-monitor"
	"github.com/FourPalms/golang-slack-monitor/notification"
	"github.com/FourPalms/golang-slack-monitor/slack"
	"github.com/FourPalms/golang-slack-monitor/storage"
)

const (
	defaultPollIntervalSecs = 60
	defaultDMsOnly          = true
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime)
	log.Println("Slack Monitor starting...")

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create implementations
	slackClient := slack.NewClient(config.Slack.XoxcToken, config.Slack.XoxdToken)
	notifier := notification.NewService(config.Notifications.NtfyTopic)
	stateStore := storage.NewFileStore()

	// Create monitor with injected dependencies
	mon := monitor.NewMonitor(slackClient, notifier, stateStore, config)

	// Set up context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT and SIGTERM for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v, shutting down gracefully...", sig)
		cancel()
	}()

	// Run the monitor
	log.Println("Starting monitoring...")
	if err := mon.Run(ctx); err != nil {
		log.Fatalf("Monitor error: %v", err)
	}

	log.Println("Monitoring stopped")
}

// loadConfig loads and validates the configuration file
func loadConfig() (*monitor.Config, error) {
	// Get home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Construct config path
	configPath := filepath.Join(home, ".slack-monitor", "config.json")

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file at %s: %w\nPlease create config file with your Slack tokens", configPath, err)
	}

	// Parse JSON
	var config monitor.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate required fields
	if config.Slack.XoxcToken == "" {
		return nil, fmt.Errorf("slack.xoxc_token is required in config")
	}
	if config.Slack.XoxdToken == "" {
		return nil, fmt.Errorf("slack.xoxd_token is required in config")
	}
	if config.Notifications.NtfyTopic == "" {
		return nil, fmt.Errorf("notifications.ntfy_topic is required in config")
	}

	// Set defaults
	if config.Slack.PollIntervalSecs == 0 {
		config.Slack.PollIntervalSecs = defaultPollIntervalSecs
	}
	if !config.Monitor.DMsOnly {
		config.Monitor.DMsOnly = defaultDMsOnly
	}

	return &config, nil
}
