package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

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

// appendToMultiLineBuffer adds current input to the multi-line buffer.
func (m Model) appendToMultiLineBuffer(input string) Model {
	m.multiLineBuffer = append(m.multiLineBuffer, input)
	return m
}
