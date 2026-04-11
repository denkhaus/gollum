// Package shared provides shared types and interfaces for the Gollum application.
package shared

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestLogEntryInitialization verifies that LogEntry can be properly initialized
// with all routing fields.
func TestLogEntryInitialization(t *testing.T) {
	sessionID := "test-session-123"
	channelID := uuid.New()
	timestamp := time.Now()
	fields := map[string]any{
		"user_id": "user-456",
		"action":  "login",
	}

	entry := LogEntry{
		Level:     "info",
		Message:   "User logged in successfully",
		Timestamp: timestamp,
		Fields:    fields,
		SessionID: sessionID,
		ChannelID: channelID,
	}

	// Verify all fields are set correctly
	if entry.Level != "info" {
		t.Errorf("expected Level 'info', got '%s'", entry.Level)
	}

	if entry.Message != "User logged in successfully" {
		t.Errorf("expected Message 'User logged in successfully', got '%s'", entry.Message)
	}

	if !entry.Timestamp.Equal(timestamp) {
		t.Errorf("expected Timestamp %v, got %v", timestamp, entry.Timestamp)
	}

	if entry.Fields["user_id"] != "user-456" {
		t.Errorf("expected Fields[user_id] 'user-456', got '%v'", entry.Fields["user_id"])
	}

	if entry.Fields["action"] != "login" {
		t.Errorf("expected Fields[action] 'login', got '%v'", entry.Fields["action"])
	}

	if entry.SessionID != sessionID {
		t.Errorf("expected SessionID '%s', got '%s'", sessionID, entry.SessionID)
	}

	if entry.ChannelID != channelID {
		t.Errorf("expected ChannelID %v, got %v", channelID, entry.ChannelID)
	}
}

// TestLogEntryWithEmptyFields verifies that LogEntry can be created with
// nil or empty Fields map.
func TestLogEntryWithEmptyFields(t *testing.T) {
	entry := LogEntry{
		Level:     "debug",
		Message:   "Debug message",
		Timestamp: time.Now(),
		Fields:    nil,
		SessionID: "",
		ChannelID: uuid.Nil,
	}

	// Verify entry is valid even with nil fields
	if entry.Level != "debug" {
		t.Errorf("expected Level 'debug', got '%s'", entry.Level)
	}

	if entry.Message != "Debug message" {
		t.Errorf("expected Message 'Debug message', got '%s'", entry.Message)
	}

	if entry.Fields != nil {
		t.Errorf("expected Fields to be nil, got %v", entry.Fields)
	}

	if entry.SessionID != "" {
		t.Errorf("expected empty SessionID, got '%s'", entry.SessionID)
	}

	if entry.ChannelID != uuid.Nil {
		t.Errorf("expected nil ChannelID, got %v", entry.ChannelID)
	}
}

// mockLogForwarder is a mock implementation of LogForwarder for testing.
type mockLogForwarder struct {
	receivedEntries []LogEntry
}

// ForwardLog stores the entry for later verification.
func (m *mockLogForwarder) ForwardLog(entry LogEntry) {
	m.receivedEntries = append(m.receivedEntries, entry)
}

// TestLogForwarderInterface verifies that mockLogForwarder implements
// the LogForwarder interface correctly.
func TestLogForwarderInterface(t *testing.T) {
	mock := &mockLogForwarder{
		receivedEntries: []LogEntry{},
	}

	entry := LogEntry{
		Level:     "error",
		Message:   "Test error",
		Timestamp: time.Now(),
		Fields:    map[string]any{"code": 500},
		SessionID: "session-789",
		ChannelID: uuid.New(),
	}

	// This call should compile and execute without errors
	mock.ForwardLog(entry)

	// Verify the entry was received
	if len(mock.receivedEntries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(mock.receivedEntries))
	}

	received := mock.receivedEntries[0]
	if received.Level != entry.Level {
		t.Errorf("expected Level '%s', got '%s'", entry.Level, received.Level)
	}

	if received.Message != entry.Message {
		t.Errorf("expected Message '%s', got '%s'", entry.Message, received.Message)
	}
}

// TestLogEntryRoutingFields verifies that LogEntry properly stores
// routing information (SessionID and ChannelID) for log forwarding.
func TestLogEntryRoutingFields(t *testing.T) {
	testCases := []struct {
		name      string
		sessionID string
		channelID uuid.UUID
	}{
		{
			name:      "With both routing fields",
			sessionID: "session-abc",
			channelID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		},
		{
			name:      "With empty SessionID",
			sessionID: "",
			channelID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		},
		{
			name:      "With nil ChannelID",
			sessionID: "session-xyz",
			channelID: uuid.Nil,
		},
		{
			name:      "With both empty",
			sessionID: "",
			channelID: uuid.Nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := LogEntry{
				Level:     "info",
				Message:   "Routing test",
				Timestamp: time.Now(),
				Fields:    nil,
				SessionID: tc.sessionID,
				ChannelID: tc.channelID,
			}

			if entry.SessionID != tc.sessionID {
				t.Errorf("expected SessionID '%s', got '%s'", tc.sessionID, entry.SessionID)
			}

			if entry.ChannelID != tc.channelID {
				t.Errorf("expected ChannelID %v, got %v", tc.channelID, entry.ChannelID)
			}
		})
	}
}
