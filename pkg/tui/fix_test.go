package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/shared"

	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"go.uber.org/mock/gomock"
)

// TestFixFirstUserMessageHasTopBorder verifies the fix for the bug where
// the first user message had no top border.
func TestFixFirstUserMessageHasTopBorder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := NewMockAgentExecutor(ctrl)
	agent.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(&gollem.ExecuteResponse{
		Texts: []string{"test response"},
	}, nil).AnyTimes()

	m := NewModel(ctx, agent)

	// Set realistic terminal size (like a typical user terminal)
	m.width = 80
	m.height = 24

	// Calculate viewport height exactly as the real code does
	reservedLines := 4 // status bar + prompt + footer
	if m.config.StatusEnabled {
		reservedLines++ // Extra line for status bar
	}
	logPanelHeight := 6
	reservedLines += logPanelHeight + 1 // +1 for log separator

	viewportHeight := m.height - reservedLines
	m.viewport.Width = m.width
	m.viewport.Height = viewportHeight

	// Simulate user typing "hi" and submitting (like in the bug report)
	userMsg := shared.Message{
		ID:        uuid.New(),
		Role: gollem.RoleUser,
		Content:   "hi",
		Timestamp: time.Now(),
	}
	m.messages = append(m.messages, userMsg)
	m.viewport.SetContent(m.updateViewportContent())
	m.viewport.GotoTop() // The fix!

	// Simulate agent responding
	agentMsg := shared.Message{
		ID:        uuid.New(),
		Role: gollem.RoleAssistant,
		Content:   "Hello! How can I help you today? I'm here to assist with a variety of tasks including: Searching and storing information in my knowledge base, Researching topics on the web, Managing files and code, Running commands and processes, Working with agents for specialized tasks. What would you like help with?",
		Timestamp: time.Now(),
	}
	m.messages = append(m.messages, agentMsg)
	m.viewport.SetContent(m.updateViewportContent())
	m.viewport.GotoTop() // The fix!

	// Get the viewport view
	view := m.viewport.View()

	// Verify the fix: both messages should be visible with headers
	if !contains(view, "You") {
		t.Error("FAILED: User message header is NOT visible (bug still present)")
		t.Logf("Viewport view:\n%s", view)
	} else {
		t.Log("SUCCESS: User message header IS visible")
	}

	if !contains(view, "Agent:") {
		t.Error("FAILED: Agent message is NOT visible")
		t.Logf("Viewport view:\n%s", view)
		t.Logf("Viewport height: %d, width: %d", m.viewport.Height, m.viewport.Width)
	} else {
		t.Log("SUCCESS: Agent message IS visible")
	}

	// Verify the top border character is present
	if !strings.Contains(view, "╭") {
		t.Error("FAILED: Top border character '╭' is NOT present")
	} else {
		t.Log("SUCCESS: Top border character '╭' IS present")
	}
}

// TestFixPreservesScrolling verifies that users can still scroll down
// to see more content when it exceeds viewport height.
func TestFixPreservesScrolling(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := NewMockAgentExecutor(ctrl)
	agent.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(&gollem.ExecuteResponse{
		Texts: []string{"test response"},
	}, nil).AnyTimes()

	m := NewModel(ctx, agent)

	m.width = 80
	m.height = 24

	reservedLines := 4
	if m.config.StatusEnabled {
		reservedLines++
	}
	logPanelHeight := 6
	reservedLines += logPanelHeight + 1

	viewportHeight := m.height - reservedLines
	m.viewport.Width = m.width
	m.viewport.Height = viewportHeight

	// Add a long conversation that exceeds viewport height
	for i := 0; i < 5; i++ {
		userMsg := shared.Message{
			ID:        uuid.New(),
			Role: gollem.RoleUser,
			Content:   "Test message",
			Timestamp: time.Now(),
		}
		m.messages = append(m.messages, userMsg)

		agentMsg := shared.Message{
			ID:        uuid.New(),
			Role: gollem.RoleAssistant,
			Content:   strings.Repeat("This is a long response. ", 10),
			Timestamp: time.Now(),
		}
		m.messages = append(m.messages, agentMsg)
	}

	m.viewport.SetContent(m.updateViewportContent())
	m.viewport.GotoTop()

	// Verify we can see the first message
	view := m.viewport.View()
	if !contains(view, "You") {
		t.Error("FAILED: Cannot see first message after GotoTop")
	} else {
		t.Log("SUCCESS: Can see first message after GotoTop")
	}

	// Verify we can scroll down
	m.viewport.ScrollDown(10)
	viewAfterScroll := m.viewport.View()
	if len(viewAfterScroll) == 0 || viewAfterScroll == view {
		t.Error("FAILED: Scrolling down doesn't change the view")
	} else {
		t.Log("SUCCESS: Scrolling down changes the view")
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

// findSubstring finds a substring in a string.
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
