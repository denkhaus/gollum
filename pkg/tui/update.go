package tui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Update handles incoming messages and updates the model state.
//
// This is the core event loop of the TUI. Each message (key press, mouse event,
// custom message) is processed to produce a new model state and optional commands.
//
// In Bubbletea's architecture, Update must be a pure function:
//   - Same input always produces same output
//   - No side effects (use Cmd for those)
//   - Returns new Model, never modifies in place
//
// # Phase 2 Update Logic
//
//   - tea.KeyMsg: Handle keyboard input
//   - ctrl+c: Graceful shutdown
//   - esc: Cancel active agent execution
//   - enter: Submit input to agent
//   - up/down: Navigate input history
//   - All other keys: Delegate to textinput component
//   - tickMsg: Check context cancellation, update timer
//   - agentCompleteMsg: Handle agent response/error
//   - WindowSizeMsg: Update terminal dimensions
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		// Check if context was canceled
		select {
		case <-m.ctx.Done():
			m.quit = true
			return m, tea.Quit
		default:
		}
		return m, m.tickCmd()

	case agentCompleteMsg:
		m.agentExecuting = false
		m.textInput.Focus()

		if msg.err != nil {
			m.err = msg.err
			m.messages = append(m.messages, "❌ Error: "+msg.err.Error())
		} else if msg.response != nil {
			m.messages = append(m.messages, msg.response.Texts...)
		}

		// Restore preserved input if user canceled during execution
		if m.cancelRequested && m.preservedInput != "" {
			m.textInput.SetValue(m.preservedInput)
			m.textInput.CursorEnd()
			m.preservedInput = ""
			m.cancelRequested = false
		}

		return m, nil

	default:
		// Update text input component
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}
}

// handleKeyMsg handles keyboard input.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle key combinations
	switch msg.Type {
	case tea.KeyCtrlC:
		// Trigger graceful shutdown
		m.quit = true
		m.messages = append(m.messages, "^C")
		return m, tea.Quit

	case tea.KeyEscape:
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

	case tea.KeyEnter:
		// Submit input to agent
		input := strings.TrimSpace(m.textInput.Value())
		if input == "" {
			return m, nil
		}

		// Check for exit commands
		if input == "quit" || input == "exit" {
			m.quit = true
			m.messages = append(m.messages, "👋 Goodbye!")
			return m, tea.Quit
		}

		// Add to history
		m.inputHistory = append(m.inputHistory, input)
		m.inputHistoryIndex = len(m.inputHistory)

		// Clear input and prepare for agent execution
		m.textInput.SetValue("")

		// Set agent execution state BEFORE creating the command
		// (this is needed because commands can't modify the model)
		m.agentExecuting = true
		m.agentStartTime = time.Now()
		m.textInput.Blur()

		// Create a per-request context that can be canceled
		reqCtx, cancel := m.ctx, func() {}
		if m.ctx != nil {
			reqCtx, cancel = context.WithCancel(m.ctx)
		}
		m.currentCancel = cancel

		// Return the command that will execute the agent
		return m, func() tea.Msg {
			defer cancel()
			m.currentCancel = nil

			response, err := m.agent.Execute(reqCtx, input)
			return agentCompleteMsg{response: response, err: err}
		}

	case tea.KeyUp, tea.KeyDown:
		// Handle history navigation
		return m.handleHistoryNavigation(msg.Type)

	default:
		// Pass other keys to textinput
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}
}

// handleHistoryNavigation handles up/down arrow for input history.
func (m Model) handleHistoryNavigation(keyType tea.KeyType) (tea.Model, tea.Cmd) {
	if len(m.inputHistory) == 0 {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(tea.KeyMsg{Type: keyType})
		return m, cmd
	}

	switch keyType {
	case tea.KeyUp:
		// Navigate to older history
		if m.inputHistoryIndex > 0 {
			m.inputHistoryIndex--
			m.textInput.SetValue(m.inputHistory[m.inputHistoryIndex])
			m.textInput.CursorEnd()
		}
	case tea.KeyDown:
		// Navigate to newer history
		if m.inputHistoryIndex < len(m.inputHistory)-1 {
			m.inputHistoryIndex++
			m.textInput.SetValue(m.inputHistory[m.inputHistoryIndex])
			m.textInput.CursorEnd()
		} else if m.inputHistoryIndex == len(m.inputHistory)-1 {
			// Clear input when going past the newest history item
			m.inputHistoryIndex = len(m.inputHistory)
			m.textInput.SetValue("")
		}
	}

	return m, nil
}
