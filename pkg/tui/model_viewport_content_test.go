package tui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestUpdateViewportContentTracksMessagePositions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80

	// Add messages
	m.messages = []Message{
		{ID: uuid.New(), Type: MessageTypeUser, Content: "Hello", Timestamp: time.Now()},
		{ID: uuid.New(), Type: MessageTypeAgent, Content: "Hi there", Timestamp: time.Now()},
		{ID: uuid.New(), Type: MessageTypeTool, Content: "Tool output", Timestamp: time.Now(), IsTool: true, Collapsed: true},
	}

	// Update viewport content
	_ = m.updateViewportContent()

	// Verify message line positions are tracked
	if len(m.messageLinePositions) != 3 {
		t.Errorf("Expected 3 message line positions, got %d", len(m.messageLinePositions))
	}

	// First message should start at line 0
	if m.messageLinePositions[0] != 0 {
		t.Errorf("First message should start at line 0, got %d", m.messageLinePositions[0])
	}

	// Subsequent messages should have increasing line positions
	for i := 1; i < len(m.messageLinePositions); i++ {
		if m.messageLinePositions[i] <= m.messageLinePositions[i-1] {
			t.Errorf("Message %d line position (%d) should be greater than message %d (%d)",
				i, m.messageLinePositions[i], i-1, m.messageLinePositions[i-1])
		}
	}

	// Now verify clicking on each message selects the correct one
	m.height = 24
	m.viewport.Height = 20
	m.viewport.SetContent(m.updateViewportContent())

	// Click on first message (at its start line)
	clickOnFirst := tea.MouseMsg{Type: tea.MouseLeft, Y: 0}
	resultModel, _ := m.handleClickOnToolMessage(clickOnFirst)
	result := resultModel.(Model)
	if result.selectedMessageIndex != 0 {
		t.Errorf("Click at Y=0 should select message 0, got %d", result.selectedMessageIndex)
	}

	// Click on second message (at its start line)
	clickY := m.messageLinePositions[1]
	clickOnSecond := tea.MouseMsg{Type: tea.MouseLeft, Y: clickY}
	resultModel2, _ := result.handleClickOnToolMessage(clickOnSecond)
	result2 := resultModel2.(Model)
	if result2.selectedMessageIndex != 1 {
		t.Errorf("Click at Y=%d (message 1 start) should select message 1, got %d", clickY, result2.selectedMessageIndex)
	}

	// Click on third message (collapsed tool, at its start line)
	clickY2 := m.messageLinePositions[2]
	clickOnThird := tea.MouseMsg{Type: tea.MouseLeft, Y: clickY2}
	resultModel3, _ := result2.handleClickOnToolMessage(clickOnThird)
	result3 := resultModel3.(Model)
	if result3.selectedMessageIndex != 2 {
		t.Errorf("Click at Y=%d (message 2 start) should select message 2, got %d", clickY2, result3.selectedMessageIndex)
	}
}
