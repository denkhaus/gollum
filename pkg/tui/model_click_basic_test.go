package tui

import (
	"context"
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
	userMsg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeUser,
		Content:   "Hello, this is a user message",
		Timestamp: time.Now(),
	}
	toolMsg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeTool,
		Content:   "This is a long tool output that should be collapsed",
		Timestamp: time.Now(),
		IsTool:    true,
		Collapsed: true,
	}
	agentMsg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeAgent,
		Content:   "Agent response here",
		Timestamp: time.Now(),
	}

	m.messages = []Message{userMsg, toolMsg, agentMsg}
	content := m.updateViewportContent()
	m.viewport.SetContent(content)

	t.Logf("Message line positions: %v", m.messageLinePositions)
	t.Logf("Content:\n%s", content)

	// Get the start and end lines for the tool message
	// The actual message content (without blank gaps)
	toolStartLine := m.messageLinePositions[1]
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
	userMsg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeUser,
		Content:   "Hello",
		Timestamp: time.Now(),
	}
	toolMsg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeTool,
		Content:   "This is tool output that should be visible",
		Timestamp: time.Now(),
		IsTool:    true,
		Collapsed: false, // EXPANDED
	}
	agentMsg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeAgent,
		Content:   "Response",
		Timestamp: time.Now(),
	}

	m.messages = []Message{userMsg, toolMsg, agentMsg}
	content := m.updateViewportContent()
	m.viewport.SetContent(content)

	t.Logf("Message line positions: %v", m.messageLinePositions)

	// Click at the start of the tool message
	toolStartLine := m.messageLinePositions[1]
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
	m.messages = []Message{
		{ID: uuid.New(), Type: MessageTypeUser, Content: "First message", Timestamp: time.Now()},
	}
	m.updateViewportContent()
	m.viewport.SetContent(m.cachedContent)

	t.Logf("After first message: positions=%v, lastRenderedCount=%d", m.messageLinePositions, m.lastRenderedCount)

	// Add more messages via differential update
	m.messages = append(m.messages, Message{
		ID:        uuid.New(),
		Type:      MessageTypeTool,
		Content:   "Tool output",
		Timestamp: time.Now(),
		IsTool:    true,
		Collapsed: true,
	})
	m.updateViewportContent()
	m.viewport.SetContent(m.cachedContent)

	t.Logf("After tool message: positions=%v, lastRenderedCount=%d", m.messageLinePositions, m.lastRenderedCount)

	// Add one more
	m.messages = append(m.messages, Message{
		ID:        uuid.New(),
		Type:      MessageTypeAgent,
		Content:   "Agent response",
		Timestamp: time.Now(),
	})
	m.updateViewportContent()
	m.viewport.SetContent(m.cachedContent)

	t.Logf("After agent message: positions=%v, lastRenderedCount=%d", m.messageLinePositions, m.lastRenderedCount)

	// Now click on the tool message (index 1)
	toolStartLine := m.messageLinePositions[1]
	clickMsg := tea.MouseMsg{Type: tea.MouseLeft, Y: toolStartLine}
	resultModel, _ := m.handleClickOnToolMessage(clickMsg)
	result := resultModel.(Model)

	if result.selectedMessageIndex != 1 {
		t.Errorf("Click at Y=%d should select tool message (index 1), got %d",
			toolStartLine, result.selectedMessageIndex)
	}

	// Also verify clicking on the first message
	userStartLine := m.messageLinePositions[0]
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
		msgType := MessageTypeAgent
		content := fmt.Sprintf("Message %d with some content to make it longer", i)
		isTool := false

		// Every third message is a tool message
		if i%3 == 2 {
			msgType = MessageTypeTool
			content = fmt.Sprintf("Tool output %d", i)
			isTool = true
		}

		m.messages = append(m.messages, Message{
			ID:        uuid.New(),
			Type:      msgType,
			Content:   content,
			Timestamp: time.Now(),
			IsTool:    isTool,
			Collapsed: isTool, // Tool messages start collapsed
		})
	}

	m.updateViewportContent()
	m.viewport.SetContent(m.cachedContent)

	t.Logf("Message line positions: %v", m.messageLinePositions)
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
	userMsg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeUser,
		Content:   "Hello, please help me",
		Timestamp: time.Now(),
	}
	m.messages = append(m.messages, userMsg)
	content := m.updateViewportContent()
	m.viewport.SetContent(content)

	t.Logf("After user message: positions=%v, lastRenderedCount=%d", m.messageLinePositions, m.lastRenderedCount)

	// Now add a tool message (collapsed)
	toolMsg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeTool,
		Content:   "Tool output here",
		Timestamp: time.Now(),
		AgentID:   uuid.New(),
		AgentRole: "runner",
		IsTool:    true,
		Collapsed: true,
	}
	m.messages = append(m.messages, toolMsg)
	content = m.updateViewportContent()
	m.viewport.SetContent(content)

	t.Logf("After tool message: positions=%v, lastRenderedCount=%d", m.messageLinePositions, m.lastRenderedCount)
	t.Logf("User message lines: %d", m.messageLinePositions[1])
	t.Logf("Content:\n%s", content)

	// The tool message starts AFTER the user message + 2 blank lines
	// User message (5 lines) + 2 blank lines = tool message starts at line 7
	userLines := countLines(m.formatMessage(0, m.messages[0]))
	expectedToolStart := userLines + 2 // +2 for blank lines
	t.Logf("Expected tool start: %d, actual: %d", expectedToolStart, m.messageLinePositions[1])

	if m.messageLinePositions[1] != expectedToolStart {
		t.Errorf("Tool message position incorrect: expected %d, got %d", expectedToolStart, m.messageLinePositions[1])
	}

	// Now click at the START of the tool message
	toolStartLine := m.messageLinePositions[1]
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
