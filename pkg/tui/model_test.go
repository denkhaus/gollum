package tui

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/m-mizutani/gollem"
)

// mockAgentExecutor is a mock implementation of AgentExecutor for testing
type mockAgentExecutor struct {
	executeFunc func(ctx context.Context, input string) (*gollem.ExecuteResponse, error)
}

func (m *mockAgentExecutor) Execute(ctx context.Context, input string) (*gollem.ExecuteResponse, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, input)
	}
	return &gollem.ExecuteResponse{Texts: []string{"response"}}, nil
}

func TestNewModel(t *testing.T) {
	ctx := context.Background()
	agent := &mockAgentExecutor{}
	m := NewModel(ctx, agent)

	// Verify initial state
	if m.quit {
		t.Error("NewModel() should have quit=false")
	}

	if len(m.messages) != 0 {
		t.Error("NewModel() should have empty messages")
	}

	// Verify text input is initialized
	if !m.textInput.Focused() {
		t.Error("NewModel() textInput should be focused")
	}
}

func TestInit(t *testing.T) {
	ctx := context.Background()
	agent := &mockAgentExecutor{}
	m := NewModel(ctx, agent)

	cmd := m.Init()
	if cmd == nil {
		t.Error("Init() should return a command")
	}
}

func TestUpdate_CtrlCQuits(t *testing.T) {
	ctx := context.Background()
	agent := &mockAgentExecutor{}
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
	ctx := context.Background()
	cancelCh := make(chan struct{})
	agent := &mockAgentExecutor{
		executeFunc: func(_ context.Context, _ string) (*gollem.ExecuteResponse, error) {
			<-cancelCh
			return nil, context.Canceled
		},
	}
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
	ctx := context.Background()
	agent := &mockAgentExecutor{
		executeFunc: func(_ context.Context, input string) (*gollem.ExecuteResponse, error) {
			if input != "test input" {
				t.Errorf("Execute() input = %q, want %q", input, "test input")
			}
			return &gollem.ExecuteResponse{Texts: []string{"response"}}, nil
		},
	}
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
	ctx := context.Background()
	agent := &mockAgentExecutor{}
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
			ctx := context.Background()
			agent := &mockAgentExecutor{}
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
	ctx := context.Background()
	agent := &mockAgentExecutor{}
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
	ctx := context.Background()
	agent := &mockAgentExecutor{}
	m := NewModel(ctx, agent)

	newModel, cmd := m.Update(tickMsg{})

	if cmd == nil {
		t.Error("tickMsg should return a command")
	}

	_ = newModel.(Model)
}

func TestUpdate_TickMsgWithCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	agent := &mockAgentExecutor{}
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

func TestUpdate_AgentCompleteMsg(t *testing.T) {
	ctx := context.Background()
	agent := &mockAgentExecutor{}
	m := NewModel(ctx, agent)

	// Start agent execution
	m.agentExecuting = true
	m.textInput.Blur()

	// Send agent complete message
	response := &gollem.ExecuteResponse{Texts: []string{"test response"}}
	newModel, cmd := m.Update(agentCompleteMsg{response: response, err: nil})

	if cmd != nil {
		t.Error("agentCompleteMsg should not return a command")
	}

	newM := newModel.(Model)
	if newM.agentExecuting {
		t.Error("agentCompleteMsg should clear agentExecuting flag")
	}

	if !newM.textInput.Focused() {
		t.Error("agentCompleteMsg should focus text input")
	}

	if len(newM.messages) != 1 {
		t.Error("agentCompleteMsg should add response to messages")
	}
}

