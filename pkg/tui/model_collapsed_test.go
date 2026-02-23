package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestFormatCollapsedToolMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80

	toolMsg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeTool,
		Content:   "This is a very long tool output that should be hidden when collapsed. It contains multiple lines of output that would clutter the conversation view.",
		Timestamp: time.Now(),
		AgentID:   uuid.New(),
		AgentRole: "runner",
		IsTool:    true,
		Collapsed: true,
	}

	got := m.formatMessage(0, toolMsg)

	// Verify collapsed indicator is present
	if !strings.Contains(got, "Click to expand") {
		t.Error("Collapsed tool message should contain 'Click to expand'")
	}

	// Verify it shows the "Tool" label
	if !strings.Contains(got, "Tool") {
		t.Error("Collapsed tool message should contain 'Tool'")
	}

	// Verify it's compact (fewer lines than expanded)
	lines := strings.Count(got, "\n")
	if lines > 5 {
		t.Errorf("Collapsed message should be compact (max 5 lines), got %d lines", lines)
	}

	// Verify the full content is NOT shown
	if strings.Contains(got, "very long tool output") {
		t.Error("Collapsed message should NOT contain the full tool output")
	}
}

// TestFormatExpandedToolMessage tests that expanded tool messages show full content.
func TestFormatExpandedToolMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80

	toolMsg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeTool,
		Content:   "This is the tool output content that should be visible.",
		Timestamp: time.Now(),
		AgentID:   uuid.New(),
		AgentRole: "runner",
		IsTool:    true,
		Collapsed: false, // Expanded state
	}

	got := m.formatMessage(0, toolMsg)

	// Verify the content IS shown
	if !strings.Contains(got, "tool output content") {
		t.Error("Expanded tool message should contain the full content")
	}

	// Verify collapsed indicator is NOT present
	if strings.Contains(got, "Click to expand") {
		t.Error("Expanded tool message should NOT contain 'Click to expand'")
	}
}

// TestGetMessageAtLine tests finding messages by line position.
func TestGetMessageAtLine(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80

	// Add some messages
	m.messages = []Message{
		{ID: uuid.New(), Type: MessageTypeUser, Content: "Hello", Timestamp: time.Now()},
		{ID: uuid.New(), Type: MessageTypeAgent, Content: "Hi there, how can I help?", Timestamp: time.Now()},
		{ID: uuid.New(), Type: MessageTypeTool, Content: "Tool output", Timestamp: time.Now(), IsTool: true},
	}

	// Build the line-to-message mapping by updating viewport content
	m.updateViewportContent()

	// Verify the mapping was built
	if len(m.lineToMessage) == 0 {
		t.Fatal("lineToMessage should be populated after updateViewportContent()")
	}

	// Test that lines within each message return the correct message index
	// Use getMessageStartLine() to find where each message starts
	// Message 0 starts at line 0, Message 1 starts at getMessageStartLine(1), etc.

	tests := []struct {
		name     string
		line     int
		expected int
	}{
		{"first message line 0", 0, 0},
		{"before first", -1, -1},
	}

	// Test first message (line 0)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.getMessageAtLine(tt.line)
			if got != tt.expected {
				t.Errorf("getMessageAtLine(%d) = %d, want %d", tt.line, got, tt.expected)
			}
		})
	}

	// Test lines within bounds
	for line := 0; line < len(m.lineToMessage); line++ {
		expected := m.lineToMessage[line]
		got := m.getMessageAtLine(line)
		if got != expected {
			t.Errorf("getMessageAtLine(%d) = %d, want %d", line, got, expected)
		}
	}

	// Test line beyond bounds
	if got := m.getMessageAtLine(len(m.lineToMessage) + 100); got != -1 {
		t.Errorf("getMessageAtLine(beyond bounds) should return -1, got %d", got)
	}
}

