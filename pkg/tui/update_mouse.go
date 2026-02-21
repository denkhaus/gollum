package tui

// This file handles mouse events for the TUI.

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

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
//
// For mouse clicks (tea.MouseLeft), this function detects clicks on tool message
// headers to toggle collapse/expand state.
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
		return m, m.createDebounceCommand(m.mouseDebounceTag, direction, m.activeViewport)

	case tea.MouseLeft:
		// Handle click on tool message header to toggle collapse state
		return m.handleClickOnToolMessage(msg)
	}

	// For other mouse events, pass to text input (for clicks, etc.)
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// handleClickOnToolMessage handles mouse clicks to toggle tool message collapse state.
// When a user clicks on a tool message header in the viewport, it toggles between
// collapsed (showing only summary) and expanded (showing full content).
func (m Model) handleClickOnToolMessage(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Only process clicks in the main viewport area
	// The viewport Y position starts after the header
	// Click Y is relative to the terminal, we need to convert to content position

	// Get the viewport's Y offset (how far the content is scrolled)
	yOffset := m.viewport.YOffset

	// Convert click Y to content line position
	// This is a simplified approach - we assume each visual line maps to one content line
	// The click Y needs to be within the viewport height
	viewportHeight := m.viewport.Height

	// Check if click is within the viewport area (not in input, status, or footer)
	// The viewport occupies the top portion of the screen
	// We need to account for the viewport's position in the layout
	clickY := msg.Y

	// Only process clicks within the viewport bounds
	if clickY < 0 || clickY >= viewportHeight {
		// Click is outside viewport, pass to text input
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	// Calculate the line number in the content (0-indexed from top of content)
	contentLine := yOffset + clickY

	// Find which message is at this line
	msgIdx := m.getMessageAtLine(contentLine)
	if msgIdx < 0 {
		// No message found at this line
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	// Toggle collapse state if it's a tool message
	if m.toggleMessageCollapse(msgIdx) {
		// Message was toggled, rebuild viewport content
		m.viewport.SetContent(m.updateViewportContent())
		return m, nil
	}

	// Not a tool message or couldn't toggle, pass to text input
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// handleMouseDebounceMsg processes debounced mouse scroll events.
func (m Model) handleMouseDebounceMsg(msg mouseDebounceMsg) (tea.Model, tea.Cmd) {
	// Only apply if the tag matches (this is the latest scroll event)
	if msg.tag == m.mouseDebounceTag {
		m = m.scrollViewport(msg.viewport, msg.direction)
	}
	return m, nil
}

// scrollViewport scrolls the specified viewport by the given direction.
func (m Model) scrollViewport(viewportName Viewport, direction int) Model {
	scrollLines := 3 // Scroll 3 lines per debounced event for smooth scrolling

	if viewportName == ViewportLogs {
		m = m.scrollLogViewport(direction, scrollLines)
	} else {
		m = m.scrollMainViewport(direction, scrollLines)
	}

	return m
}

// scrollMainViewport scrolls the main viewport.
func (m Model) scrollMainViewport(direction int, scrollLines int) Model {
	if direction < 0 {
		m.viewport.LineUp(scrollLines)
	} else {
		m.viewport.LineDown(scrollLines)
	}
	return m
}

// scrollLogViewport scrolls the log viewport.
func (m Model) scrollLogViewport(direction int, scrollLines int) Model {
	if direction < 0 {
		m.logViewport.LineUp(scrollLines)
	} else {
		m.logViewport.LineDown(scrollLines)
	}
	return m
}

// incrementDebounceTag increments the debounce tag and returns the new value.
func (m Model) incrementDebounceTag() (Model, int) {
	m.mouseDebounceTag++
	return m, m.mouseDebounceTag
}

// createDebounceCommand creates a debounce command for mouse scrolling.
func (m Model) createDebounceCommand(tag int, direction int, viewportName Viewport) tea.Cmd {
	return tea.Tick(m.mouseDebounceDuration, func(_ time.Time) tea.Msg {
		return mouseDebounceMsg{
			tag:       tag,
			direction: direction,
			viewport:  viewportName,
		}
	})
}
