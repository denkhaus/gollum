package tui

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
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

	if newM.messages[0].Type != MessageTypeError {
		t.Error("Error message should have MessageTypeError")
	}

	if !strings.Contains(newM.messages[0].Content, "test error") {
		t.Error("Error message content should contain the error text")
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
	m.messages = []Message{
		{
			ID:        uuid.New(),
			Type:      MessageTypeSystem,
			Content:   "message 1",
			Timestamp: time.Now(),
		},
		{
			ID:        uuid.New(),
			Type:      MessageTypeSystem,
			Content:   "message 2",
			Timestamp: time.Now(),
		},
	}
	m.viewport.SetContent(m.updateViewportContent())

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

// Phase 4 Tests

func TestUpdate_SlashCommands(t *testing.T) {
	tests := []struct {
		name         string
		command      string
		wantQuit     bool
		wantMsgType  MessageType
		wantMsgCount int
	}{
		{
			name:         "/clear command",
			command:      "/clear",
			wantQuit:     false,
			wantMsgType:  MessageTypeSystem,
			wantMsgCount: 1, // Messages cleared system message
		},
		{
			name:         "/help command",
			command:      "/help",
			wantQuit:     false,
			wantMsgType:  MessageTypeSystem,
			wantMsgCount: 1, // Help system message
		},
		{
			name:         "/quit command",
			command:      "/quit",
			wantQuit:     true,
			wantMsgType:  MessageTypeSystem,
			wantMsgCount: 1, // Goodbye message
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			agent := &mockAgentExecutor{}
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

			if !tt.wantQuit && len(newM.messages) > 0 && newM.messages[0].Type != tt.wantMsgType {
				t.Errorf("Command should add %s message, got %s", tt.wantMsgType, newM.messages[0].Type)
			}
		})
	}
}

func TestUpdate_AltEnterMultiLine(t *testing.T) {
	ctx := context.Background()
	agent := &mockAgentExecutor{}
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
	newM = newModel.(Model)

	// Press Alt+Enter again to add another line
	newModel, _ = newM.Update(altEnterMsg)
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
	ctx := context.Background()
	agent := &mockAgentExecutor{}
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
	newModel, _ = newM.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("git")})
	newM = newModel.(Model)

	if newM.searchState.query != "git" {
		t.Error("Typing should update search query")
	}

	// Should have 3 matches (git status, git commit, git push)
	if len(newM.searchState.results) != 3 {
		t.Errorf("Search for 'git' should find 3 matches, got %d", len(newM.searchState.results))
	}

	// Press Esc to exit search
	newModel, _ = newM.Update(tea.KeyMsg{Type: tea.KeyEscape})
	newM = newModel.(Model)

	if newM.searchState.active {
		t.Error("Esc should exit search mode")
	}
}

func TestAddToHistory(t *testing.T) {
	ctx := context.Background()
	agent := &mockAgentExecutor{}
	m := NewModel(ctx, agent)

	// Set a small history max size for testing
	m.config.HistoryMaxSize = 3

	// Add some entries
	m.addToHistory("first")
	m.addToHistory("second")
	m.addToHistory("third")
	m.addToHistory("fourth")

	// Should only have 3 entries due to size limit
	if len(m.inputHistory) != 3 {
		t.Errorf("History should be limited to max size, got %d", len(m.inputHistory))
	}

	// Oldest entry should be dropped
	if m.inputHistory[0] != "second" {
		t.Error("Oldest entry should be dropped when exceeding max size")
	}

	// Newest entry should be present
	if m.inputHistory[2] != "fourth" {
		t.Error("Newest entry should be present")
	}

	// Test duplicate prevention
	m.addToHistory("fourth") // Same as last entry
	if len(m.inputHistory) != 3 {
		t.Error("Duplicate entry should not be added")
	}
}

func TestSearchHistory(t *testing.T) {
	ctx := context.Background()
	agent := &mockAgentExecutor{}
	m := NewModel(ctx, agent)

	m.inputHistory = []string{"git status", "git commit", "ls -la", "git log"}

	// Search for "git"
	results := m.searchHistory("git")
	if len(results) != 3 {
		t.Errorf("Search for 'git' should find 3 results, got %d", len(results))
	}

	// Verify indices are correct
	if results[0] != 0 || results[1] != 1 || results[2] != 3 {
		t.Error("Search results should have correct indices")
	}

	// Search for "ls"
	results = m.searchHistory("ls")
	if len(results) != 1 {
		t.Errorf("Search for 'ls' should find 1 result, got %d", len(results))
	}

	// Search for non-existent
	results = m.searchHistory("xyz")
	if len(results) != 0 {
		t.Error("Search for non-existent should find 0 results")
	}

	// Case insensitive search
	results = m.searchHistory("GIT")
	if len(results) != 3 {
		t.Error("Search should be case insensitive")
	}
}

func TestGetMessageCount(t *testing.T) {
	ctx := context.Background()
	agent := &mockAgentExecutor{}
	m := NewModel(ctx, agent)

	if m.getMessageCount() != 0 {
		t.Error("getMessageCount should return 0 for empty model")
	}

	m.messages = []Message{
		{ID: uuid.New(), Type: MessageTypeSystem, Content: "msg1", Timestamp: time.Now()},
		{ID: uuid.New(), Type: MessageTypeUser, Content: "msg2", Timestamp: time.Now()},
	}

	if m.getMessageCount() != 2 {
		t.Error("getMessageCount should return correct count")
	}
}

func TestRenderStatusBar(t *testing.T) {
	ctx := context.Background()
	agent := &mockAgentExecutor{}
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
	ctx := context.Background()
	agent := &mockAgentExecutor{}
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
	ctx := context.Background()
	agent := &mockAgentExecutor{}
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
	ctx := context.Background()
	agent := &mockAgentExecutor{}
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
	ctx := context.Background()
	agent := &mockAgentExecutor{}
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

	if len(newM.messages) > 0 && newM.messages[0].Type != MessageTypeError {
		t.Error("Unknown command should add error message type")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.HistoryMaxSize != 1000 {
		t.Error("Default history max size should be 1000")
	}

	if !config.EnableTimestamps {
		t.Error("Default should enable timestamps")
	}

	if !config.EnableColors {
		t.Error("Default should enable colors")
	}

	if !config.StatusEnabled {
		t.Error("Default should enable status bar")
	}

	if !config.MultiLineEnabled {
		t.Error("Default should enable multi-line input")
	}
}

func TestSetConfig(t *testing.T) {
	ctx := context.Background()
	agent := &mockAgentExecutor{}
	m := NewModel(ctx, agent)

	newConfig := Config{
		HistoryMaxSize:    500,
		EnableTimestamps: false,
		EnableColors:     false,
		StatusEnabled:    false,
		MultiLineEnabled: false,
	}

	m.SetConfig(newConfig)

	if m.config.HistoryMaxSize != 500 {
		t.Error("SetConfig should update history max size")
	}

	if m.config.EnableTimestamps {
		t.Error("SetConfig should update timestamps setting")
	}
}
