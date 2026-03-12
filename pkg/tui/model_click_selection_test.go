package tui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// Helper function to get collapsed state from message metadata
func getCollapsed(msg channel.Message) bool {
	if msg.Metadata == nil {
		return false
	}
	if collapsed, ok := msg.Metadata["collapsed"].(bool); ok {
		return collapsed
	}
	return false
}

func TestMultipleDoubleClicks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 50
	m.viewport.Height = 40

	// Create messages with collapsed tool messages
	m.messages = []channel.Message{
		{ID: uuid.New(), Type: channel.MessageTypeUserChat, Content: "Hello", Timestamp: time.Now()},
		{ID: uuid.New(), Type: channel.MessageTypeToolResponse, Content: "Tool 1", Timestamp: time.Now(), Metadata: map[string]any{"is_tool": true, "collapsed": true}},
		{ID: uuid.New(), Type: channel.MessageTypeToolResponse, Content: "Tool 2", Timestamp: time.Now(), Metadata: map[string]any{"is_tool": true, "collapsed": true}},
	}
	m.updateViewportContent()
	m.viewport.SetContent(m.cachedContent)

	// Simulate multiple double-clicks on the first tool message
	toolStartLine := m.getMessageStartLine(1)

	for i := 0; i < 5; i++ {
		// Reset click tracking to ensure clean state for each double-click
		// This simulates waiting >500ms between double-click attempts
		m.lastClickTime = time.Time{}
		m.lastClickedMessageIndex = -1

		t.Logf("Iteration %d start: tool message Collapsed=%v", i, getCollapsed(m.messages[1]))

		// First click
		click1 := tea.MouseMsg{Type: tea.MouseLeft, Y: toolStartLine}
		resultModel1, _ := m.handleClickOnToolMessage(click1)
		m = resultModel1.(Model)

		t.Logf("Iteration %d after first click: Collapsed=%v, lastClickedMessageIndex=%d", i, getCollapsed(m.messages[1]), m.lastClickedMessageIndex)

		// Simulate quick second click (within threshold)
		m.lastClickTime = m.lastClickTime.Add(-100 * time.Millisecond)

		// Second click (should be double-click)
		click2 := tea.MouseMsg{Type: tea.MouseLeft, Y: toolStartLine}
		resultModel2, _ := m.handleClickOnToolMessage(click2)
		m = resultModel2.(Model)

		t.Logf("Iteration %d after second click: Collapsed=%v", i, getCollapsed(m.messages[1]))

		// Check that collapse state toggled
		// Start: Collapsed=true
		// After iteration 0: Collapsed=false (expanded)
		// After iteration 1: Collapsed=true (collapsed)
		// After iteration 2: Collapsed=false
		// etc.
		var expectedCollapsed bool
		switch i {
		case 0:
			expectedCollapsed = false // First toggle: true -> false
		case 1:
			expectedCollapsed = true // Second toggle: false -> true
		case 2:
			expectedCollapsed = false // Third toggle: true -> false
		case 3:
			expectedCollapsed = true
		case 4:
			expectedCollapsed = false
		default:
			expectedCollapsed = (i % 2) == 0 // After even iterations: collapsed, After odd: expanded
		}

		if getCollapsed(m.messages[1]) != expectedCollapsed {
			t.Errorf("Iteration %d: expected Collapsed=%v, got %v", i, expectedCollapsed, getCollapsed(m.messages[1]))
		}

		// Update toolStartLine for next iteration (line positions may have changed)
		toolStartLine = m.getMessageStartLine(1)
	}
}

// TestDoubleClickLinePositionChange tests that double-clicking still works when line positions change.
func TestDoubleClickLinePositionChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 50
	m.viewport.Height = 40

	// Create messages with collapsed tool messages
	m.messages = []channel.Message{
		{ID: uuid.New(), Type: channel.MessageTypeUserChat, Content: "Hello", Timestamp: time.Now()},
		{ID: uuid.New(), Type: channel.MessageTypeToolResponse, Content: "Tool 1", Timestamp: time.Now(), Metadata: map[string]any{"is_tool": true, "collapsed": true}},
		{ID: uuid.New(), Type: channel.MessageTypeAgentChat, Content: "Agent response", Timestamp: time.Now()},
	}
	m.updateViewportContent()
	m.viewport.SetContent(m.cachedContent)

	// Get initial positions
	initialMsg2Start := m.getMessageStartLine(2)
	t.Logf("Initial message 2 start line: %d", initialMsg2Start)

	// Double-click to expand tool message
	toolStartLine := m.getMessageStartLine(1)
	click1 := tea.MouseMsg{Type: tea.MouseLeft, Y: toolStartLine}
	resultModel1, _ := m.handleClickOnToolMessage(click1)
	m = resultModel1.(Model)

	// Quick second click
	m.lastClickTime = m.lastClickTime.Add(-100 * time.Millisecond)
	click2 := tea.MouseMsg{Type: tea.MouseLeft, Y: toolStartLine}
	resultModel2, _ := m.handleClickOnToolMessage(click2)
	m = resultModel2.(Model)

	t.Logf("After expand: message 2 start=%d, Collapsed=%v", m.getMessageStartLine(2), getCollapsed(m.messages[1]))

	// Tool message should be expanded now
	if getCollapsed(m.messages[1]) {
		t.Error("Tool message should be expanded after double-click")
	}

	// The agent message should now start at a higher line number
	// (because the expanded tool message takes more space)
	newMsg2Start := m.getMessageStartLine(2)
	if newMsg2Start <= initialMsg2Start {
		t.Errorf("Agent message should start at higher line after tool expansion, was %d, now %d",
			initialMsg2Start, newMsg2Start)
	}
}

