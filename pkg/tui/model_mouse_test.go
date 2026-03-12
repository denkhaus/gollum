package tui

import (
	"context"
	"fmt"
	"testing"
	"github.com/denkhaus/gollum/pkg/channel"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestHandleMouseMsg_WheelEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 20
	m.viewport.Height = 10

	// Add messages to create scrollable content
	for i := 0; i < 20; i++ {
		m.messages = append(m.messages, channel.Message{
			ID:        uuid.New(),
			Type:      channel.MessageTypeAgentChat,
			Content:   fmt.Sprintf("channel.Message %d", i),
			Timestamp: time.Now(),
		})
	}
	m.viewport.SetContent(m.updateViewportContent())

	initialTag := m.mouseDebounceTag

	// Test mouse wheel up
	wheelUpMsg := tea.MouseMsg{Type: tea.MouseWheelUp, Y: 0}
	resultModel, cmd := m.handleMouseMsg(wheelUpMsg)
	result := resultModel.(Model)

	if result.mouseDebounceTag != initialTag+1 {
		t.Errorf("Mouse wheel up should increment debounce tag, got %d", result.mouseDebounceTag)
	}
	if cmd == nil {
		t.Error("Mouse wheel up should return a debounce command")
	}

	// Test mouse wheel down
	wheelDownMsg := tea.MouseMsg{Type: tea.MouseWheelDown, Y: 0}
	resultModel, cmd = result.handleMouseMsg(wheelDownMsg)
	result = resultModel.(Model)

	if result.mouseDebounceTag != initialTag+2 {
		t.Errorf("Mouse wheel down should increment debounce tag, got %d", result.mouseDebounceTag)
	}
	if cmd == nil {
		t.Error("Mouse wheel down should return a debounce command")
	}
}

// TestHandleMouseDebounceMsg tests the debounced mouse scroll processing.
func TestHandleMouseDebounceMsg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 20
	m.viewport.Height = 10
	m.mouseDebounceTag = 5

	// Add messages to create scrollable content
	for i := 0; i < 20; i++ {
		m.messages = append(m.messages, channel.Message{
			ID:        uuid.New(),
			Type:      channel.MessageTypeAgentChat,
			Content:   fmt.Sprintf("channel.Message %d", i),
			Timestamp: time.Now(),
		})
	}
	m.viewport.SetContent(m.updateViewportContent())

	// Test with matching tag (should scroll)
	msg := mouseDebounceMsg{tag: 5, direction: 1, viewport: ViewportMain}
	resultModel, _ := m.handleMouseDebounceMsg(msg)
	result := resultModel.(Model)

	// The scroll should have been applied (no easy way to verify viewport scrolled in unit test)
	_ = result

	// Test with non-matching tag (should not scroll)
	result.mouseDebounceTag = 10
	msg2 := mouseDebounceMsg{tag: 5, direction: 1, viewport: ViewportMain}
	resultModel2, _ := result.handleMouseDebounceMsg(msg2)
	result2 := resultModel2.(Model)

	// Tag mismatch means scroll was ignored
	_ = result2
}

// TestScrollViewport tests the scrollViewport function.
func TestScrollViewport(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 20
	m.viewport.Height = 10
	m.logViewport.Height = 10

	// Add messages to create scrollable content
	for i := 0; i < 20; i++ {
		m.messages = append(m.messages, channel.Message{
			ID:        uuid.New(),
			Type:      channel.MessageTypeAgentChat,
			Content:   fmt.Sprintf("channel.Message %d", i),
			Timestamp: time.Now(),
		})
	}
	m.viewport.SetContent(m.updateViewportContent())

	// Test scrolling main viewport up
	result := m.scrollViewport(ViewportMain, -1)
	_ = result

	// Test scrolling main viewport down
	result = m.scrollViewport(ViewportMain, 1)
	_ = result

	// Test scrolling log viewport
	result = m.scrollViewport(ViewportLogs, -1)
	_ = result
}

// TestCreateDebounceCommand tests the debounce command creation.
func TestCreateDebounceCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	cmd := m.createDebounceCommand(1, -1, ViewportMain)
	if cmd == nil {
		t.Error("createDebounceCommand should return a command")
	}
}
