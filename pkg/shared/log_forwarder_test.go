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
	sessionID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
	channelID := uuid.New()
	agentID := uuid.New()
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
		SessionContext: SessionContext{
			SessionID: sessionID,
			ChannelID: channelID,
			AgentID:   agentID,
		},
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

	if entry.AgentID != agentID {
		t.Errorf("expected AgentID %v, got %v", agentID, entry.AgentID)
	}
}

// TestLogEntryWithEmptyFields verifies that LogEntry can be created with
// nil or empty Fields map.
func TestLogEntryWithEmptyFields(t *testing.T) {
	entry := LogEntry{
		Level:       "debug",
		Message:     "Debug message",
		Timestamp:   time.Now(),
		Fields:      nil,
		SessionContext: SessionContext{
			SessionID: uuid.Nil,
			ChannelID: uuid.Nil,
			AgentID:   uuid.Nil,
		},
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

	if entry.SessionID != uuid.Nil {
		t.Errorf("expected nil SessionID, got '%s'", entry.SessionID)
	}

	if entry.ChannelID != uuid.Nil {
		t.Errorf("expected nil ChannelID, got %v", entry.ChannelID)
	}

	if entry.AgentID != uuid.Nil {
		t.Errorf("expected nil AgentID, got %v", entry.AgentID)
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
		SessionContext: SessionContext{
			SessionID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440002"),
			ChannelID: uuid.New(),
			AgentID:   uuid.New(),
		},
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
		sessionID uuid.UUID
		channelID uuid.UUID
		agentID   uuid.UUID
	}{
		{
			name:      "With both routing fields",
			sessionID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440003"),
			channelID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			agentID:   uuid.MustParse("550e8400-e29b-41d4-a716-446655440005"),
		},
		{
			name:      "With nil SessionID",
			sessionID: uuid.Nil,
			channelID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			agentID:   uuid.Nil,
		},
		{
			name:      "With nil ChannelID",
			sessionID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440004"),
			channelID: uuid.Nil,
			agentID:   uuid.Nil,
		},
		{
			name:      "With both nil",
			sessionID: uuid.Nil,
			channelID: uuid.Nil,
			agentID:   uuid.Nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := LogEntry{
				Level:     "info",
				Message:   "Routing test",
				Timestamp: time.Now(),
				Fields:    nil,
				SessionContext: SessionContext{
					SessionID: tc.sessionID,
					ChannelID: tc.channelID,
					AgentID:   tc.agentID,
				},
			}

			if entry.SessionID != tc.sessionID {
				t.Errorf("expected SessionID '%s', got '%s'", tc.sessionID, entry.SessionID)
			}

			if entry.ChannelID != tc.channelID {
				t.Errorf("expected ChannelID %v, got %v", tc.channelID, entry.ChannelID)
			}

			if entry.AgentID != tc.agentID {
				t.Errorf("expected AgentID %v, got %v", tc.agentID, entry.AgentID)
			}
		})
	}
}
