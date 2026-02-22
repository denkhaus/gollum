package tui

// This file handles keyboard input for the TUI.

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

// Viewport represents the active viewport in the TUI.
type Viewport string

const (
	// Viewport constants for active viewport management
	ViewportMain  Viewport = "main"
	ViewportInput Viewport = "input"
	ViewportLogs  Viewport = "logs"
)

// String returns the string representation of the viewport.
func (v Viewport) String() string {
	return string(v)
}

// handleKeyMsg handles keyboard input.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If in search mode, handle search-specific keys
	if m.searchState.active {
		return m.handleSearchKeyMsg(msg)
	}

	// Handle key combinations
	switch msg.Type {
	case tea.KeyCtrlC:
		return m.handleCtrlC()

	case tea.KeyEscape:
		return m.handleEscape()

	case tea.KeyEnter:
		return m.handleEnter(msg)

	case tea.KeyUp, tea.KeyDown:
		return m.handleArrowKeys(msg.Type)

	case tea.KeyCtrlR:
		return m.handleCtrlR()

	case tea.KeyCtrlL:
		return m.handleCtrlL()

	case tea.KeyPgUp, tea.KeyPgDown:
		return m.handlePageKeys(msg)

	case tea.KeyShiftUp, tea.KeyShiftDown:
		return m.handleShiftArrows(msg.Type)

	default:
		return m.handleDefaultKey(msg)
	}
}

// handleCtrlC handles graceful shutdown.
func (m Model) handleCtrlC() (tea.Model, tea.Cmd) {
	m.quit = true
	cancelMsg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeSystem,
		Content:   "^C",
		Timestamp: time.Now(),
	}
	m.addMessage(cancelMsg)
	m.viewport.SetContent(m.updateViewportContent())
	return m, tea.Quit
}

// handleEnter handles input submission or multi-line mode.
func (m Model) handleEnter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Check for Alt+Enter (multi-line input)
	if msg.Alt && m.config.MultiLineEnabled {
		return m.handleMultiLineToggle()
	}

	// Handle multi-line submission
	if m.multiLineInput && m.config.MultiLineEnabled {
		return m.submitMultiLineInput()
	}

	return m.handleSubmitInput()
}

// handleEscape handles escape key for multi-line mode and agent cancellation.
func (m Model) handleEscape() (tea.Model, tea.Cmd) {
	// Exit multi-line mode if active
	if m.multiLineInput && m.config.MultiLineEnabled {
		return m.exitMultiLineMode()
	}

	// Cancel current agent execution if active
	if m.agentExecuting && m.currentCancel != nil {
		m.cancelRequested = true
		// Preserve the current input
		m.preservedInput = m.textInput.Value()
		m.textInput.SetValue("")
		m.currentCancel()
		m.currentCancel = nil
	}

	return m, nil
}

// handleMultiLineToggle handles Alt+Enter to toggle multi-line input.
func (m Model) handleMultiLineToggle() (tea.Model, tea.Cmd) {
	if !m.multiLineInput {
		// Start multi-line mode
		m.multiLineInput = true
		// Save current input as first line
		currentInput := m.textInput.Value()
		if currentInput != "" {
			m.multiLineBuffer = []string{currentInput}
			m.textInput.SetValue("")
		}
	} else {
		// Add newline to current buffer
		currentInput := m.textInput.Value()
		m.multiLineBuffer = append(m.multiLineBuffer, currentInput)
		m.textInput.SetValue("")
	}
	return m, nil
}

// handleSubmitInput submits the input to the agent.
func (m Model) handleSubmitInput() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.textInput.Value())
	if input == "" {
		return m, nil
	}

	// Check for slash commands
	if strings.HasPrefix(input, "/") {
		return m.executeCommand(input)
	}

	// Check for exit commands (legacy)
	if input == "quit" || input == "exit" {
		return m.handleQuitCommand()
	}

	// Add user message to messages
	userMsg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeUser,
		Content:   input,
		Timestamp: time.Now(),
	}
	m.addMessage(userMsg)
	m.viewport.SetContent(m.updateViewportContent())
	m.viewport.GotoBottom() // Show latest message at bottom

	// Add to history with size limit
	m.addToHistory(input)

	// Clear input and prepare for agent execution
	m.textInput.SetValue("")

	// Prepare and create agent execution command
	m = m.prepareAgentExecution()

	// Return the command that will execute the agent
	return m, m.createAgentCommand(input)
}

