package tui

import (
	"github.com/m-mizutani/gollem"
	"context"
	"testing"


	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/mock/gomock"
)

func TestNewProgramWithContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	p := NewProgramWithContext(ctx, agent)

	if p == nil {
		t.Fatal("NewProgramWithContext() should not return nil")
	}
}

// Phase 4 Tests

func TestUpdate_SlashCommands(t *testing.T) {
	tests := []struct {
		name         string
		command      string
		wantQuit     bool
		wantRole gollem.MessageRole
		wantMsgCount int
	}{
		{
			name:         "/clear command",
			command:      "/clear",
			wantQuit:     false,
			wantRole: gollem.RoleSystem,
			wantMsgCount: 1, // Messages cleared system message
		},
		{
			name:         "/help command",
			command:      "/help",
			wantQuit:     false,
			wantRole: gollem.RoleSystem,
			wantMsgCount: 1, // Help system message
		},
		{
			name:         "/quit command",
			command:      "/quit",
			wantQuit:     true,
			wantRole: gollem.RoleSystem,
			wantMsgCount: 1, // Goodbye message
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := context.Background()
			agent := setupMockAgent(ctrl)
			m := NewModel(ctx, agent)

			// Type command
			newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.command)})
			m = newModel.(Model)

			// Press enter - this will call executeCommand internally
			newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

			if tt.wantQuit && cmd == nil {
				t.Error("Command should return quit command")
			}

			// Handle both pointer and value types from Update
			var newM Model
			if mPtr, ok := newModel.(*Model); ok {
				newM = *mPtr
			} else {
				newM = newModel.(Model)
			}

			if tt.wantQuit && !newM.quit {
				t.Error("Command should set quit flag")
			}

			if len(newM.messages) != tt.wantMsgCount {
				t.Errorf("Command should add %d message(s), got %d", tt.wantMsgCount, len(newM.messages))
			}

			if !tt.wantQuit && len(newM.messages) > 0 && newM.messages[0].Role != tt.wantRole {
				t.Errorf("Command should add %s message, got %s", tt.wantRole, newM.messages[0].Role)
			}
		})
	}
}

func TestUpdate_AltEnterMultiLine(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	// Type some input
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("first line")})
	m = newModel.(Model)

	// Press Alt+Enter to start multi-line mode
	altEnterMsg := tea.KeyMsg{Type: tea.KeyEnter, Alt: true}
	newModel, _ = m.Update(altEnterMsg)
	newM := newModel.(Model)

	if !newM.multiLineInput {
		t.Error("Alt+Enter should enable multi-line input mode")
	}

	if len(newM.multiLineBuffer) != 1 || newM.multiLineBuffer[0] != "first line" {
		t.Error("Alt+Enter should save current line to buffer")
	}

	// Type second line
	newModel, _ = newModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("second line")})
	_ = newModel.(Model) //nolint:staticcheck // value is never used, but type assertion validates model type

	// Press Alt+Enter again to add another line
	newModel, _ = newModel.Update(altEnterMsg)
	newM = newModel.(Model)

	if len(newM.multiLineBuffer) != 2 {
		t.Error("Second Alt+Enter should add line to buffer")
	}

	// Press Esc to cancel multi-line mode
	newModel, _ = newModel.Update(tea.KeyMsg{Type: tea.KeyEscape})
	newM = newModel.(Model)

	if newM.multiLineInput {
		t.Error("Esc should disable multi-line input mode")
	}

	if len(newM.multiLineBuffer) != 0 {
		t.Error("Esc should clear multi-line buffer")
	}
}

func TestUpdate_CtrlRSearch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	// Add some history
	m.inputHistory = []string{"git status", "git commit", "git push", "ls -la"}
	m.inputHistoryIndex = 4

	// Press Ctrl+R to start search
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	newM := newModel.(Model)

	if !newM.searchState.active {
		t.Error("Ctrl+R should activate search mode")
	}

	// Verify all history entries are initially in results
	if len(newM.searchState.results) != 4 {
		t.Errorf("Search should initially show all history, got %d results", len(newM.searchState.results))
	}

	// Type to filter (git)
	newModel, _ = newModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("git")})
	newM = newModel.(Model)

	if newM.searchState.query != "git" {
		t.Error("Typing should update search query")
	}

	// Should have 3 matches (git status, git commit, git push)
	if len(newM.searchState.results) != 3 {
		t.Errorf("Search for 'git' should find 3 matches, got %d", len(newM.searchState.results))
	}

	// Press Esc to exit search
	newModel, _ = newModel.Update(tea.KeyMsg{Type: tea.KeyEscape})
	newM = newModel.(Model)

	if newM.searchState.active {
		t.Error("Esc should exit search mode")
	}
}
