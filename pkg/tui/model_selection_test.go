package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestMessageSelectionWithBoldBorder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80

	// Create a tool message and add it to the model
	toolMsg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeTool,
		Content:   "Tool output",
		Timestamp: time.Now(),
		AgentID:   uuid.New(),
		AgentRole: "runner",
		IsTool:    true,
		Collapsed: true,
	}
	m.messages = []Message{toolMsg}

	// Format without selection (index 0, not selected)
	normalFormat := m.formatMessage(0, toolMsg)

	// Select the message
	m.selectMessage(0)

	// Format with selection (index 0, selected)
	selectedFormat := m.formatMessage(0, toolMsg)

	// Selected format should use double-line borders (╔═╗║╚╝)
	if !strings.Contains(selectedFormat, "╔") {
		t.Error("Selected message should contain double-line top-left border '╔'")
	}
	if !strings.Contains(selectedFormat, "╗") {
		t.Error("Selected message should contain double-line top-right border '╗'")
	}
	if !strings.Contains(selectedFormat, "╚") {
		t.Error("Selected message should contain double-line bottom-left border '╚'")
	}
	if !strings.Contains(selectedFormat, "╝") {
		t.Error("Selected message should contain double-line bottom-right border '╝'")
	}

	// Normal format should use single-line borders (╭─│╮╰╯)
	if !strings.Contains(normalFormat, "╭") {
		t.Error("Normal message should contain single-line top-left border '╭'")
	}
	if !strings.Contains(normalFormat, "╮") {
		t.Error("Normal message should contain single-line top-right border '╮'")
	}
	if !strings.Contains(normalFormat, "╰") {
		t.Error("Normal message should contain single-line bottom-left border '╰'")
	}
	if !strings.Contains(normalFormat, "╯") {
		t.Error("Normal message should contain single-line bottom-right border '╯'")
	}
}

// TestClickSelectsMessage tests that clicking a message selects it.
func TestClickSelectsMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 24
	m.viewport.Height = 20

	// Add messages
	m.messages = []Message{
		{ID: uuid.New(), Type: MessageTypeUser, Content: "Hello", Timestamp: time.Now()},
		{ID: uuid.New(), Type: MessageTypeAgent, Content: "Response", Timestamp: time.Now()},
	}

	content := m.updateViewportContent()
	m.viewport.SetContent(content)
	t.Logf("Message line positions: %v", m.messageLinePositions)

	// Initially no message selected
	if m.selectedMessageIndex != -1 {
		t.Errorf("Expected no message selected initially, got %d", m.selectedMessageIndex)
	}

	// Click on first line (first message)
	clickMsg := tea.MouseMsg{
		Type: tea.MouseLeft,
		Y:    0,
	}

	resultModel, _ := m.handleClickOnToolMessage(clickMsg)
	result := resultModel.(Model)

	// First message should be selected
	if result.selectedMessageIndex != 0 {
		t.Errorf("Expected message 0 selected, got %d", result.selectedMessageIndex)
	}

	// Click on the actual line where second message starts
	secondMsgLine := m.messageLinePositions[1]
	clickMsg2 := tea.MouseMsg{
		Type: tea.MouseLeft,
		Y:    secondMsgLine,
	}
	t.Logf("Clicking on line %d (second message start)", secondMsgLine)

	resultModel2, _ := result.handleClickOnToolMessage(clickMsg2)
	result2 := resultModel2.(Model)

	// Second message should be selected
	if result2.selectedMessageIndex != 1 {
		t.Errorf("Expected message 1 selected, got %d", result2.selectedMessageIndex)
	}
}

