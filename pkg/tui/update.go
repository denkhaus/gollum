package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
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
// # Phase 4 Update Logic
//
//   - tea.KeyMsg: Handle keyboard input
//   - ctrl+c: Graceful shutdown
//   - esc: Cancel active agent execution or exit search mode
//   - enter: Submit input to agent or add newline in multi-line mode
//   - alt+enter: Toggle multi-line input mode
//   - up/down: Navigate input history or search results
//   - ctrl+r: Start history search
//   - ctrl+s/ctrl+r in search: Navigate search results
//   - All other keys: Delegate to textinput component
//   - tickMsg: Check context cancellation, update timer
//   - agentCompleteMsg: Handle agent response/error
//   - newMessageMsg: Handle new messages from AgentMessenger
//   - WindowSizeMsg: Update terminal dimensions and viewport
//   - exportMsg: Handle conversation export
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Update viewport size (reserve space for input, footer, and status bar)
		viewportHeight := m.height - 4
		if m.config.StatusEnabled {
			viewportHeight-- // Extra line for status bar
		}
		if viewportHeight < 1 {
			viewportHeight = 1
		}
		m.viewport.Width = msg.Width
		m.viewport.Height = viewportHeight
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
			// Add error message to messages
			errorMsg := Message{
				ID:        uuid.New(),
				Type:      MessageTypeError,
				Content:   msg.err.Error(),
				Timestamp: time.Now(),
			}
			m.messages = append(m.messages, errorMsg)
		} else if msg.response != nil {
			// Add response texts as individual messages
			for _, text := range msg.response.Texts {
				agentMsg := Message{
					ID:        uuid.New(),
					Type:      MessageTypeAgent,
					Content:   text,
					Timestamp: time.Now(),
					AgentID:   uuid.Nil, // Will be set by agent messenger
					AgentRole: "",
				}
				m.messages = append(m.messages, agentMsg)
			}
		}

		// Update viewport with new messages
		m.viewport.SetContent(m.updateViewportContent())
		m.viewport.GotoBottom()

		// Restore preserved input if user canceled during execution
		if m.cancelRequested && m.preservedInput != "" {
			m.textInput.SetValue(m.preservedInput)
			m.textInput.CursorEnd()
			m.preservedInput = ""
			m.cancelRequested = false
		}

		return m, nil

	case newMessageMsg:
		// Add new message from AgentMessenger
		m.messages = append(m.messages, msg.message)
		// Update viewport with new messages
		m.viewport.SetContent(m.updateViewportContent())
		m.viewport.GotoBottom()
		// Continue listening for more messages
		return m, m.waitForMessages()

	case exportMsg:
		// Handle conversation export
		return m.handleExport()

	default:
		// Update text input component (unless in search mode)
		if !m.searchState.active {
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
		return m, nil
	}
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
		// Trigger graceful shutdown
		m.quit = true
		cancelMsg := Message{
			ID:        uuid.New(),
			Type:      MessageTypeSystem,
			Content:   "^C",
			Timestamp: time.Now(),
		}
		m.messages = append(m.messages, cancelMsg)
		m.viewport.SetContent(m.updateViewportContent())
		return m, tea.Quit

	case tea.KeyEscape:
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

	case tea.KeyEnter:
		// Check for Alt+Enter (multi-line input)
		if msg.Alt && m.config.MultiLineEnabled {
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

		// Handle multi-line submission
		if m.multiLineInput && m.config.MultiLineEnabled {
			return m.submitMultiLineInput()
		}

		// Submit input to agent
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
			m.quit = true
			goodbyeMsg := Message{
				ID:        uuid.New(),
				Type:      MessageTypeSystem,
				Content:   "👋 Goodbye!",
				Timestamp: time.Now(),
			}
			m.messages = append(m.messages, goodbyeMsg)
			m.viewport.SetContent(m.updateViewportContent())
			return m, tea.Quit
		}

		// Add user message to messages
		userMsg := Message{
			ID:        uuid.New(),
			Type:      MessageTypeUser,
			Content:   input,
			Timestamp: time.Now(),
		}
		m.messages = append(m.messages, userMsg)
		m.viewport.SetContent(m.updateViewportContent())
		m.viewport.GotoBottom()

		// Add to history with size limit
		m.addToHistory(input)

		// Clear input and prepare for agent execution
		m.textInput.SetValue("")

		// Set agent execution state BEFORE creating the command
		// (this is needed because commands can't modify the model)
		m.agentExecuting = true
		m.agentStartTime = time.Now()
		m.textInput.Blur()

		// Create a per-request context that can be canceled
		// Defensive: ensure we always have a cancellable context
		var reqCtx context.Context
		var cancel context.CancelFunc
		if m.ctx != nil {
			reqCtx, cancel = context.WithCancel(m.ctx)
		} else {
			reqCtx, cancel = context.WithCancel(context.Background())
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

	case tea.KeyCtrlR:
		// Start history search mode
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

	default:
		// Pass other keys to textinput
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}
}