func TestUpdate_AgentCompleteMsgWithError(t *testing.T) {
	ctx := context.Background()
	agent := &mockAgentExecutor{}
	m := NewModel(ctx, agent)

	// Start agent execution
	m.agentExecuting = true
	m.textInput.Blur()

	// Send agent complete message with error
	testErr := fmt.Errorf("test error")
	newModel, cmd := m.Update(agentCompleteMsg{response: nil, err: testErr})

	if cmd != nil {
		t.Error("agentCompleteMsg with error should not return a command")
	}

	newM := newModel.(Model)
	if newM.agentExecuting {
		t.Error("agentCompleteMsg with error should clear agentExecuting flag")
	}

	if len(newM.messages) != 1 {
		t.Error("agentCompleteMsg with error should add error message")
	}

	if !strings.Contains(newM.messages[0], "Error:") {
		t.Error("Error message should be formatted correctly")
	}
}

func TestUpdate_HistoryNavigation(t *testing.T) {
	ctx := context.Background()
	agent := &mockAgentExecutor{}
	m := NewModel(ctx, agent)

	// Add some history
	m.inputHistory = []string{"first", "second", "third"}
	m.inputHistoryIndex = 3

	// Navigate up
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	newM := newModel.(Model)

	if newM.textInput.Value() != "third" {
		t.Error("Up arrow should navigate to latest history item")
	}

	if newM.inputHistoryIndex != 2 {
		t.Error("Up arrow should decrement history index")
	}

	// Navigate up again
	newModel, _ = newM.Update(tea.KeyMsg{Type: tea.KeyUp})
	newM = newModel.(Model)

	if newM.textInput.Value() != "second" {
		t.Error("Up arrow should navigate to older history item")
	}

	// Navigate down
	newModel, _ = newM.Update(tea.KeyMsg{Type: tea.KeyDown})
	newM = newModel.(Model)

	if newM.textInput.Value() != "third" {
		t.Error("Down arrow should navigate to newer history item")
	}

	// Navigate down past the end
	newModel, _ = newM.Update(tea.KeyMsg{Type: tea.KeyDown})
	newM = newModel.(Model)

	if newM.textInput.Value() != "" {
		t.Error("Down arrow past end should clear input")
	}

	if newM.inputHistoryIndex != 3 {
		t.Error("Down arrow past end should reset index")
	}
}

func TestUpdate_HistoryNavigationWithEmptyHistory(t *testing.T) {
	ctx := context.Background()
	agent := &mockAgentExecutor{}
	m := NewModel(ctx, agent)

	// Navigate up with empty history
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	newM := newModel.(Model)

	if newM.textInput.Value() != "" {
		t.Error("Up arrow with empty history should not change input")
	}
}

func TestView(t *testing.T) {
	ctx := context.Background()
	agent := &mockAgentExecutor{}
	m := NewModel(ctx, agent)

	view := m.View()
	if view == "" {
		t.Error("View() should not return empty string")
	}

	if !strings.Contains(view, ">") {
		t.Error("View() should contain prompt")
	}
}

func TestViewWithAgentExecuting(t *testing.T) {
	ctx := context.Background()
	agent := &mockAgentExecutor{}
	m := NewModel(ctx, agent)

	m.agentExecuting = true
	m.agentStartTime = m.agentStartTime.Add(-1 * time.Second) // Set to 1 second ago

	view := m.View()
	if !strings.Contains(view, "Executing") {
		t.Error("View() with executing agent should show execution status")
	}
}

func TestViewWithMessages(t *testing.T) {
	ctx := context.Background()
	agent := &mockAgentExecutor{}
	m := NewModel(ctx, agent)

	// Set height to allow multiple messages to be displayed
	m.height = 20
	m.messages = []string{"message 1", "message 2"}

	view := m.View()

	// The view adds newlines after each message
	hasMessage1 := strings.Contains(view, "message 1")
	hasMessage2 := strings.Contains(view, "message 2")

	if !hasMessage1 || !hasMessage2 {
		t.Errorf("View() should contain all messages, got: %q", view)
	}
}

func TestNewProgramWithContext(t *testing.T) {
	ctx := context.Background()
	agent := &mockAgentExecutor{}
	p := NewProgramWithContext(ctx, agent)

	if p == nil {
		t.Fatal("NewProgramWithContext() should not return nil")
	}
}
