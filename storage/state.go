package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jeremyhunt/slack-monitor"
)

// FileStore implements the monitor.StateStore interface using JSON files
type FileStore struct {
	statePath string
}

// NewFileStore creates a new file-based state store
func NewFileStore() *FileStore {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}
	monitorDir := filepath.Join(home, ".slack-monitor")
	statePath := filepath.Join(monitorDir, "state.json")

	return &FileStore{
		statePath: statePath,
	}
}

// Load loads the persistent state from disk
func (fs *FileStore) Load() (*monitor.State, error) {
	// If state file doesn't exist, create new empty state
	if _, err := os.Stat(fs.statePath); os.IsNotExist(err) {
		log.Println("No existing state file found, creating new state")
		return &monitor.State{
			LastChecked: make(map[string]string),
		}, nil
	}

	data, err := os.ReadFile(fs.statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state monitor.State
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

// Save saves the state to disk atomically
func (fs *FileStore) Save(state *monitor.State) error {
	// Ensure directory exists
	monitorDir := filepath.Dir(fs.statePath)
	if err := os.MkdirAll(monitorDir, 0700); err != nil {
		return fmt.Errorf("failed to create monitor directory: %w", err)
	}

	// Marshal state to JSON
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temporary file first, then rename (atomic)
	tempPath := fs.statePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temporary state file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, fs.statePath); err != nil {
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	return nil
}
