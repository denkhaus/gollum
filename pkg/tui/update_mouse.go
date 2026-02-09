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
	}

	// For other mouse events, pass to text input (for clicks, etc.)
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
func (m Model) scrollViewport(viewportName string, direction int) Model {
	scrollLines := 3 // Scroll 3 lines per debounced event for smooth scrolling

	if viewportName == "logs" {
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
func (m Model) createDebounceCommand(tag int, direction int, viewportName string) tea.Cmd {
	return tea.Tick(m.mouseDebounceDuration, func(_ time.Time) tea.Msg {
		return mouseDebounceMsg{
			tag:       tag,
			direction: direction,
			viewport:  viewportName,
		}
	})
}
