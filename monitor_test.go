package monitor

import (
	"strings"
	"testing"
)

// TestFormatNotification tests message formatting for notifications
func TestFormatNotification(t *testing.T) {
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
			expected: "DM from Jane: This is a very long message that exceeds the 100 character limit and should be truncated properly with ellipsis at the end to make it fit",
		},
		{
			userName: "Bob",
			message:  strings.Repeat("a", 600), // 600 chars exceeds 500 limit
			expected: "DM from Bob: " + strings.Repeat("a", 497) + "...",
		},
		{
			userName: "Bot",
			message:  "",
			expected: "DM from Bot: ",
		},
	}

	for _, tt := range tests {
		result := formatNotification(tt.userName, tt.message)
		if result != tt.expected {
			t.Errorf("formatNotification(%q, %q) = %q, want %q", tt.userName, tt.message, result, tt.expected)
		}
	}
}
