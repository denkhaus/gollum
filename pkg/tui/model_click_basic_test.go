package tui

import (
	"context"
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestClickOnCollapsedToolMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 50
	m.viewport.Height = 40

	// Create messages where a collapsed tool message is in the middle
	userMsg := channel.Message{
		ID:        uuid.New(),
		Type:      channel.MessageTypeUserChat,
		Content:   "Hello, this is a user message",
		Timestamp: time.Now(),
	}
	toolMsg := channel.Message{
		ID:        uuid.New(),
		Type:      channel.MessageTypeToolResponse,
		Content:   "This is a long tool output that should be collapsed",
		Timestamp: time.Now(),
		Metadata: map[string]any{"is_tool": true, "collapsed": true},
		
	}
	agentMsg := channel.Message{
		ID:        uuid.New(),
		Type:      channel.MessageTypeAgentChat,
		Content:   "Agent response here",
		Timestamp: time.Now(),
	}

	m.messages = []channel.Message{userMsg, toolMsg, agentMsg}
	content := m.updateViewportContent()
	m.viewport.SetContent(content)

	t.Logf("Content:\n%s", content)

	// Get the start and end lines for the tool message
	// The actual message content (without blank gaps)
	toolStartLine := m.getMessageStartLine(1)
	formattedToolMsg := m.formatMessage(1, toolMsg)
	toolMsgLines := countLines(formattedToolMsg)
	toolEndLine := toolStartLine + toolMsgLines - 1 // End of actual tool message content

	t.Logf("Tool message starts at line %d, ends at line %d (content only, no gaps)", toolStartLine, toolEndLine)

	// Click at various positions within the tool message
	for clickY := toolStartLine; clickY <= toolEndLine && clickY < m.viewport.Height; clickY++ {
		clickMsg := tea.MouseMsg{Type: tea.MouseLeft, Y: clickY}
		resultModel, _ := m.handleClickOnToolMessage(clickMsg)
		result := resultModel.(Model)

		if result.selectedMessageIndex != 1 {
			t.Errorf("Click at Y=%d (within tool message lines %d-%d) should select message 1 (tool), got %d",
				clickY, toolStartLine, toolEndLine, result.selectedMessageIndex)
		} else {
			t.Logf("Click at Y=%d correctly selected tool message (index 1)", clickY)
		}

		// Reset for next iteration
		m.selectedMessageIndex = -1
	}
}

// TestClickOnExpandedToolMessage tests clicking on an expanded tool message.
func TestClickOnExpandedToolMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 50
	m.viewport.Height = 40

	// Create messages with an EXPANDED tool message
	userMsg := channel.Message{
		ID:        uuid.New(),
		Type:      channel.MessageTypeUserChat,
		Content:   "Hello",
		Timestamp: time.Now(),
	}
	toolMsg := channel.Message{
		ID:        uuid.New(),
		Type:      channel.MessageTypeToolResponse,
		Content:   "This is tool output that should be visible",
		Timestamp: time.Now(),
		Metadata: map[string]any{"is_tool": true, "collapsed": true},
		 // EXPANDED
	}
	agentMsg := channel.Message{
		ID:        uuid.New(),
		Type:      channel.MessageTypeAgentChat,
		Content:   "Response",
		Timestamp: time.Now(),
	}

	m.messages = []channel.Message{userMsg, toolMsg, agentMsg}
	content := m.updateViewportContent()
	m.viewport.SetContent(content)

	// Click at the start of the tool message
	toolStartLine := m.getMessageStartLine(1)
	clickMsg := tea.MouseMsg{Type: tea.MouseLeft, Y: toolStartLine}
	resultModel, _ := m.handleClickOnToolMessage(clickMsg)
	result := resultModel.(Model)

	if result.selectedMessageIndex != 1 {
		t.Errorf("Click at Y=%d (tool message start) should select message 1, got %d",
			toolStartLine, result.selectedMessageIndex)
	}
}

