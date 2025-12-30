package slack

import (
	"testing"
)

// TestNewClient tests Slack client initialization
func TestNewClient(t *testing.T) {
	xoxcToken := "test-xoxc"
	xoxdToken := "test-xoxd"

	client := NewClient(xoxcToken, xoxdToken)
	if client == nil {
		t.Error("Expected non-nil client")
	}
	if client.xoxcToken != xoxcToken {
		t.Errorf("Expected xoxc token '%s', got '%s'", xoxcToken, client.xoxcToken)
	}
	if client.xoxdToken != xoxdToken {
		t.Errorf("Expected xoxd token '%s', got '%s'", xoxdToken, client.xoxdToken)
	}
	if client.httpClient == nil {
		t.Error("Expected initialized HTTP client")
	}
}
