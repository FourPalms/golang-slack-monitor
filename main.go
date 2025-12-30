package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Constants for configurable behavior
const (
	DefaultPollIntervalSecs   = 60
	MaxMessagePreviewLength   = 100
	NotificationRateLimitSec  = 2
	SlackAPIConversationLimit = 200
	SlackAPIMessageLimit      = 100
	DefaultDMsOnly            = true
)

// Config represents the application configuration
type Config struct {
	Slack struct {
		XoxcToken        string `json:"xoxc_token"`
		XoxdToken        string `json:"xoxd_token"`
		WorkspaceID      string `json:"workspace_id"`
		PollIntervalSecs int    `json:"poll_interval_seconds"`
	} `json:"slack"`
	Notifications struct {
		NtfyTopic string `json:"ntfy_topic"`
	} `json:"notifications"`
	Monitor struct {
		DMsOnly bool `json:"dms_only"`
	} `json:"monitor"`
}

// State represents the persistent state tracking last checked timestamps
type State struct {
	LastChecked map[string]string `json:"last_checked"` // channel_id -> timestamp
}

// SlackConversation represents a Slack conversation (DM or channel)
type SlackConversation struct {
	ID   string `json:"id"`
	User string `json:"user"` // For DMs, this is the other user's ID
}

// SlackConversationsResponse represents the API response from conversations.list
type SlackConversationsResponse struct {
	OK       bool                `json:"ok"`
	Channels []SlackConversation `json:"channels"`
	Error    string              `json:"error"`
}

// SlackMessage represents a single Slack message
type SlackMessage struct {
	Type      string `json:"type"`
	User      string `json:"user"`
	Text      string `json:"text"`
	Timestamp string `json:"ts"`
}

// SlackHistoryResponse represents the API response from conversations.history
type SlackHistoryResponse struct {
	OK       bool           `json:"ok"`
	Messages []SlackMessage `json:"messages"`
	Error    string         `json:"error"`
}

// SlackUser represents a Slack user
type SlackUser struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	RealName string `json:"real_name"`
}

// SlackUserResponse represents the API response from users.info
type SlackUserResponse struct {
	OK    bool      `json:"ok"`
	User  SlackUser `json:"user"`
	Error string    `json:"error"`
}

// SlackAuthResponse represents the API response from auth.test
type SlackAuthResponse struct {
	OK     bool   `json:"ok"`
	URL    string `json:"url"`
	Team   string `json:"team"`
	User   string `json:"user"`
	TeamID string `json:"team_id"`
	UserID string `json:"user_id"`
	Error  string `json:"error"`
}

// getMonitorDir returns the .slack-monitor directory path
func getMonitorDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}
	return filepath.Join(home, ".slack-monitor")
}

// getConfigPath returns the path to the config file
func getConfigPath() string {
	return filepath.Join(getMonitorDir(), "config.json")
}

// getStatePath returns the path to the state file
func getStatePath() string {
	return filepath.Join(getMonitorDir(), "state.json")
}

// loadConfig loads and validates the configuration file
func loadConfig() (*Config, error) {
	configPath := getConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file at %s: %w\nPlease create config file with your Slack tokens", configPath, err)
	}

	var config Config
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
		config.Slack.PollIntervalSecs = DefaultPollIntervalSecs
	}
	if config.Monitor.DMsOnly == false {
		config.Monitor.DMsOnly = DefaultDMsOnly
	}

	return &config, nil
}

// loadState loads the persistent state from disk
func loadState() (*State, error) {
	statePath := getStatePath()

	// If state file doesn't exist, create new empty state
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		log.Println("No existing state file found, creating new state")
		return &State{
			LastChecked: make(map[string]string),
		}, nil
	}

	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	// Ensure map is initialized
	if state.LastChecked == nil {
		state.LastChecked = make(map[string]string)
	}

	log.Printf("State loaded successfully (%d conversations tracked)", len(state.LastChecked))
	return &state, nil
}

// saveState saves the state to disk atomically
func saveState(state *State) error {
	statePath := getStatePath()

	// Ensure directory exists
	monitorDir := getMonitorDir()
	if err := os.MkdirAll(monitorDir, 0700); err != nil {
		return fmt.Errorf("failed to create monitor directory: %w", err)
	}

	// Marshal state to JSON
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temporary file first, then rename (atomic)
	tempPath := statePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temporary state file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, statePath); err != nil {
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	return nil
}

// SlackClient handles API calls to Slack
type SlackClient struct {
	xoxcToken           string
	xoxdToken           string
	httpClient          *http.Client
	authenticatedUserID string // ID of the authenticated user (to filter own messages)
}

