package tui

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/channel"
	"github.com/denkhaus/gollum/pkg/logger"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestHandleNewMessageMsg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 20

	newMsg := channel.Message{
		ID:        uuid.New(),
		Type:      channel.MessageTypeAgentChat,
		Content:   "Test message",
		Timestamp: time.Now(),
	}

	result, _ := m.handleNewMessageMsg(newMessageMsg{message: newMsg})

	if len(result.messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(result.messages))
	}
	if result.messages[0].Content != "Test message" {
		t.Errorf("Expected 'Test message', got %s", result.messages[0].Content)
	}
}

// TestWithLoggerService tests the WithLoggerService option.
func TestWithLoggerService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)

	mockLogger := logger.NewMockLoggerService(ctrl)
	opt := WithLoggerService(mockLogger)

	m := NewModel(ctx, agent)
	opt(&m)

	if m.logService == nil {
		t.Error("WithLoggerService should set logService")
	}
}

// TestHandleExport tests the export functionality.
func TestHandleExport(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 20

	// Add some messages
	m.messages = []channel.Message{
		{ID: uuid.New(), Type: channel.MessageTypeUserChat, Content: "Hello", Timestamp: time.Now()},
		{ID: uuid.New(), Type: channel.MessageTypeAgentChat, Content: "Hi there!", Timestamp: time.Now()},
	}

	// Test handleExport
	resultModel, _ := m.handleExport()
	result := resultModel.(Model)

	// Should have added a success or error message
	if len(result.messages) != 3 {
		t.Errorf("Expected 3 messages (2 original + export result), got %d", len(result.messages))
	}

	// Cleanup: remove the created export file
	t.Cleanup(func() {
		// The export file is created in the current working directory
		// Find and remove any gollum_export_*.txt files created during this test
		entries, err := os.ReadDir(".")
		if err != nil {
			t.Logf("Failed to read directory for cleanup: %v", err)
			return
		}
		for _, entry := range entries {
			if matched, _ := filepath.Match("gollum_export_*.txt", entry.Name()); matched {
				if err := os.Remove(entry.Name()); err != nil {
					t.Logf("Failed to remove export file %s: %v", entry.Name(), err)
				}
			}
		}
	})
}

// TestUpdate_LogTickMsg tests the Update function with logTickMsg.
func TestUpdate_LogTickMsg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 20

	// Test handling logTickMsg
	updatedModel, cmd := m.Update(logTickMsg{})
	if cmd == nil {
		t.Error("Update with logTickMsg should return a command")
	}
	_ = updatedModel // Model is updated
}

// TestUpdate_WindowSizeMsg tests the Update function with WindowSizeMsg.
func TestUpdate_WindowSizeMsg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	// Test handling WindowSizeMsg
	updatedModel, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	result := updatedModel.(Model)

	if result.width != 100 {
		t.Errorf("Expected width 100, got %d", result.width)
	}
	if result.height != 40 {
		t.Errorf("Expected height 40, got %d", result.height)
	}
}

// TestUpdate_KeyDebounceMsg tests the Update function with keyDebounceMsg.
func TestUpdate_KeyDebounceMsg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 20
	m.mouseDebounceTag = 5

	// Test handling keyDebounceMsg with matching tag
	debounceMsg := keyDebounceMsg{tag: 5, direction: -1, viewport: ViewportMain}
	updatedModel, _ := m.Update(debounceMsg)
	_ = updatedModel // Model is updated
}

// TestUpdate_SearchModeDefault tests the Update function default case in search mode.
func TestUpdate_SearchModeDefault(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.searchState.active = true

	// In search mode, default messages should return nil cmd
	updatedModel, cmd := m.Update(tea.FocusMsg{})
	result := updatedModel.(Model)

	if cmd != nil {
		t.Error("Update with unknown message in search mode should return nil command")
	}
	if !result.searchState.active {
		t.Error("Search mode should remain active")
	}
}

// TestHandleTickMsg_ContextCancelled tests tickMsg with cancelled context.
func TestHandleTickMsg_ContextCancelled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 20

	result, cmd := m.handleTickMsg(tickMsg{})
	if !result.quit {
		t.Error("Model should quit when context is cancelled")
	}
	if cmd == nil {
		t.Error("Should return tea.Quit command when context is cancelled")
	}
}

// TestHandleLogTickMsg tests handleLogTickMsg.
func TestHandleLogTickMsg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.width = 80
	m.height = 20

	result, cmd := m.handleLogTickMsg(logTickMsg{})
	if cmd == nil {
		t.Error("handleLogTickMsg should return a command")
	}
	_ = result
}

// TestFetchNewLogEntries tests fetchNewLogEntries without log service.
func TestFetchNewLogEntries_NoLogService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)
	m.logService = nil // No log service

	result := m.fetchNewLogEntries()
	if len(result.logEntries) != 0 {
		t.Error("Should have no log entries without log service")
	}
}
