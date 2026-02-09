package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/denkhaus/gollum/pkg/logger"
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
		// Update viewport size (reserve space for input, footer, status bar, and log panel)
		reservedLines := 4 // status bar + prompt + footer
		if m.config.StatusEnabled {
			reservedLines++ // Extra line for status bar
		}
		logPanelHeight := 6                 // Reserve 6 lines for log panel
		reservedLines += logPanelHeight + 1 // +1 for log separator

		viewportHeight := m.height - reservedLines
		if viewportHeight < 1 {
			viewportHeight = 1
			// If we're very short, reduce log panel height
			logPanelHeight = m.height - reservedLines + logPanelHeight
			if logPanelHeight < 3 {
				logPanelHeight = 3 // Minimum log panel height
			}
		}
		m.viewport.Width = msg.Width
		m.viewport.Height = viewportHeight
		m.logViewport.Width = msg.Width
		m.logViewport.Height = logPanelHeight
		return m, nil

	case tickMsg:
		// Check if context was canceled
		select {
		case <-m.ctx.Done():
			m.quit = true
			return m, tea.Quit
		default:
		}
		return m, tea.Batch(m.tickCmd(), m.logTickCmd())

	case logTickMsg:
		// Fetch new log entries from logger service using sequence-based filtering
		if m.logService != nil {
			newLogs := m.logService.GetLogs(logger.LogFilter{
				SinceSeq: m.lastLogFetchSeq,
				Reverse:  false,
			})
			if len(newLogs) > 0 {
				// Update last fetch sequence to the most recent log's sequence
				m.lastLogFetchSeq = newLogs[len(newLogs)-1].Sequence
				// Format and append new log entries
				for _, log := range newLogs {
					formatted := fmt.Sprintf("%s [%s] %s",
						log.Timestamp.Format("15:04:05"),
						strings.ToUpper(log.Level),
						log.Message)
					m.logEntries = append(m.logEntries, formatted)
				}
				// Update log viewport content
				m.logViewport.SetContent(strings.Join(m.logEntries, "\n"))
				m.logViewport.GotoBottom()
			}
		}
		return m, m.logTickCmd()

	case agentCompleteMsg:
		m.agentExecuting = false
		m.agentStartTime = time.Time{} // Clear start time
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
			// Update viewport with error message
			m.viewport.SetContent(m.updateViewportContent())
			m.viewport.GotoTop()
		} else if msg.response != nil {
			// Only add response texts if NOT using AgentMessenger
			// When AgentMessenger is active (messageChan configured), messages are sent
			// via the channel during execution, avoiding duplicates
			if m.messageChan == nil {
				// Add response texts as individual messages (legacy mode)
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
				// Update viewport with new messages
				m.viewport.SetContent(m.updateViewportContent())
				m.viewport.GotoTop()
			}
			// When messageChan is active, messages were already added via newMessageMsg
			// during execution, so no viewport update needed here
		}

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
		m.viewport.GotoTop()
		// Continue listening for more messages
		return m, m.waitForMessages()

	case exportMsg:
		// Handle conversation export
		return m.handleExport()

	case tea.MouseMsg:
		// Handle mouse events with debouncing to prevent UI slowdown
		// Only process mouse wheel events for scrolling
		return m.handleMouseMsg(msg)

	case mouseDebounceMsg:
		// Process debounced mouse scroll
		// Only apply if the tag matches (this is the latest scroll event)
		if msg.tag == m.mouseDebounceTag {
			scrollLines := 3 // Scroll 3 lines per debounced event for smooth scrolling
			if msg.viewport == "logs" {
				if msg.direction < 0 {
					m.logViewport.LineUp(scrollLines)
				} else {
					m.logViewport.LineDown(scrollLines)
				}
			} else {
				if msg.direction < 0 {
					m.viewport.LineUp(scrollLines)
				} else {
					m.viewport.LineDown(scrollLines)
				}
			}
		}
		return m, nil

	default:
		// Update text input component with remaining messages (unless in search mode)
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
		m.viewport.GotoTop() // Start from top to show conversation from beginning

		// Add to history with size limit
		m.addToHistory(input)

		// Clear input and prepare for agent execution
		m.textInput.SetValue("")

		// Set agent execution state BEFORE creating the command
		// (this is needed because commands can't modify the model)
		m.agentExecuting = true
		m.agentStartTime = time.Now() // Track start time for elapsed timer
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
		// Arrow keys always scroll the active viewport
		// Scroll multiple lines at once for better performance
		scrollLines := 5 // Scroll 5 lines per keypress
		if m.activeViewport == "logs" {
			if msg.Type == tea.KeyUp {
				m.logViewport.LineUp(scrollLines)
			} else {
				m.logViewport.LineDown(scrollLines)
			}
			return m, nil
		}
		// Main viewport
		if msg.Type == tea.KeyUp {
			m.viewport.LineUp(scrollLines)
		} else {
			m.viewport.LineDown(scrollLines)
		}
		return m, nil

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

	case tea.KeyCtrlL:
		// Toggle between main and log viewport
		if m.activeViewport == "main" {
			m.activeViewport = "logs"
		} else {
			m.activeViewport = "main"
		}
		return m, nil

	case tea.KeyPgUp, tea.KeyPgDown:
		// Scroll the active viewport
		if m.activeViewport == "logs" {
			var cmd tea.Cmd
			m.logViewport, cmd = m.logViewport.Update(msg)
			return m, cmd
		}
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case tea.KeyShiftUp, tea.KeyShiftDown:
		// Scroll the active viewport (faster, alternative to Page Up/Down)
		scrollLines := 10 // Scroll 10 lines for faster navigation
		if m.activeViewport == "logs" {
			if msg.Type == tea.KeyShiftUp {
				m.logViewport.LineUp(scrollLines)
			} else {
				m.logViewport.LineDown(scrollLines)
			}
			return m, nil
		}
		if msg.Type == tea.KeyShiftUp {
			m.viewport.LineUp(scrollLines)
		} else {
			m.viewport.LineDown(scrollLines)
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

// handleMouseMsg handles mouse events with debouncing for smooth scrolling.
//
// This function implements the debounce pattern from Charmbracelet's example:
// https://github.com/charmbracelet/bubbletea/blob/main/examples/debounce/main.go
//
// The pattern works by:
// 1. Incrementing a tag on each mouse event
// 2. Scheduling a debounce command that includes the current tag
// 3. Only processing the scroll if the tag matches when the command fires
//
// This prevents rapid mouse wheel events from overwhelming the UI.
func (m Model) handleMouseMsg(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.MouseWheelUp, tea.MouseWheelDown:
		// Increment tag to invalidate any pending debounce commands
		m.mouseDebounceTag++

		// Determine scroll direction (-1 for up, 1 for down)
		direction := -1
		if msg.Type == tea.MouseWheelDown {
			direction = 1
		}

		// Return a debounce command with the current tag
		return m, tea.Tick(m.mouseDebounceDuration, func(_ time.Time) tea.Msg {
			return mouseDebounceMsg{
				tag:       m.mouseDebounceTag,
				direction: direction,
				viewport:  m.activeViewport,
			}
		})
	}

	// For other mouse events, pass to text input (for clicks, etc.)
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}