// TestToggleMessageCollapse tests toggling collapse state.
func TestToggleMessageCollapse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80

	// Add messages
	m.messages = []Message{
		{ID: uuid.New(), Type: MessageTypeUser, Content: "Hello", Timestamp: time.Now()},
		{ID: uuid.New(), Type: MessageTypeTool, Content: "Tool output", Timestamp: time.Now(), IsTool: true, Collapsed: true},
		{ID: uuid.New(), Type: MessageTypeAgent, Content: "Response", Timestamp: time.Now()},
	}

	// Test toggling tool message (should succeed)
	if !m.toggleMessageCollapse(1) {
		t.Error("toggleMessageCollapse(1) should return true for tool message")
	}
	if m.messages[1].Collapsed {
		t.Error("Tool message should be expanded after toggle")
	}

	// Toggle again (should collapse)
	if !m.toggleMessageCollapse(1) {
		t.Error("toggleMessageCollapse(1) should return true for expanded tool message")
	}
	if !m.messages[1].Collapsed {
		t.Error("Tool message should be collapsed after second toggle")
	}

	// Test toggling non-tool message (should fail)
	if m.toggleMessageCollapse(0) {
		t.Error("toggleMessageCollapse(0) should return false for user message")
	}

	// Test invalid index
	if m.toggleMessageCollapse(-1) {
		t.Error("toggleMessageCollapse(-1) should return false")
	}
	if m.toggleMessageCollapse(100) {
		t.Error("toggleMessageCollapse(100) should return false for out of bounds")
	}
}

// TestToggleMessageCollapseInvalidatesCache tests that toggling collapse invalidates the format cache.
func TestToggleMessageCollapseInvalidatesCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80

	// Add a tool message
	toolMsg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeTool,
		Content:   "Tool output",
		Timestamp: time.Now(),
		IsTool:    true,
		Collapsed: true,
	}
	m.messages = []Message{toolMsg}

	// Build the cache
	content := m.updateViewportContent()
	m.viewport.SetContent(content)

	// Verify cache is populated
	if len(m.formatCache) == 0 {
		t.Error("Format cache should be populated after updateViewportContent")
	}

	// Toggle collapse
	m.toggleMessageCollapse(0)

	// Verify cache for that message was invalidated
	if _, exists := m.formatCache[toolMsg.ID]; exists {
		t.Error("Format cache for tool message should be invalidated after toggle")
	}

	// Verify cached content was cleared
	if m.cachedContent != "" {
		t.Error("Cached content should be cleared after toggle to force rebuild")
	}
}

// TestMessageAdapterToMessageCollapsedState tests that MessageAdapter.ToMessage sets collapsed state for tool messages.
func TestMessageAdapterToMessageCollapsedState(t *testing.T) {
	tests := []struct {
		name            string
		adapter         MessageAdapter
		expectCollapsed bool
	}{
		{
			name: "tool message with IsTool=true",
			adapter: MessageAdapter{
				ID:      uuid.New(),
				Type:    MessageTypeAdapterAgent,
				Content: "output",
				IsTool:  true,
			},
			expectCollapsed: true,
		},
		{
			name: "message with MessageTypeAdapterTool type",
			adapter: MessageAdapter{
				ID:      uuid.New(),
				Type:    MessageTypeAdapterTool,
				Content: "output",
				IsTool:  false,
			},
			expectCollapsed: true,
		},
		{
			name: "regular agent message",
			adapter: MessageAdapter{
				ID:      uuid.New(),
				Type:    MessageTypeAdapterAgent,
				Content: "response",
				IsTool:  false,
			},
			expectCollapsed: false,
		},
		{
			name: "user message",
			adapter: MessageAdapter{
				ID:      uuid.New(),
				Type:    MessageTypeAdapterUser,
				Content: "question",
				IsTool:  false,
			},
			expectCollapsed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.adapter.ToMessage()
			if got.Collapsed != tt.expectCollapsed {
				t.Errorf("ToMessage().Collapsed = %v, want %v", got.Collapsed, tt.expectCollapsed)
			}
		})
	}
}
