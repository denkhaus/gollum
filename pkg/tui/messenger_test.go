package tui

import (
	"context"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/mocks"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestMessageTypeString(t *testing.T) {
	tests := []struct {
		name     string
		mt       MessageType
		expected string
	}{
		{"User", MessageTypeUser, "user"},
		{"Agent", MessageTypeAgent, "agent"},
		{"Tool", MessageTypeTool, "tool"},
		{"System", MessageTypeSystem, "system"},
		{"Error", MessageTypeError, "error"},
		{"Unknown", MessageType(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mt.String(); got != tt.expected {
				t.Errorf("MessageType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMessageTypeAdapterToMessageType(t *testing.T) {
	tests := []struct {
		name     string
		adapter  MessageTypeAdapter
		expected MessageType
	}{
		{"User", MessageTypeAdapterUser, MessageTypeUser},
		{"Agent", MessageTypeAdapterAgent, MessageTypeAgent},
		{"Tool", MessageTypeAdapterTool, MessageTypeTool},
		{"System", MessageTypeAdapterSystem, MessageTypeSystem},
		{"Error", MessageTypeAdapterError, MessageTypeError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.adapter.ToMessageType(); got != tt.expected {
				t.Errorf("MessageTypeAdapter.ToMessageType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMessageAdapterToMessage(t *testing.T) {
	agentID := uuid.New()
	adapter := MessageAdapter{
		ID:        uuid.New(),
		Type:      MessageTypeAdapterAgent,
		Content:   "test message",
		AgentID:   agentID,
		AgentRole: "test-role",
		IsTool:    false,
	}

	msg := adapter.ToMessage()

	if msg.ID != adapter.ID {
		t.Errorf("ToMessage() ID = %v, want %v", msg.ID, adapter.ID)
	}

	if msg.Type != MessageTypeAgent {
		t.Errorf("ToMessage() Type = %v, want %v", msg.Type, MessageTypeAgent)
	}

	if msg.Content != adapter.Content {
		t.Errorf("ToMessage() Content = %v, want %v", msg.Content, adapter.Content)
	}

	if msg.AgentID != adapter.AgentID {
		t.Errorf("ToMessage() AgentID = %v, want %v", msg.AgentID, adapter.AgentID)
	}

	if msg.AgentRole != adapter.AgentRole {
		t.Errorf("ToMessage() AgentRole = %v, want %v", msg.AgentRole, adapter.AgentRole)
	}

	if msg.IsTool != adapter.IsTool {
		t.Errorf("ToMessage() IsTool = %v, want %v", msg.IsTool, adapter.IsTool)
	}

	// Timestamp should be set to a recent time
	if time.Since(msg.Timestamp) > time.Second {
		t.Errorf("ToMessage() Timestamp should be recent, got %v", msg.Timestamp)
	}
}

func TestSetAndGetMessengerChannel(t *testing.T) {
	// Save original channel to restore later
	original := GetMessengerChannel()
	defer func() {
		if original != nil {
			SetMessengerChannel(original)
		} else {
			// Clear the channel by setting to nil
			globalMessengerMutex.Lock()
			globalMessengerChannel = nil
			globalMessengerMutex.Unlock()
		}
	}()

	// Create a test channel
	ch := make(chan MessageAdapter, 10)
	SetMessengerChannel(ch)

	// Verify we can get the same channel back
	got := GetMessengerChannel()
	if got == nil {
		t.Fatal("GetMessengerChannel() returned nil after SetMessengerChannel")
	}

	// Since MessengerChannel is send-only, we verify it works by using SendMessage
	testMsg := MessageAdapter{
		ID:      uuid.New(),
		Type:    MessageTypeAdapterSystem,
		Content: "test",
	}

	// Send using SendMessage
	go func() {
		SendMessage(testMsg)
	}()

	// Try to receive from the original channel (should work)
	select {
	case msg := <-ch:
		if msg.ID != testMsg.ID {
			t.Errorf("Got wrong message ID")
		}
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

func TestSendMessage(t *testing.T) {
	// Save original channel
	original := GetMessengerChannel()
	defer func() {
		if original != nil {
			SetMessengerChannel(original)
		} else {
			globalMessengerMutex.Lock()
			globalMessengerChannel = nil
			globalMessengerMutex.Unlock()
		}
	}()

	t.Run("with channel set", func(t *testing.T) {
		ch := make(chan MessageAdapter, 10)
		SetMessengerChannel(ch)

		msg := MessageAdapter{
			ID:      uuid.New(),
			Type:    MessageTypeAdapterSystem,
			Content: "test message",
		}

		if !SendMessage(msg) {
			t.Error("SendMessage() returned false when channel is set")
		}

		// Verify message was sent
		select {
		case got := <-ch:
			if got.ID != msg.ID {
				t.Errorf("Got wrong message ID")
			}
		case <-time.After(time.Second):
			t.Fatal("Message not received on channel")
		}
	})

	t.Run("without channel set", func(t *testing.T) {
		// Clear the channel
		globalMessengerMutex.Lock()
		globalMessengerChannel = nil
		globalMessengerMutex.Unlock()

		msg := MessageAdapter{
			ID:      uuid.New(),
			Type:    MessageTypeAdapterSystem,
			Content: "test message",
		}

		if SendMessage(msg) {
			t.Error("SendMessage() returned true when no channel is set")
		}
	})
}

func TestModelSetAndGetMessageChannel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	// Create mock directly to ensure import is used
	_ = mocks.NewMockAgentExecutor(ctrl)
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	// Initially should be nil
	if m.GetMessageChannel() != nil {
		t.Error("NewModel should have nil message channel")
	}

	// Set a channel
	ch := make(chan Message, 10)
	m.SetMessageChannel(ch)

	// Verify we can get it back
	if got := m.GetMessageChannel(); got != ch {
		t.Errorf("GetMessageChannel() = %v, want %v", got, ch)
	}
}

func TestModelFormatAgentName(t *testing.T) {
	testAgentID := uuid.MustParse("56301234-1234-1234-1234-123456789abc")

	tests := []struct {
		name     string
		agentID  uuid.UUID
		role     string
		expected string
	}{
		{
			name:     "with short role",
			agentID:  uuid.New(),
			role:     "assistant",
			expected: "assistant",
		},
		{
			name:     "with long role",
			agentID:  testAgentID,
			role:     "this-is-a-very-long-role-name",
			expected: "5630", // First 4 chars of ID when role > 10 chars
		},
		{
			name:     "with empty role",
			agentID:  uuid.MustParse("12345678-1234-1234-1234-123456789abc"),
			role:     "",
			expected: "1234", // First 4 chars of ID
		},
		{
			name:     "with 10 char role",
			agentID:  uuid.New(),
			role:     "1234567890",
			expected: "1234567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAgentName(tt.agentID, tt.role)
			if got != tt.expected {
				t.Errorf("formatAgentName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestModelUpdateViewportContent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	_ = mocks.NewMockAgentExecutor(ctrl)
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	// Add some messages
	m.messages = []Message{
		{
			ID:        uuid.New(),
			Type:      MessageTypeUser,
			Content:   "user input",
			Timestamp: time.Now(),
		},
		{
			ID:        uuid.New(),
			Type:      MessageTypeAgent,
			Content:   "agent response",
			Timestamp: time.Now(),
		},
	}

	content := m.updateViewportContent()

	// Should contain formatted messages
	if len(content) == 0 {
		t.Error("updateViewportContent() returned empty string")
	}

	// Should contain message content (though exact format depends on formatMessage)
	// We check for key words that should be in the formatted output
	if !containsWord(content, "user input") {
		t.Error("updateViewportContent() should contain user message content")
	}

	if !containsWord(content, "agent response") {
		t.Error("updateViewportContent() should contain agent message content")
	}
}

// Helper function to check if a word is in a string (case-insensitive)
func containsWord(s, word string) bool {
	// Simple check - in production you'd use a more robust method
	return len(s) > 0 && len(word) > 0 && containsSubstring(s, word)
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func TestModelFormatMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	_ = mocks.NewMockAgentExecutor(ctrl)
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	tests := []struct {
		name  string
		msg   Message
		check func(string) bool
	}{
		{
			name: "user message",
			msg: Message{
				ID:        uuid.New(),
				Type:      MessageTypeUser,
				Content:   "test user message",
				Timestamp: time.Now(),
			},
			check: func(s string) bool {
				return len(s) > 0 && (containsWord(s, "You") || containsWord(s, "test user message"))
			},
		},
		{
			name: "agent message",
			msg: Message{
				ID:        uuid.New(),
				Type:      MessageTypeAgent,
				Content:   "test agent message",
				Timestamp: time.Now(),
				AgentID:   uuid.New(),
				AgentRole: "assistant",
			},
			check: func(s string) bool {
				return len(s) > 0 && (containsWord(s, "assistant") || containsWord(s, "test agent message"))
			},
		},
		{
			name: "tool message",
			msg: Message{
				ID:        uuid.New(),
				Type:      MessageTypeTool,
				Content:   "test tool message",
				Timestamp: time.Now(),
				AgentID:   uuid.New(),
				AgentRole: "tool-runner",
				IsTool:    true,
			},
			check: func(s string) bool {
				return len(s) > 0 && (containsWord(s, "Tool") || containsWord(s, "test tool message"))
			},
		},
		{
			name: "system message",
			msg: Message{
				ID:        uuid.New(),
				Type:      MessageTypeSystem,
				Content:   "test system message",
				Timestamp: time.Now(),
			},
			check: func(s string) bool {
				return len(s) > 0 && (containsWord(s, "System") || containsWord(s, "test system message"))
			},
		},
		{
			name: "error message",
			msg: Message{
				ID:        uuid.New(),
				Type:      MessageTypeError,
				Content:   "test error",
				Timestamp: time.Now(),
			},
			check: func(s string) bool {
				return len(s) > 0 && (containsWord(s, "Error") || containsWord(s, "test error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.formatMessage(0, tt.msg) // index 0 = no selection
			if !tt.check(got) {
				t.Errorf("formatMessage() check failed for %s, got: %q", tt.name, got)
			}
		})
	}
}
