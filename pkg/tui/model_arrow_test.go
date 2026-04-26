package tui

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/shared"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestArrowKeyScrolling(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.activeViewport = ViewportMain // Set to main viewport for scrolling

	// Add some messages to scroll through
	for i := 0; i < 20; i++ {
		m.messages = append(m.messages, shared.Message{
			ID:        uuid.New(),
			Type:      shared.MessageTypeAgentChat,
			Content:   fmt.Sprintf("shared.Message %d", i),
			Timestamp: time.Now(),
		})
	}
	m.viewport.SetContent(m.updateViewportContent())
	m.viewport.GotoBottom()
	initialYOffset := m.viewport.YOffset

	// Press arrow up - should scroll immediately
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = model.(Model)

	// YOffset should change immediately
	if m.viewport.YOffset >= initialYOffset {
		t.Errorf("YOffset should decrease after up arrow, was %d, now %d", initialYOffset, m.viewport.YOffset)
	}
}

// TestArrowKeyHistoryNavigation verifies that arrow keys navigate history when in input mode.
func TestArrowKeyHistoryNavigation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.activeViewport = ViewportInput // Input focus for history navigation
	m.inputHistory = []string{"cmd1", "cmd2", "cmd3"}
	m.inputHistoryIndex = len(m.inputHistory) // Start at "newest" position

	// Press up - should navigate history immediately
	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = model.(Model)

	// Input should be set immediately
	if m.textInput.Value() != "cmd3" {
		t.Errorf("Expected input to be 'cmd3', got '%s'", m.textInput.Value())
	}

	// No command should be scheduled
	if cmd != nil {
		t.Error("No command should be scheduled for history navigation")
	}
}

// TestArrowKeyDifferentViewports verifies scrolling works for both viewports.
func TestArrowKeyDifferentViewports(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80

	// Add log entries
	for i := 0; i < 20; i++ {
		m.logEntries = append(m.logEntries, fmt.Sprintf("Log entry %d", i))
	}
	m.logViewport.SetContent(strings.Join(m.logEntries, "\n"))
	m.logViewport.GotoBottom()
	initialLogYOffset := m.logViewport.YOffset

	// Test log viewport scrolling
	m.activeViewport = ViewportLogs
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = model.(Model)

	// Log viewport should scroll immediately
	if m.logViewport.YOffset >= initialLogYOffset {
		t.Errorf("Log viewport should scroll up, was %d, now %d", initialLogYOffset, m.logViewport.YOffset)
	}
}