// TestHandleClickOnToolMessage tests mouse click handling for collapsing/expanding tool messages.
func TestHandleClickOnToolMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 24
	m.viewport.Height = 20 // Set viewport height

	// Add messages with known positions
	m.messages = []channel.Message{
		{ID: uuid.New(), Type: channel.MessageTypeUserChat, Content: "Hello", Timestamp: time.Now()},
		{ID: uuid.New(), Type: channel.MessageTypeToolResponse, Content: "Tool output", Timestamp: time.Now(), Metadata: map[string]any{"is_tool": true, "collapsed": true}},
		{ID: uuid.New(), Type: channel.MessageTypeAgentChat, Content: "Response", Timestamp: time.Now()},
	}

	// Set up the viewport content
	content := m.updateViewportContent()
	m.viewport.SetContent(content)

	t.Logf("Line to message mapping: %v", m.lineToMessage)

	// Click on the actual line where tool message starts
	toolStartLine := m.getMessageStartLine(1)
	clickMsg := tea.MouseMsg{
		Type: tea.MouseLeft,
		Y:    toolStartLine, // Click on the tool message
	}
	t.Logf("Clicking at line %d (tool message start)", toolStartLine)

	resultModel, _ := m.handleClickOnToolMessage(clickMsg)
	result := resultModel.(Model)

	// Verify the message is selected (single click behavior)
	if result.selectedMessageIndex != 1 {
		t.Errorf("channel.Message at index 1 should be selected, got %d", result.selectedMessageIndex)
	}

	// Single click should NOT toggle collapse (still collapsed)
	if !getCollapsed(result.messages[1]) {
		t.Error("Tool message should still be collapsed after single click (double-click required to toggle)")
	}

	// Now simulate a double-click by clicking again quickly on the same message
	// The lastClickTime and lastClickedMessageIndex were already set by the first click
	result.lastClickTime = result.lastClickTime.Add(-100 * time.Millisecond) // Simulate quick second click

	resultModel2, _ := result.handleClickOnToolMessage(clickMsg)
	result2 := resultModel2.(Model)

	// After double-click, the tool message should be expanded
	if getCollapsed(result2.messages[1]) {
		t.Error("Tool message should be expanded after double-click")
	}

	// Selection should still be on message 1
	if result2.selectedMessageIndex != 1 {
		t.Errorf("channel.Message at index 1 should still be selected, got %d", result2.selectedMessageIndex)
	}
}

// TestHandleClickOnNonToolMessage tests that clicking on non-tool messages selects them but doesn't toggle collapse.
func TestHandleClickOnNonToolMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 24
	m.viewport.Height = 20

	// Add messages
	m.messages = []channel.Message{
		{ID: uuid.New(), Type: channel.MessageTypeUserChat, Content: "Hello", Timestamp: time.Now()},
		{ID: uuid.New(), Type: channel.MessageTypeAgentChat, Content: "Response", Timestamp: time.Now()},
	}

	content := m.updateViewportContent()
	m.viewport.SetContent(content)
	// lineToMessage is built by updateViewportContent()

	// Click on line 0 (user message)
	clickMsg := tea.MouseMsg{
		Type: tea.MouseLeft,
		Y:    0,
	}

	resultModel, _ := m.handleClickOnToolMessage(clickMsg)
	result := resultModel.(Model)

	// User message should be selected
	if result.selectedMessageIndex != 0 {
		t.Errorf("channel.Message at index 0 should be selected, got %d", result.selectedMessageIndex)
	}

	// Messages should be unchanged (no collapse toggle for non-tool messages)
	if len(result.messages) != 2 {
		t.Error("Messages should be unchanged after clicking non-tool message")
	}
}

// TestHandleClickOutsideViewport tests that clicks outside viewport bounds are handled.
func TestHandleClickOutsideViewport(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 24
	m.viewport.Height = 10

	m.messages = []channel.Message{
		{ID: uuid.New(), Type: channel.MessageTypeToolResponse, Content: "Tool", Timestamp: time.Now(), Metadata: map[string]any{"is_tool": true, "collapsed": true}},
	}

	content := m.updateViewportContent()
	m.viewport.SetContent(content)
	// lineToMessage is built by updateViewportContent()

	// Click outside viewport bounds (Y = 15, but viewport height is 10)
	clickMsg := tea.MouseMsg{
		Type: tea.MouseLeft,
		Y:    15, // Outside viewport
	}

	resultModel, _ := m.handleClickOnToolMessage(clickMsg)
	result := resultModel.(Model)

	// Tool message should still be collapsed (click was outside viewport)
	if !getCollapsed(result.messages[0]) {
		t.Error("Tool message should remain collapsed when click is outside viewport")
	}
}

// TestHandleClickOnInvalidLine tests clicking on a line with no message.
func TestHandleClickOnInvalidLine(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 24
	m.viewport.Height = 20

	// No messages set (empty lineToMessage)

	// Click somewhere
	clickMsg := tea.MouseMsg{
		Type: tea.MouseLeft,
		Y:    5,
	}

	resultModel, _ := m.handleClickOnToolMessage(clickMsg)
	// Should not panic and should pass to text input
	_ = resultModel.(Model)
}
