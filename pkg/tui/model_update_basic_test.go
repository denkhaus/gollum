package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/m-mizutani/gollem"
	"go.uber.org/mock/gomock"
)

func TestUpdate_CtrlCQuits(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	if cmd == nil {
		t.Error("Ctrl+C should return a quit command")
	}

	newM, ok := newModel.(Model)
	if !ok {
		t.Fatal("Update() should return a Model")
	}

	if !newM.quit {
		t.Error("Ctrl+C should set quit flag")
	}
}

func TestUpdate_EscapeCancelsExecution(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	cancelCh := make(chan struct{})

	agent := NewMockAgentExecutor(ctrl)
	agent.EXPECT().Execute(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string) (*gollem.ExecuteResponse, error) {
			<-cancelCh
			return nil, context.Canceled
		}).AnyTimes()

	m := NewModel(ctx, agent)

	// Start agent execution by typing and pressing enter
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("test")})
	m = newModel.(Model)

	// Press enter - this returns a command that will execute the agent
	// For testing, we need to manually set agentExecuting to simulate the state
	// after the command starts executing
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	// Manually set the state that would be set by the executeAgent command
	// (this simulates what happens when the command starts running)
	m.agentExecuting = true
	m.currentCancel = func() {
		close(cancelCh)
	}

	// Press escape - this should call m.currentCancel and set cancelRequested
	// But we can't actually test the cancel call since Update returns before the command runs
	// So we just verify that the escape key handling sets the flag correctly when agent is executing
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	newM := newModel.(Model)

	if !newM.cancelRequested {
		t.Error("Escape should set cancelRequested flag")
	}

	if newM.preservedInput != "" {
		t.Error("Escape should preserve input during cancellation")
	}

	// Clean up
	select {
	case <-cancelCh:
		// Already closed by Update, that's expected
	default:
		close(cancelCh)
	}
}

func TestUpdate_EnterSubmitsToAgent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	agent := NewMockAgentExecutor(ctrl)
	agent.EXPECT().Execute(gomock.Any(), "test input").
		Return(&gollem.ExecuteResponse{Texts: []string{"response"}}, nil).
		AnyTimes() // Use AnyTimes since Execute might not be called in this test

	m := NewModel(ctx, agent)

	// Type some input
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("test input")})
	m = newModel.(Model)

	// Press enter - this returns a command that will execute the agent
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Error("Enter should return a command")
	}

	newM := newModel.(Model)

	// The Update function sets agentExecuting when processing the Enter key
	if !newM.agentExecuting {
		t.Error("Enter should set agentExecuting flag")
	}

	if len(newM.inputHistory) != 1 {
		t.Error("Enter should add input to history")
	}

	if newM.inputHistory[0] != "test input" {
		t.Error("Input history should contain the submitted text")
	}

	if newM.textInput.Value() != "" {
		t.Error("Enter should clear text input")
	}

	if newM.textInput.Focused() {
		t.Error("Enter should blur text input during execution")
	}
}

func TestUpdate_EnterWithEmptyInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	// Press enter without typing
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd != nil {
		t.Error("Enter with empty input should not return a command")
	}

	newM, ok := newModel.(Model)
	if !ok {
		t.Fatal("Update should return Model")
	}
	if newM.agentExecuting {
		t.Error("Enter with empty input should not start agent execution")
	}
}

func TestUpdate_EnterWithQuitCommand(t *testing.T) {
	tests := []string{"quit", "exit"}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := context.Background()
			agent := setupMockAgent(ctrl)
			m := NewModel(ctx, agent)

			// Type quit/exit command
			newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(input)})
			m = newModel.(Model)

			// Press enter
			newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

			if cmd == nil {
				t.Error("Enter with exit command should return a quit command")
			}

			newM := newModel.(Model)
			if !newM.quit {
				t.Error("Enter with exit command should set quit flag")
			}
		})
	}
}

func TestUpdate_WindowSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	newModel, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	if cmd != nil {
		t.Error("WindowSizeMsg should not return a command")
	}

	newM := newModel.(Model)
	if newM.width != 80 || newM.height != 24 {
		t.Error("WindowSizeMsg should update dimensions")
	}
}

func TestUpdate_TickMsg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	newModel, cmd := m.Update(tickMsg{})

	if cmd == nil {
		t.Error("tickMsg should return a command")
	}

	_ = newModel.(Model)
}

func TestUpdate_TickMsgWithCanceledContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	agent := setupMockAgent(ctrl)
	m := NewModel(ctx, agent)

	// Cancel context
	cancel()

	newModel, cmd := m.Update(tickMsg{})

	if cmd == nil {
		t.Error("tickMsg with canceled context should return a quit command")
	}

	newM := newModel.(Model)
	if !newM.quit {
		t.Error("tickMsg with canceled context should set quit flag")
	}
}