// newSlackClient creates a new Slack API client
func newSlackClient(config *Config) *SlackClient {
	return &SlackClient{
		xoxcToken: config.Slack.XoxcToken,
		xoxdToken: config.Slack.XoxdToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// makeSlackRequest makes an authenticated request to the Slack API
func (c *SlackClient) makeSlackRequest(method, endpoint string, params url.Values) ([]byte, error) {
	apiURL := "https://slack.com/api/" + endpoint

	var req *http.Request
	var err error

	if method == "GET" {
		if len(params) > 0 {
			apiURL += "?" + params.Encode()
		}
		req, err = http.NewRequest("GET", apiURL, nil)
	} else {
		req, err = http.NewRequest("POST", apiURL, strings.NewReader(params.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication - stealth mode uses both tokens AND two cookies
	// Slack requires both "d" and "d-s" cookies (discovered from rusq/slackdump library)
	// CRITICAL: xoxd goes in "d" cookie, xoxc goes in token parameter (not the other way around!)
	dCookie := &http.Cookie{
		Name:  "d",
		Value: c.xoxdToken, // xoxd token goes in "d" cookie
	}
	req.AddCookie(dCookie)

	// d-s cookie is a timestamp (current Unix time - 10 seconds)
	dsCookie := &http.Cookie{
		Name:  "d-s",
		Value: fmt.Sprintf("%d", time.Now().Unix()-10),
	}
	req.AddCookie(dsCookie)

	// For POST requests, token is in the body parameters (not Authorization header)
	// For GET requests, we may need to add it as a query parameter
	// Authorization header is NOT used with stealth mode cookies

	// Add browser User-Agent to match slack-mcp-server
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// getDMConversations fetches all DM conversations
func (c *SlackClient) getDMConversations() ([]SlackConversation, error) {
	params := url.Values{}
	params.Set("types", "im")
	params.Set("exclude_archived", "true")
	params.Set("limit", fmt.Sprintf("%d", SlackAPIConversationLimit))
	params.Set("token", c.xoxcToken) // GET requests need token as query parameter

	body, err := c.makeSlackRequest("GET", "conversations.list", params)
	if err != nil {
		return nil, err
	}

	var response SlackConversationsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse conversations response: %w", err)
	}

	if !response.OK {
		return nil, fmt.Errorf("Slack API error: %s", response.Error)
	}

	return response.Channels, nil
}

// getConversationHistory fetches messages from a conversation since a given timestamp
func (c *SlackClient) getConversationHistory(channelID, oldestTS string) ([]SlackMessage, error) {
	params := url.Values{}
	params.Set("channel", channelID)
	if oldestTS != "" {
		params.Set("oldest", oldestTS)
	}
	params.Set("limit", fmt.Sprintf("%d", SlackAPIMessageLimit))
	params.Set("token", c.xoxcToken) // GET requests need token as query parameter

	body, err := c.makeSlackRequest("GET", "conversations.history", params)
	if err != nil {
		return nil, err
	}

	var response SlackHistoryResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse history response: %w", err)
	}

	if !response.OK {
		return nil, fmt.Errorf("Slack API error: %s", response.Error)
	}

	return response.Messages, nil
}

// testAuth validates the authentication tokens and returns the authenticated user ID
func (c *SlackClient) testAuth() (string, error) {
	// slack-go uses POST with token as a parameter for auth.test
	params := url.Values{
		"token": {c.xoxcToken},
	}
	body, err := c.makeSlackRequest("POST", "auth.test", params)
	if err != nil {
		return "", err
	}

	var response SlackAuthResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse auth response: %w", err)
	}

	if !response.OK {
		return "", fmt.Errorf("Slack authentication failed: %s", response.Error)
	}

	log.Printf("Authenticated as %s (%s) in workspace %s", response.User, response.UserID, response.Team)
	return response.UserID, nil
}

// getUserInfo fetches information about a user
func (c *SlackClient) getUserInfo(userID string) (*SlackUser, error) {
	params := url.Values{}
	params.Set("user", userID)

	body, err := c.makeSlackRequest("GET", "users.info", params)
	if err != nil {
		return nil, err
	}

	var response SlackUserResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse user response: %w", err)
	}

	if !response.OK {
		return nil, fmt.Errorf("Slack API error: %s", response.Error)
	}

	return &response.User, nil
}

// NotificationService handles sending notifications
type NotificationService struct {
	ntfyTopic  string
	httpClient *http.Client
	lastNotify time.Time // For rate limiting
}

// newNotificationService creates a new notification service
func newNotificationService(ntfyTopic string) *NotificationService {
	return &NotificationService{
		ntfyTopic: ntfyTopic,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		lastNotify: time.Time{},
	}
}

// sendNotification sends a notification to ntfy.sh
func (n *NotificationService) sendNotification(message string) error {
	// Rate limiting: prevent notification spam
	if time.Since(n.lastNotify) < NotificationRateLimitSec*time.Second {
		log.Println("Rate limiting: skipping notification")
		return nil
	}

	ntfyURL := fmt.Sprintf("https://ntfy.sh/%s", n.ntfyTopic)

	req, err := http.NewRequest("POST", ntfyURL, strings.NewReader(message))
	if err != nil {
		return fmt.Errorf("failed to create notification request: %w", err)
	}

	req.Header.Set("Title", "Slack Monitor")
	req.Header.Set("Priority", "default")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ntfy returned status %d: %s", resp.StatusCode, string(body))
	}

	n.lastNotify = time.Now()
	log.Printf("Notification sent: %s", message)
	return nil
}

// formatMessage formats a Slack message for notification
func formatMessage(userName, messageText string) string {
	// Truncate message for mobile display
	if len(messageText) > MaxMessagePreviewLength {
		messageText = messageText[:MaxMessagePreviewLength-3] + "..."
	}
	return fmt.Sprintf("DM from %s: %s", userName, messageText)
}

// checkForNewMessages checks a single DM conversation for new messages
func checkForNewMessages(slackClient *SlackClient, notifier *NotificationService, state *State, channelID, userID string) error {
	// Get last checked timestamp for this conversation
	lastChecked, exists := state.LastChecked[channelID]
	if !exists {
		// First time checking this conversation, start from now
		lastChecked = fmt.Sprintf("%.6f", float64(time.Now().Unix()))
		log.Printf("First time checking %s, starting from now", channelID)
	}

	// Fetch messages since last check
	messages, err := slackClient.getConversationHistory(channelID, lastChecked)
	if err != nil {
		return fmt.Errorf("failed to get conversation history for %s: %w", channelID, err)
	}

	// Process messages in reverse order (oldest first)
	newCount := 0
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]

		// Skip non-user messages and our own messages
		if msg.User == "" || msg.Type != "message" || msg.User == slackClient.authenticatedUserID {
			continue
		}

		// Get user info for sender name
		user, err := slackClient.getUserInfo(msg.User)
		if err != nil {
			log.Printf("Warning: failed to get user info for %s: %v", msg.User, err)
			user = &SlackUser{Name: msg.User} // Fallback to user ID
		}

		// Send notification
		displayName := user.RealName
		if displayName == "" {
			displayName = user.Name
		}
		notificationMsg := formatMessage(displayName, msg.Text)
		if err := notifier.sendNotification(notificationMsg); err != nil {
			log.Printf("Warning: failed to send notification: %v", err)
		}

		newCount++

		// Update last checked to this message's timestamp
		state.LastChecked[channelID] = msg.Timestamp
	}

	if newCount > 0 {
		log.Printf("Found %d new message(s) in %s", newCount, channelID)
	}

	// If no new messages but we checked, update timestamp to now to avoid re-checking old messages
	if newCount == 0 && exists {
		state.LastChecked[channelID] = fmt.Sprintf("%.6f", float64(time.Now().Unix()))
	}

	return nil
}

