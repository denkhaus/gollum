package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles for TUI rendering
var (
	// Prompt style for the input area
	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3498DB")). // Blue
			Bold(true)

	// Instructions style for help text
	instructionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7F8C8D")). // Gray
				Faint(true)

	// Border style for output box
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3498DB")). // Blue border
			Padding(0, 1)
)

// View renders the model state as a string for display.
//
// This function is called after each Update to display the current state.
// It should be a pure function with no side effects.
//
// The view is rendered in order:
//  1. Output area (shows accumulated messages)
//  2. Prompt with current text input
//  3. Instructions footer
//
// # Phase 1 View
//
// Simple rendering with:
//   - Text output area (empty or with user input echo)
//   - Focused text input with prompt
//   - Instructions (ctrl+c to quit, enter to submit)
func (m Model) View() string {
	var b strings.Builder

	// Title section
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#2ECC71")). // Green
		Bold(true).
		Render("🚀 Gollum TUI - Phase 1 (Bubbletea Foundation)")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Output section (if there's any output)
	if m.output != "" {
		b.WriteString(borderStyle.Render(m.output))
		b.WriteString("\n\n")
	}

	// Prompt and input section
	b.WriteString(promptStyle.Render("▶ "))
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")

	// Instructions footer
	instructions := instructionStyle.Render(
		"ctrl+c/esc: quit • enter: submit",
	)
	b.WriteString(instructions)

	return b.String()
}

// NewProgram creates a new Bubbletea program with the TUI model.
//
// This is the main entry point for running the TUI.
//
// Usage:
//
//	p := tui.NewProgram()
//	if _, err := p.Run(); err != nil {
//	    log.Fatal(err)
//	}
func NewProgram() *tea.Program {
	m := NewModel()
	return tea.NewProgram(m, tea.WithAltScreen()) // Use alt screen for full TUI experience
}