// handleSearchKeyMsg handles keyboard input during history search.
func (m Model) handleSearchKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		// Exit search mode
		m.exitSearch()
		return m, nil

	case tea.KeyEnter:
		// Accept the selected history entry and exit search mode
		m.exitSearch()
		return m, nil

	case tea.KeyCtrlS, tea.KeyCtrlR:
		// Navigate to next/previous search result
		if msg.Type == tea.KeyCtrlR {
			m.prevSearchResult()
		} else {
			m.nextSearchResult()
		}
		return m, nil

	case tea.KeyRunes:
		// Update search query as user types
		input := string(msg.Runes)
		m.textInput.SetValue(input)
		m.searchState.query = input

		// Update search results
		if input == "" {
			// Show all history when query is empty
			m.searchState.results = make([]int, 0, len(m.inputHistory))
			for i := range m.inputHistory {
				m.searchState.results = append(m.searchState.results, i)
			}
		} else {
			m.searchState.results = m.searchHistory(input)
		}

		// Update display to show first match
		if len(m.searchState.results) > 0 {
			m.searchState.matchedIdx = 0
			m.textInput.SetValue(m.inputHistory[m.searchState.results[0]])
		}
		return m, nil

	case tea.KeyBackspace:
		// Handle backspace in search mode
		current := m.textInput.Value()
		if len(current) > 0 {
			m.textInput.SetValue(current[:len(current)-1])
			m.searchState.query = m.textInput.Value()

			// Update search results
			if m.searchState.query == "" {
				m.searchState.results = make([]int, 0, len(m.inputHistory))
				for i := range m.inputHistory {
					m.searchState.results = append(m.searchState.results, i)
				}
			} else {
				m.searchState.results = m.searchHistory(m.searchState.query)
			}

			// Update display
			if len(m.searchState.results) > 0 {
				m.searchState.matchedIdx = 0
				m.textInput.SetValue(m.inputHistory[m.searchState.results[0]])
			}
		}
		return m, nil

	default:
		// Ignore other keys in search mode
		return m, nil
	}
}

// submitMultiLineInput submits the multi-line buffer as a single message.
func (m Model) submitMultiLineInput() (tea.Model, tea.Cmd) {
	currentInput := m.textInput.Value()
	if currentInput != "" {
		m.multiLineBuffer = append(m.multiLineBuffer, currentInput)
	}

	// Join all lines with newlines
	input := strings.Join(m.multiLineBuffer, "\n")
	input = strings.TrimSpace(input)

	if input == "" {
		return m.exitMultiLineMode()
	}

	// Exit multi-line mode first
	m.multiLineInput = false
	m.multiLineBuffer = []string{}
	m.textInput.SetValue(input)

	// Now submit as regular input
	return m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEnter})
}

// exitMultiLineMode exits multi-line input mode, discarding the buffer.
func (m Model) exitMultiLineMode() (tea.Model, tea.Cmd) {
	m.multiLineInput = false
	m.multiLineBuffer = []string{}
	m.textInput.SetValue("")
	return m, nil
}

// handleExport handles conversation export to a file.
func (m Model) handleExport() (tea.Model, tea.Cmd) {
	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("gollum_export_%s.txt", timestamp)

	// Build export content
	var content strings.Builder
	content.WriteString("# Gollum Conversation Export\n")
	content.WriteString(fmt.Sprintf("# Exported: %s\n", time.Now().Format(time.RFC3339)))
	content.WriteString(fmt.Sprintf("# Total Messages: %d\n", len(m.messages)))
	content.WriteString(strings.Repeat("=", 60) + "\n\n")

	for _, msg := range m.messages {
		timestamp := msg.Timestamp.Format("2006-01-02 15:04:05")
		var prefix string

		switch msg.Type {
		case MessageTypeUser:
			prefix = fmt.Sprintf("[%s] 👤 You:", timestamp)
		case MessageTypeAgent:
			prefix = fmt.Sprintf("[%s] 🤖 Agent:", timestamp)
		case MessageTypeTool:
			prefix = fmt.Sprintf("[%s] ⚡ Tool:", timestamp)
		case MessageTypeSystem:
			prefix = fmt.Sprintf("[%s] 🚀 System:", timestamp)
		case MessageTypeError:
			prefix = fmt.Sprintf("[%s] ❌ Error:", timestamp)
		default:
			prefix = fmt.Sprintf("[%s] ❓ Unknown:", timestamp)
		}

		content.WriteString(prefix + "\n")
		content.WriteString(msg.Content)
		content.WriteString("\n\n")
	}

	// Write to file
	err := os.WriteFile(filename, []byte(content.String()), 0644)
	if err != nil {
		errorMsg := Message{
			ID:        uuid.New(),
			Type:      MessageTypeError,
			Content:   fmt.Sprintf("Failed to export conversation: %v", err),
			Timestamp: time.Now(),
		}
		m.messages = append(m.messages, errorMsg)
	} else {
		successMsg := Message{
			ID:        uuid.New(),
			Type:      MessageTypeSystem,
			Content:   fmt.Sprintf("Conversation exported to: %s", filename),
			Timestamp: time.Now(),
		}
		m.messages = append(m.messages, successMsg)
	}

	m.viewport.SetContent(m.updateViewportContent())
	m.viewport.GotoBottom()
	return m, nil
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