// TestDoubleClickThreshold tests that double-click only works within the threshold.
func TestDoubleClickThreshold(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 24
	m.viewport.Height = 20

	// Add messages with a collapsed tool message
	m.messages = []Message{
		{ID: uuid.New(), Type: MessageTypeUser, Content: "Hello", Timestamp: time.Now()},
		{ID: uuid.New(), Type: MessageTypeTool, Content: "Tool output", Timestamp: time.Now(), IsTool: true, Collapsed: true},
	}

	content := m.updateViewportContent()
	m.viewport.SetContent(content)

	t.Logf("Message line positions: %v", m.messageLinePositions)
	t.Logf("Line to message mapping: %v", m.lineToMessage)

	// Click on the actual line where tool message starts
	toolStartLine := m.messageLinePositions[1]
	clickMsg := tea.MouseMsg{
		Type: tea.MouseLeft,
		Y:    toolStartLine,
	}
	t.Logf("Clicking at line %d (tool message start)", toolStartLine)

	resultModel, _ := m.handleClickOnToolMessage(clickMsg)
	result := resultModel.(Model)

	// Message should be selected but still collapsed
	if result.selectedMessageIndex != 1 {
		t.Errorf("Expected message 1 selected, got %d", result.selectedMessageIndex)
	}
	if !result.messages[1].Collapsed {
		t.Error("Message should still be collapsed after single click")
	}

	// Simulate a slow second click (beyond threshold)
	// Set lastClickTime to be more than 500ms ago
	result.lastClickTime = result.lastClickTime.Add(-600 * time.Millisecond)

	resultModel2, _ := result.handleClickOnToolMessage(clickMsg)
	result2 := resultModel2.(Model)

	// Should still be collapsed (not a double-click due to timeout)
	if !result2.messages[1].Collapsed {
		t.Error("Message should still be collapsed after slow second click (not a double-click)")
	}

	// Now test a quick second click (within threshold)
	result2.lastClickTime = result2.lastClickTime.Add(-100 * time.Millisecond) // 100ms ago

	resultModel3, _ := result2.handleClickOnToolMessage(clickMsg)
	result3 := resultModel3.(Model)

	// Should now be expanded (double-click detected)
	if result3.messages[1].Collapsed {
		t.Error("Message should be expanded after quick double-click")
	}
}

// TestDoubleClickDifferentY tests that double-click only works on the same message (not same Y).
func TestDoubleClickDifferentY(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 24
	m.viewport.Height = 20

	// Add messages with collapsed tool messages
	m.messages = []Message{
		{ID: uuid.New(), Type: MessageTypeTool, Content: "Tool 1", Timestamp: time.Now(), IsTool: true, Collapsed: true},
		{ID: uuid.New(), Type: MessageTypeTool, Content: "Tool 2", Timestamp: time.Now(), IsTool: true, Collapsed: true},
	}

	content := m.updateViewportContent()
	m.viewport.SetContent(content)

	// Log the actual line positions for debugging
	t.Logf("lineToMessage: %v", m.lineToMessage)
	t.Logf("messageLinePositions: %v", m.messageLinePositions)

	// First click on first message (line 0)
	clickMsg := tea.MouseMsg{
		Type: tea.MouseLeft,
		Y:    0,
	}

	resultModel, _ := m.handleClickOnToolMessage(clickMsg)
	result := resultModel.(Model)

	// Message 0 should be selected
	if result.selectedMessageIndex != 0 {
		t.Errorf("Expected message 0 selected, got %d", result.selectedMessageIndex)
	}

	// Find a line that belongs to message 1
	msg1StartLine := m.messageLinePositions[1]
	t.Logf("Message 1 starts at line %d", msg1StartLine)

	// Second click on line belonging to message 1 (different message, within threshold time)
	clickMsg2 := tea.MouseMsg{
		Type: tea.MouseLeft,
		Y:    msg1StartLine,
	}

	// Set quick time for double-click detection
	result.lastClickTime = result.lastClickTime.Add(-100 * time.Millisecond)

	resultModel2, _ := result.handleClickOnToolMessage(clickMsg2)
	result2 := resultModel2.(Model)

	// Message 1 should be selected
	if result2.selectedMessageIndex != 1 {
		t.Errorf("Expected message 1 selected, got %d", result2.selectedMessageIndex)
	}

	// Both messages should still be collapsed (different message = not double-click)
	if !result2.messages[0].Collapsed {
		t.Error("Message 0 should still be collapsed (click was on different message)")
	}
	if !result2.messages[1].Collapsed {
		t.Error("Message 1 should still be collapsed (only one click on this message)")
	}
}
