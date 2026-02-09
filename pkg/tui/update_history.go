package tui

// This file handles input history navigation for the TUI.

import tea "github.com/charmbracelet/bubbletea"

// handleHistoryNavigation handles up/down arrow for input history.
func (m Model) handleHistoryNavigation(keyType tea.KeyType) (tea.Model, tea.Cmd) {
	if len(m.inputHistory) == 0 {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(tea.KeyMsg{Type: keyType})
		return m, cmd
	}

	return m.navigateHistory(keyType), nil
}

// navigateHistory moves through input history based on key type.
func (m Model) navigateHistory(keyType tea.KeyType) Model {
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
	return m
}
