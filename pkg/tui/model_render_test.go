package tui

import (
	"context"
	"strings"
	"testing"
	"github.com/denkhaus/gollum/pkg/channel"

	"go.uber.org/mock/gomock"
)

func TestRenderStatusBar(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	m.width = 80
	m.height = 20
	m.config.StatusEnabled = true

	statusBar := m.renderStatusBar()
	if statusBar == "" {
		t.Error("renderStatusBar should not return empty string")
	}

	if !strings.Contains(statusBar, "Messages: 0") {
		t.Error("Status bar should show message count")
	}

	if !strings.Contains(statusBar, "Idle") {
		t.Error("Status bar should show idle status")
	}
}

func TestRenderFooter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	m.config.MultiLineEnabled = true

	footer := m.renderFooter()
	if footer == "" {
		t.Error("renderFooter should not return empty string")
	}

	if !strings.Contains(footer, "Multi-line") {
		t.Error("Footer should show multi-line shortcut")
	}

	if !strings.Contains(footer, "Ctrl+R: Search") {
		t.Error("Footer should show search shortcut")
	}
}

func TestRenderFooterWithMultiLine(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	m.config.MultiLineEnabled = true
	m.multiLineInput = true

	footer := m.renderFooter()
	if !strings.Contains(footer, "New line") {
		t.Error("Footer in multi-line mode should show new line shortcut")
	}

	if !strings.Contains(footer, "Submit") {
		t.Error("Footer in multi-line mode should show submit shortcut")
	}
}

func TestRenderFooterWithSearch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	m.searchState.active = true
	m.searchState.results = []int{0, 1}

	footer := m.renderFooter()
	if !strings.Contains(footer, "Navigate") {
		t.Error("Footer in search mode should show navigation shortcuts")
	}

	if !strings.Contains(footer, "Accept") {
		t.Error("Footer in search mode should show accept shortcut")
	}
}

func TestExecuteCommand_Unknown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	// Try unknown command - executeCommand returns (tea.Model, tea.Cmd) which could be *Model
	result, _ := m.executeCommand("/unknown")

	// Handle both pointer and value types
	var newM Model
	if mPtr, ok := result.(*Model); ok {
		newM = *mPtr
	} else {
		newM = result.(Model)
	}

	if len(newM.messages) != 1 {
		t.Errorf("Unknown command should add error message, got %d messages", len(newM.messages))
	}

	if len(newM.messages) > 0 && newM.messages[0].Type != channel.MessageTypeError {
		t.Error("Unknown command should add error message type")
	}
}