// TestClickOnToolMessageWithDifferentialUpdate tests clicking after messages are added via differential update.
func TestClickOnToolMessageWithDifferentialUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 50
	m.viewport.Height = 40

	// Start with one message
	m.messages = []channel.Message{
		{ID: uuid.New(), Type: channel.MessageTypeUserChat, Content: "First message", Timestamp: time.Now()},
	}
	m.updateViewportContent()
	m.viewport.SetContent(m.cachedContent)

	t.Logf("After first message: lastRenderedCount=%d", m.lastRenderedCount)

	// Add more messages via differential update
	m.messages = append(m.messages, channel.Message{
		ID:        uuid.New(),
		Type:      channel.MessageTypeToolResponse,
		Content:   "Tool output",
		Timestamp: time.Now(),
		Metadata: map[string]any{"is_tool": true, "collapsed": true},
		
	})
	m.updateViewportContent()
	m.viewport.SetContent(m.cachedContent)

	t.Logf("After tool message: lastRenderedCount=%d", m.lastRenderedCount)

	// Add one more
	m.messages = append(m.messages, channel.Message{
		ID:        uuid.New(),
		Type:      channel.MessageTypeAgentChat,
		Content:   "Agent response",
		Timestamp: time.Now(),
	})
	m.updateViewportContent()
	m.viewport.SetContent(m.cachedContent)

	t.Logf("After agent message: lastRenderedCount=%d", m.lastRenderedCount)

	// Now click on the tool message (index 1)
	toolStartLine := m.getMessageStartLine(1)
	clickMsg := tea.MouseMsg{Type: tea.MouseLeft, Y: toolStartLine}
	resultModel, _ := m.handleClickOnToolMessage(clickMsg)
	result := resultModel.(Model)

	if result.selectedMessageIndex != 1 {
		t.Errorf("Click at Y=%d should select tool message (index 1), got %d",
			toolStartLine, result.selectedMessageIndex)
	}

	// Also verify clicking on the first message
	userStartLine := m.getMessageStartLine(0)
	clickMsg2 := tea.MouseMsg{Type: tea.MouseLeft, Y: userStartLine}
	resultModel2, _ := m.handleClickOnToolMessage(clickMsg2)
	result2 := resultModel2.(Model)

	if result2.selectedMessageIndex != 0 {
		t.Errorf("Click at Y=%d should select user message (index 0), got %d",
			userStartLine, result2.selectedMessageIndex)
	}
}

// TestClickOnToolMessageWithScrolling tests clicking when content is scrolled.
func TestClickOnToolMessageWithScrolling(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 20
	m.viewport.Height = 10 // Small viewport to force scrolling

	// Create many messages so content overflows viewport
	for i := 0; i < 10; i++ {
		msgType := channel.MessageTypeAgentChat
		content := fmt.Sprintf("channel.Message %d with some content to make it longer", i)
		isTool := false

		// Every third message is a tool message
		if i%3 == 2 {
			msgType = channel.MessageTypeToolResponse
			content = fmt.Sprintf("Tool output %d", i)
			isTool = true
		}

		msg := channel.Message{
			ID:        uuid.New(),
			Type:      msgType,
			Content:   content,
			Timestamp: time.Now(),
			Metadata:  make(map[string]any),
		}
		if isTool {
			msg.Metadata["is_tool"] = true
			msg.Metadata["collapsed"] = true
		}
		m.messages = append(m.messages, msg)
	}

	m.updateViewportContent()
	m.viewport.SetContent(m.cachedContent)

	t.Logf("Total content lines: %d", countLines(m.cachedContent))
	t.Logf("Viewport height: %d", m.viewport.Height)

	// Scroll down a bit
	m.viewport.LineDown(10)
	yOffset := m.viewport.YOffset
	t.Logf("Scrolled down, YOffset: %d", yOffset)

	// Click at Y=0 in the viewport (which is actually at content line yOffset)
	clickMsg := tea.MouseMsg{Type: tea.MouseLeft, Y: 0}
	resultModel, _ := m.handleClickOnToolMessage(clickMsg)
	result := resultModel.(Model)

	// The click handler rebuilds content, so use the result's message line positions
	expectedMsgIdx := result.getMessageAtLine(yOffset)
	t.Logf("Click at viewport Y=0 (content line %d) should select message %d", yOffset, expectedMsgIdx)

	if result.selectedMessageIndex != expectedMsgIdx {
		t.Errorf("Click at viewport Y=0 (content line %d) should select message %d, got %d",
			yOffset, expectedMsgIdx, result.selectedMessageIndex)
	}

	// Now click at Y=5 in the viewport
	clickMsg2 := tea.MouseMsg{Type: tea.MouseLeft, Y: 5}
	resultModel2, _ := result.handleClickOnToolMessage(clickMsg2)
	result2 := resultModel2.(Model)

	// The click handler rebuilds content, so use the result's message line positions
	expectedMsgIdx2 := result2.getMessageAtLine(yOffset + 5)
	t.Logf("Click at viewport Y=5 (content line %d) should select message %d", yOffset+5, expectedMsgIdx2)

	if result2.selectedMessageIndex != expectedMsgIdx2 {
		t.Errorf("Click at viewport Y=5 (content line %d) should select message %d, got %d",
			yOffset+5, expectedMsgIdx2, result2.selectedMessageIndex)
	}
}

