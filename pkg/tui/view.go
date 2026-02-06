package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// View renders the model state as a string for display.
//
// This function is called after each Update to display the current state.
// It should be a pure function with no side effects.
//
// The view is rendered in order:
//  1. Messages area (shows conversation history)
//  2. Prompt with current text input or agent execution status
//  3. Instructions footer
func (m Model) View() string {
	if m.quit {
		return ""
	}

	var b strings.Builder

	// Messages section (with scroll handling)
	maxMessages := m.height - 3 // Reserve space for input and prompt
	if maxMessages < 1 {
		maxMessages = 1
	}

	displayMessages := m.messages
	if len(displayMessages) > maxMessages {
		displayMessages = displayMessages[len(displayMessages)-maxMessages:]
	}

	for _, msg := range displayMessages {
		b.WriteString(msg + "\n")
	}

	// Prompt and input section
	if m.agentExecuting {
		elapsed := time.Since(m.agentStartTime)
		b.WriteString(fmt.Sprintf("\r> [Executing... %v]", elapsed.Round(time.Second)))
	} else {
		b.WriteString("\r> " + m.textInput.View())
	}

	return b.String()
}

// NewProgramWithContext creates a new Bubbletea program with the TUI model.
//
// This is the main entry point for running the TUI with agent integration.
//
// Usage:
//
//	p := tui.NewProgramWithContext(ctx, agent)
//	if _, err := p.Run(); err != nil {
//	    log.Fatal(err)
//	}
func NewProgramWithContext(ctx context.Context, agent AgentExecutor) *tea.Program {
	m := NewModel(ctx, agent)
	return tea.NewProgram(m, tea.WithContext(ctx), tea.WithAltScreen())
}
