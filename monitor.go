package monitor

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Message represents a message in a Slack conversation
type Message struct {
	Timestamp string // Slack message timestamp (unique ID)
	User      string // User ID who sent the message
	Text      string // Message text content
	Type      string // Message type (e.g., "message")
}

// Conversation represents a Slack DM conversation
type Conversation struct {
	ID   string // Channel ID (e.g., "D06...")
	User string // Other user's ID in the DM
}

// State represents the monitoring state - tracks last checked timestamp per conversation
type State struct {
	LastChecked map[string]string // channel_id -> timestamp
}

// User represents a Slack user
type User struct {
	ID       string
	Name     string
	RealName string
}

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

// SlackClient defines the interface for Slack API operations
type SlackClient interface {
	// TestAuth validates authentication and returns the authenticated user ID
	TestAuth() (string, error)

	// GetDMConversations returns all DM conversations
	GetDMConversations() ([]Conversation, error)

	// GetConversationHistory fetches messages since the given timestamp
	GetConversationHistory(channelID, oldestTS string) ([]Message, error)

	// GetUserInfo fetches information about a user
	GetUserInfo(userID string) (*User, error)

	// GetAuthenticatedUserID returns the ID of the authenticated user
	GetAuthenticatedUserID() string
}

// Notifier defines the interface for sending notifications
type Notifier interface {
	// SendNotification sends a notification message
	SendNotification(message string) error
}

// StateStore defines the interface for state persistence
type StateStore interface {
	// Load loads the state from storage
	Load() (*State, error)

	// Save persists the state to storage
	Save(state *State) error
}

// Monitor represents the core monitoring logic
type Monitor struct {
	slackClient SlackClient
	notifier    Notifier
	stateStore  StateStore
	config      *Config
}

// NewMonitor creates a new Monitor instance
func NewMonitor(slackClient SlackClient, notifier Notifier, stateStore StateStore, config *Config) *Monitor {
	return &Monitor{
		slackClient: slackClient,
		notifier:    notifier,
		stateStore:  stateStore,
		config:      config,
	}
}

// Run starts the monitoring loop
func (m *Monitor) Run(ctx context.Context) error {
	// Validate authentication
	userID, err := m.slackClient.TestAuth()
	if err != nil {
		return err
	}
	_ = userID // Will be used for message filtering

	// Load state
	state, err := m.stateStore.Load()
	if err != nil {
		return err
	}

	pollInterval := time.Duration(m.config.Slack.PollIntervalSecs) * time.Second
	log.Println("Starting monitoring...")

	// Use check-then-wait pattern to prevent overlapping cycles
	for {
		// Check for cancellation before starting cycle
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// Run check cycle
		log.Println("Checking for new messages...")
		cycleStart := time.Now()
		if err := m.checkAllConversations(ctx, state); err != nil {
			// Log error but continue monitoring
			log.Printf("Error checking conversations: %v", err)
		}
		cycleDuration := time.Since(cycleStart)

		log.Printf("Check cycle completed in %dms, waiting %ds before next cycle", cycleDuration.Milliseconds(), int(pollInterval.Seconds()))

		// Wait for configured interval AFTER check completes
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(pollInterval):
			// Next cycle will start
		}
	}
}

// checkAllConversations checks all DM conversations for new messages
func (m *Monitor) checkAllConversations(ctx context.Context, state *State) error {
	// Get all DM conversations
	conversations, err := m.slackClient.GetDMConversations()
	if err != nil {
		return err
	}

	log.Printf("Checking %d DM conversation(s)", len(conversations))

	// Check each conversation for new messages
	for _, conv := range conversations {
		// Check for cancellation before each conversation
		select {
		case <-ctx.Done():
			return nil
		default:
			// Continue processing
		}

		if err := m.checkConversation(conv, state); err != nil {
			// Log error but continue checking other conversations
			continue
		}
	}

	// Save state after each check cycle
	if err := m.stateStore.Save(state); err != nil {
		return err
	}
	log.Printf("State saved (%d conversations tracked)", len(state.LastChecked))
	return nil
}

// checkConversation checks a single conversation for new messages
func (m *Monitor) checkConversation(conv Conversation, state *State) error {
	// Get last checked timestamp for this conversation
	lastChecked, exists := state.LastChecked[conv.ID]
	if !exists {
		// First time checking this conversation, start from now to avoid backlog spam
		lastChecked = formatTimestamp(time.Now())
		state.LastChecked[conv.ID] = lastChecked
	}

	// Fetch messages since last check
	messages, err := m.slackClient.GetConversationHistory(conv.ID, lastChecked)
	if err != nil {
		return err
	}

	// Process messages in reverse order (oldest first)
	newCount := 0
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]

		// Skip non-user messages and our own messages
		if msg.User == "" || msg.Type != "message" || msg.User == m.slackClient.GetAuthenticatedUserID() {
			continue
		}

		// Get user info for sender name
		user, err := m.slackClient.GetUserInfo(msg.User)
		if err != nil {
			user = &User{Name: msg.User} // Fallback to user ID
		}

		// Format notification message
		displayName := user.RealName
		if displayName == "" {
			displayName = user.Name
		}
		notificationMsg := formatNotification(displayName, msg.Text)

		// Send notification
		if err := m.notifier.SendNotification(notificationMsg); err != nil {
			// Log error but continue processing
			_ = err
		}

		newCount++

		// Update last checked to this message's timestamp
		state.LastChecked[conv.ID] = msg.Timestamp
	}

	if newCount == 0 {
		// No new messages - update timestamp to now to avoid re-checking
		state.LastChecked[conv.ID] = formatTimestamp(time.Now())
	}

	return nil
}

// formatTimestamp formats a time.Time as a Slack timestamp
func formatTimestamp(t time.Time) string {
	return formatFloat(float64(t.Unix()))
}

// formatFloat formats a float with 6 decimal places (Slack timestamp format)
func formatFloat(f float64) string {
	return fmt.Sprintf("%.6f", f)
}

// formatNotification formats a message for notification
func formatNotification(userName, messageText string) string {
	const maxLength = 500
	if len(messageText) > maxLength {
		messageText = messageText[:maxLength-3] + "..."
	}
	return fmt.Sprintf("DM from %s: %s", userName, messageText)
}
