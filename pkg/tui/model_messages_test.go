package tui

import (
	"github.com/m-mizutani/gollem"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/shared"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestAddMessageRingBuffer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80

	// Set a small limit for testing
	m.config.MaxMessages = 5

	// Add 5 messages (at limit)
	for i := 0; i < 5; i++ {
		msg := shared.Message{
			ID:        uuid.New(),
			Role: gollem.RoleAssistant,
			Content:   fmt.Sprintf("shared.Message %d", i),
			Timestamp: time.Now(),
		}
		m.addMessage(msg)
	}

	// Verify we have 5 messages
	if len(m.messages) != 5 {
		t.Fatalf("Expected 5 messages, got %d", len(m.messages))
	}

	// Store the first message ID to verify it gets evicted
	firstMsgID := m.messages[0].ID

	// Add format cache entry for first message
	m.formatCache[firstMsgID] = "cached content"

	// Trigger differential rendering to populate cache
	_ = m.updateViewportContent()
	if m.cachedContent == "" {
		t.Error("Expected cachedContent to be populated after updateViewportContent")
	}
	if m.lastRenderedCount != 5 {
		t.Errorf("Expected lastRenderedCount=5, got %d", m.lastRenderedCount)
	}

	// Add 6th message (exceeds limit by 1)
	newMsg := shared.Message{
		ID:        uuid.New(),
		Role: gollem.RoleAssistant,
		Content:   "shared.Message 5",
		Timestamp: time.Now(),
	}
	m.addMessage(newMsg)

	// Verify we still have 5 messages (oldest evicted)
	if len(m.messages) != 5 {
		t.Errorf("Expected 5 messages after eviction, got %d", len(m.messages))
	}

	// Verify first message was evicted
	if m.messages[0].ID == firstMsgID {
		t.Error("First message should have been evicted")
	}

	// Verify format cache was cleaned up for evicted message
	if _, exists := m.formatCache[firstMsgID]; exists {
		t.Error("Format cache should have been cleaned up for evicted message")
	}

	// Verify differential rendering cache was invalidated
	if m.cachedContent != "" {
		t.Error("Expected cachedContent to be cleared after message eviction")
	}
	if m.lastRenderedCount != 0 {
		t.Errorf("Expected lastRenderedCount=0 after eviction, got %d", m.lastRenderedCount)
	}
}

// TestAddMessageWithZeroLimit verifies that when MaxMessages is 0,
// the default limit of 500 is used as a fallback.
func TestAddMessageWithZeroLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80

	// Set MaxMessages to 0 (should use default of 500)
	m.config.MaxMessages = 0

	// Add messages up to 501 (exceeds default of 500)
	for i := 0; i < 501; i++ {
		msg := shared.Message{
			ID:        uuid.New(),
			Role: gollem.RoleAssistant,
			Content:   fmt.Sprintf("shared.Message %d", i),
			Timestamp: time.Now(),
		}
		m.addMessage(msg)
	}

	// Verify we have 500 messages (default limit applied)
	if len(m.messages) != 500 {
		t.Errorf("Expected 500 messages (default limit), got %d", len(m.messages))
	}

	// Verify first message is "shared.Message 1" (index 1, since 0 was evicted)
	if m.messages[0].Content != "shared.Message 1" {
		t.Errorf("Expected first message to be 'shared.Message 1', got '%s'", m.messages[0].Content)
	}
}

// TestCountLines tests the countLines helper function.
// Note: countLines strips trailing newlines before counting to match
// how lines appear visually in the viewport content.
func TestCountLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"empty string", "", 0},
		{"single line", "hello", 1},
		{"two lines", "hello\nworld", 2},
		{"three lines", "a\nb\nc", 3},
		{"trailing newline", "hello\n", 1},            // trailing \n is stripped
		{"multiple trailing newlines", "a\nb\n\n", 3}, // one trailing \n is stripped, leaving "a\nb\n"
		{"only newlines", "\n\n\n", 3},                // one trailing \n is stripped, leaving "\n\n"
		{"only newlines single", "\n", 0},             // single \n is stripped, leaving ""
		{"only newlines double", "\n\n", 2},           // one trailing \n is stripped, leaving "\n" = 2 lines
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countLines(tt.input)
			if got != tt.expected {
				t.Errorf("countLines(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}