// monitorLoop runs the main monitoring loop
func monitorLoop(ctx context.Context, config *Config, state *State) {
	log.Println("Starting monitoring loop...")

	slackClient := newSlackClient(config)

	// Validate authentication and get authenticated user ID
	userID, err := slackClient.testAuth()
	if err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}
	slackClient.authenticatedUserID = userID

	notifier := newNotificationService(config.Notifications.NtfyTopic)

	ticker := time.NewTicker(time.Duration(config.Slack.PollIntervalSecs) * time.Second)
	defer ticker.Stop()

	// Run first check immediately
	checkAllConversations(slackClient, notifier, state)

	for {
		select {
		case <-ctx.Done():
			log.Println("Monitoring loop stopping...")
			return
		case <-ticker.C:
			checkAllConversations(slackClient, notifier, state)
		}
	}
}

// checkAllConversations checks all DM conversations for new messages
func checkAllConversations(slackClient *SlackClient, notifier *NotificationService, state *State) {
	log.Println("Checking for new messages...")

	// Get all DM conversations
	conversations, err := slackClient.getDMConversations()
	if err != nil {
		log.Printf("Error getting conversations: %v", err)
		return
	}

	log.Printf("Checking %d DM conversation(s)", len(conversations))

	// Check each conversation for new messages
	for _, conv := range conversations {
		if err := checkForNewMessages(slackClient, notifier, state, conv.ID, conv.User); err != nil {
			log.Printf("Error checking conversation %s: %v", conv.ID, err)
			continue
		}
	}

	// Save state after each check cycle
	if err := saveState(state); err != nil {
		log.Printf("Warning: failed to save state: %v", err)
	}

	log.Println("Check cycle complete")
}

func main() {
	log.Println("Slack Monitor starting...")

	// Load config
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}
	log.Printf("Config loaded successfully (poll interval: %ds)", config.Slack.PollIntervalSecs)

	// Load state
	state, err := loadState()
	if err != nil {
		log.Fatalf("State error: %v", err)
	}

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v, shutting down gracefully...", sig)
		cancel()
	}()

	// Start monitoring loop
	monitorLoop(ctx, config, state)

	// Save state on exit
	if err := saveState(state); err != nil {
		log.Printf("Warning: failed to save state: %v", err)
	}

	log.Println("Slack Monitor stopped.")
}