// TestClickOnFirstToolMessage tests clicking on the first tool message after a user message.
// This is a specific test case to ensure the first tool message's position is calculated correctly.
func TestClickOnFirstToolMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 50
	m.viewport.Height = 40

	// Add a user message
	userMsg := channel.Message{
		ID:        uuid.New(),
		Type:      channel.MessageTypeUserChat,
		Content:   "Hello, please help me",
		Timestamp: time.Now(),
	}
	m.messages = append(m.messages, userMsg)
	content := m.updateViewportContent()
	m.viewport.SetContent(content)

	t.Logf("After user message: lastRenderedCount=%d", m.lastRenderedCount)

	// Now add a tool message (collapsed)
	toolMsg := channel.Message{
		ID:        uuid.New(),
		Type:      channel.MessageTypeToolResponse,
		Content:   "Tool output here",
		Timestamp: time.Now(),
		AgentID:   uuid.New(),
		AgentRole: "runner",
		Metadata: map[string]any{"is_tool": true, "collapsed": true},
		
	}
	m.messages = append(m.messages, toolMsg)
	content = m.updateViewportContent()
	m.viewport.SetContent(content)

	t.Logf("After tool message: lastRenderedCount=%d", m.lastRenderedCount)
	t.Logf("User message lines: %d", m.getMessageStartLine(1))
	t.Logf("Content:\n%s", content)

	// The tool message starts AFTER the user message + 2 blank lines
	// User message (5 lines) + 2 blank lines = tool message starts at line 7
	userLines := countLines(m.formatMessage(0, m.messages[0]))
	expectedToolStart := userLines + 2 // +2 for blank lines
	actualToolStart := m.getMessageStartLine(1)
	t.Logf("Expected tool start: %d, actual: %d", expectedToolStart, actualToolStart)

	if actualToolStart != expectedToolStart {
		t.Errorf("Tool message position incorrect: expected %d, got %d", expectedToolStart, actualToolStart)
	}

	// Now click at the START of the tool message
	toolStartLine := m.getMessageStartLine(1)
	clickMsg := tea.MouseMsg{Type: tea.MouseLeft, Y: toolStartLine}
	resultModel, _ := m.handleClickOnToolMessage(clickMsg)
	result := resultModel.(Model)

	// Should select the TOOL message (index 1), not the user message (index 0)
	if result.selectedMessageIndex != 1 {
		t.Errorf("Click at Y=%d (tool message start) should select tool message (index 1), got %d",
			toolStartLine, result.selectedMessageIndex)
	}

	// Also click at a line within the tool message (not at the start)
	clickMsg2 := tea.MouseMsg{Type: tea.MouseLeft, Y: toolStartLine + 1}
	resultModel2, _ := result.handleClickOnToolMessage(clickMsg2)
	result2 := resultModel2.(Model)

	if result2.selectedMessageIndex != 1 {
		t.Errorf("Click at Y=%d (within tool message) should select tool message (index 1), got %d",
			toolStartLine+1, result2.selectedMessageIndex)
	}
}
