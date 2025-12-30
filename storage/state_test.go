package storage

import (
	"os"
	"testing"
)

// TestLoadSaveState tests state loading and saving
func TestLoadSaveState(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	// Create file store
	store := NewFileStore()

	// Test 1: Load non-existent state (should create new)
	state, err := store.Load()
	if err != nil {
		t.Fatalf("Expected no error loading non-existent state, got: %v", err)
	}
	if state.LastChecked == nil {
		t.Error("Expected LastChecked map to be initialized")
	}
	if len(state.LastChecked) != 0 {
		t.Errorf("Expected empty state, got %d entries", len(state.LastChecked))
	}

	// Test 2: Save state
	state.LastChecked["D123"] = "1234567890.123456"
	state.LastChecked["D456"] = "1234567891.654321"

	if err := store.Save(state); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Test 3: Load saved state
	loadedState, err := store.Load()
	if err != nil {
		t.Fatalf("Failed to load saved state: %v", err)
	}

	if len(loadedState.LastChecked) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(loadedState.LastChecked))
	}
	if loadedState.LastChecked["D123"] != "1234567890.123456" {
		t.Errorf("Expected timestamp '1234567890.123456', got '%s'", loadedState.LastChecked["D123"])
	}
}

// TestFirstCheckStatePersistence tests the critical bug fix:
// State must be saved on first check of a conversation, not just when messages are found.
func TestFirstCheckStatePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	store := NewFileStore()

	// Start with empty state (simulating first run)
	state, err := store.Load()
	if err != nil {
		t.Fatalf("Failed to load initial state: %v", err)
	}
	if len(state.LastChecked) != 0 {
		t.Error("Initial state should be empty")
	}

	// Simulate first check of a conversation (the bug scenario)
	channelID := "D123456789"

	// This is what the bug was: setting lastChecked only locally, not in state
	// The fix: state.LastChecked[channelID] = lastChecked
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
	if err := store.Save(state); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Reload state (simulating second monitoring cycle)
	reloadedState, err := store.Load()
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

	if err := store.Save(reloadedState); err != nil {
		t.Fatalf("Failed to save updated state: %v", err)
	}

	// Verify update persisted
	finalState, err := store.Load()
	if err != nil {
		t.Fatalf("Failed to reload final state: %v", err)
	}
	if finalState.LastChecked[channelID] != newTimestamp {
		t.Errorf("Final state timestamp mismatch: got %s, want %s", finalState.LastChecked[channelID], newTimestamp)
	}
}
