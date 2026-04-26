package tui

import (
	"context"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/shared"

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
	m.messages = []shared.Message{
		{ID: uuid.New(), Type: shared.MessageTypeUserChat, Content: "Hello", Timestamp: time.Now()},
		{ID: uuid.New(), Type: shared.MessageTypeAgentChat, Content: "Hi there", Timestamp: time.Now()},
		{ID: uuid.New(), Type: shared.MessageTypeToolResponse, Content: "Tool output", Timestamp: time.Now(), Metadata: map[string]any{"is_tool": true, "collapsed": true}},
	}

	// Update viewport content
	_ = m.updateViewportContent()

	// Verify message start lines are tracked via lineToMessage
	// First message should start at line 0
	if startLine := m.getMessageStartLine(0); startLine != 0 {
		t.Errorf("First message should start at line 0, got %d", startLine)
	}

	// Verify all messages have valid start lines
	for i := 0; i < 3; i++ {
		startLine := m.getMessageStartLine(i)
		if startLine < 0 {
			t.Errorf("shared.Message %d should have a valid start line", i)
		}
	}

	// Verify start lines are in increasing order
	prevStartLine := -1
	for i := 0; i < 3; i++ {
		startLine := m.getMessageStartLine(i)
		if startLine <= prevStartLine {
			t.Errorf("shared.Message %d start line (%d) should be greater than previous (%d)",
				i, startLine, prevStartLine)
		}
		prevStartLine = startLine
	}

	// Now verify clicking on each message selects the correct one
	m.height = 24
	m.viewport.Height = 20
	m.viewport.SetContent(m.updateViewportContent())

	// Click on first message (at its start line)
	clickOnFirst := tea.MouseMsg{Button: tea.MouseButtonLeft, Y: 0}
	resultModel, _ := m.handleClickOnToolMessage(clickOnFirst)
	result := resultModel.(Model)
	if result.selectedMessageIndex != 0 {
		t.Errorf("Click at Y=0 should select message 0, got %d", result.selectedMessageIndex)
	}

	// Click on second message (at its start line)
	clickY := m.getMessageStartLine(1)
	clickOnSecond := tea.MouseMsg{Button: tea.MouseButtonLeft, Y: clickY}
	resultModel2, _ := result.handleClickOnToolMessage(clickOnSecond)
	result2 := resultModel2.(Model)
	if result2.selectedMessageIndex != 1 {
		t.Errorf("Click at Y=%d (message 1 start) should select message 1, got %d", clickY, result2.selectedMessageIndex)
	}

	// Click on third message (collapsed tool, at its start line)
	clickY2 := m.getMessageStartLine(2)
	clickOnThird := tea.MouseMsg{Button: tea.MouseButtonLeft, Y: clickY2}
	resultModel3, _ := result2.handleClickOnToolMessage(clickOnThird)
	result3 := resultModel3.(Model)
	if result3.selectedMessageIndex != 2 {
		t.Errorf("Click at Y=%d (message 2 start) should select message 2, got %d", clickY2, result3.selectedMessageIndex)
	}
}
