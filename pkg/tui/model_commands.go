// Package tui provides Bubbletea-based terminal user interface components for Gollum.
//
// This file contains command handling and history management methods.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/denkhaus/gollum/pkg/shared"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
)

// addToHistory adds input to history with size limit enforcement.
func (m *Model) addToHistory(input string) {
	// Skip empty inputs and duplicates of the most recent entry
	if input == "" || (len(m.inputHistory) > 0 && m.inputHistory[len(m.inputHistory)-1] == input) {
		return
	}

	m.inputHistory = append(m.inputHistory, input)

	// Enforce history size limit
	if len(m.inputHistory) > m.config.HistoryMaxSize {
		// Keep only the most recent entries
		m.inputHistory = m.inputHistory[len(m.inputHistory)-m.config.HistoryMaxSize:]
	}

	// Reset history index to point to the "new" position (after the added entry)
	m.inputHistoryIndex = len(m.inputHistory)
}

// getMessageCount returns the total number of messages.
func (m Model) getMessageCount() int {
	return len(m.messages)
}

// executeCommand handles TUI slash commands.
func (m *Model) executeCommand(cmd string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return m, nil
	}

	command := parts[0]

	switch command {
	case "/clear":
		// Clear all messages and cache
		m.messages = []shared.Message{}
		m.clearFormatCache()
		m.viewport.SetContent("")
		systemMsg := shared.Message{
			ID:        uuid.New(),
			Role: gollem.RoleSystem,
			Content:   "Messages cleared",
			Timestamp: time.Now(),
		}
		m.addMessage(systemMsg)
		m.viewport.SetContent(m.updateViewportContent())
		m.viewport.GotoTop()
		return m, nil

	case "/quit":
		// Quit the TUI
		m.quit = true
		goodbyeMsg := shared.Message{
			ID:        uuid.New(),
			Role: gollem.RoleSystem,
			Content:   "👋 Goodbye!",
			Timestamp: time.Now(),
		}
		m.addMessage(goodbyeMsg)
		m.viewport.SetContent(m.updateViewportContent())
		return m, tea.Quit

	case "/help":
		// Show help message
		helpText := `TUI Commands:
/clear    - Clear all messages
/export   - Export conversation to file
/help     - Show this help message
/quit     - Exit the TUI

Keyboard Shortcuts:
Ctrl+C    - Quit
Esc       - Cancel active agent
Enter     - Submit input
Alt+Enter - New line (multi-line input)
↑/↓       - Navigate history
Ctrl+R    - Search history (type query, use C-s/C-r to navigate)`
		helpMsg := shared.Message{
			ID:        uuid.New(),
			Role: gollem.RoleSystem,
			Content:   helpText,
			Timestamp: time.Now(),
		}
		m.addMessage(helpMsg)
		m.viewport.SetContent(m.updateViewportContent())
		m.viewport.GotoTop()
		return m, nil

	case "/export":
		// Export conversation - return a command to handle the export
		return m, func() tea.Msg {
			return exportMsg{}
		}

	default:
		// Unknown command
		errorMsg := shared.Message{
			ID:        uuid.New(),
			Role: gollem.RoleSystem,
			Content:   fmt.Sprintf("Unknown command: %s. Type /help for available commands.", command),
			Timestamp: time.Now(),
		}
		m.addMessage(errorMsg)
		m.viewport.SetContent(m.updateViewportContent())
		m.viewport.GotoTop()
		return m, nil
	}
}