// handleQuitCommand handles quit/exit commands.
func (m Model) handleQuitCommand() (tea.Model, tea.Cmd) {
	m.quit = true
	goodbyeMsg := Message{
		ID:        uuid.New(),
		Type:      MessageTypeSystem,
		Content:   "👋 Goodbye!",
		Timestamp: time.Now(),
	}
	m.addMessage(goodbyeMsg)
	m.viewport.SetContent(m.updateViewportContent())
	return m, tea.Quit
}

// handleArrowKeys handles arrow keys for viewport scrolling or input history.
// For viewport scrolling, uses immediate scroll with debounce protection against rapid key repeat.
func (m Model) handleArrowKeys(keyType tea.KeyType) (tea.Model, tea.Cmd) {
	// If input focus is active, use arrow keys for history navigation
	if m.activeViewport == ViewportInput {
		return m.handleHistoryNavigation(keyType)
	}

	// Scroll immediately
	scrollLines := 5
	if m.activeViewport == ViewportLogs {
		if keyType == tea.KeyUp {
			m.logViewport.ScrollUp(scrollLines)
		} else {
			m.logViewport.ScrollDown(scrollLines)
		}
		return m, nil
	}

	// Main viewport
	if keyType == tea.KeyUp {
		m.viewport.ScrollUp(scrollLines)
	} else {
		m.viewport.ScrollDown(scrollLines)
	}
	return m, nil
}

// handleKeyDebounceMsg processes debounced arrow key scroll events.
func (m Model) handleKeyDebounceMsg(msg keyDebounceMsg) (tea.Model, tea.Cmd) {
	// Only apply if the tag matches (this is the latest key event)
	if msg.tag == m.keyDebounceTag {
		m = m.scrollViewport(msg.viewport, msg.direction)
	}
	return m, nil
}

// handleCtrlR starts history search mode.
func (m Model) handleCtrlR() (tea.Model, tea.Cmd) {
	if len(m.inputHistory) > 0 {
		m.searchState = searchState{
			active:     true,
			query:      "",
			matchedIdx: 0,
			results:    make([]int, 0, len(m.inputHistory)),
		}
		// Initialize with all history entries
		for i := range m.inputHistory {
			m.searchState.results = append(m.searchState.results, i)
		}
		if len(m.searchState.results) > 0 {
			m.searchState.matchedIdx = len(m.searchState.results) - 1
			m.textInput.SetValue(m.inputHistory[m.searchState.results[m.searchState.matchedIdx]])
			m.textInput.CursorEnd()
		}
	}
	return m, nil
}

// handleCtrlL cycles through main viewport → log viewport → input focus (for history).
func (m Model) handleCtrlL() (tea.Model, tea.Cmd) {
	switch m.activeViewport {
	case "main":
		m.activeViewport = ViewportLogs
	case ViewportLogs:
		m.activeViewport = ViewportInput
	case ViewportInput:
		m.activeViewport = ViewportMain
	}
	return m, nil
}

// handlePageKeys handles page up/down keys.
func (m Model) handlePageKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If input focus is active, use page keys for faster history navigation
	if m.activeViewport == ViewportInput {
		if msg.Type == tea.KeyPgUp {
			return m.navigateHistory(tea.KeyUp), nil
		}
		return m.navigateHistory(tea.KeyDown), nil
	}

	// Scroll the active viewport
	if m.activeViewport == ViewportLogs {
		var cmd tea.Cmd
		m.logViewport, cmd = m.logViewport.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// handleShiftArrows handles shift+arrow keys for faster scrolling.
func (m Model) handleShiftArrows(keyType tea.KeyType) (tea.Model, tea.Cmd) {
	// If input focus is active, use shift+arrows for faster history navigation
	if m.activeViewport == ViewportInput {
		return m.navigateHistory(keyType), nil
	}

	// Scroll the active viewport (faster, alternative to Page Up/Down)
	scrollLines := 10 // Scroll 10 lines for faster navigation
	if m.activeViewport == ViewportLogs {
		if keyType == tea.KeyShiftUp {
			m.logViewport.LineUp(scrollLines)
		} else {
			m.logViewport.LineDown(scrollLines)
		}
		return m, nil
	}
	if keyType == tea.KeyShiftUp {
		m.viewport.LineUp(scrollLines)
	} else {
		m.viewport.LineDown(scrollLines)
	}
	return m, nil
}

// handleDefaultKey passes other keys to textinput.
func (m Model) handleDefaultKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}
