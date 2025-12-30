package slack

// conversationResponse represents a Slack conversation (DM or channel) from API
type conversationResponse struct {
	ID            string `json:"id"`
	User          string `json:"user"`            // For DMs, this is the other user's ID
	IsUserDeleted bool   `json:"is_user_deleted"` // Whether the user has been deleted
}

// conversationsListResponse represents the API response from conversations.list
type conversationsListResponse struct {
	OK       bool                   `json:"ok"`
	Channels []conversationResponse `json:"channels"`
	Error    string                 `json:"error"`
}

// messageResponse represents a single Slack message from API
type messageResponse struct {
	Type      string `json:"type"`
	User      string `json:"user"`
	Text      string `json:"text"`
	Timestamp string `json:"ts"`
}

// conversationsHistoryResponse represents the API response from conversations.history
type conversationsHistoryResponse struct {
	OK       bool              `json:"ok"`
	Messages []messageResponse `json:"messages"`
	Error    string            `json:"error"`
}

// userResponse represents a Slack user from API
type userResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	RealName string `json:"real_name"`
}

// usersInfoResponse represents the API response from users.info
type usersInfoResponse struct {
	OK    bool         `json:"ok"`
	User  userResponse `json:"user"`
	Error string       `json:"error"`
}

// authTestResponse represents the API response from auth.test
type authTestResponse struct {
	OK     bool   `json:"ok"`
	URL    string `json:"url"`
	Team   string `json:"team"`
	User   string `json:"user"`
	TeamID string `json:"team_id"`
	UserID string `json:"user_id"`
	Error  string `json:"error"`
}
