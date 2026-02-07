package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// View renders the model state as a string for display.
//
// This function is called after each Update to display the current state.
// It should be a pure function with no side effects.
//
// The view is rendered in order:
//  1. Viewport with scrollable message history
//  2. Prompt with current text input or agent execution status
//  3. Instructions footer
func (m Model) View() string {
	if m.quit {
		return ""
	}

	var b strings.Builder

	// Viewport section with scrollable messages
	viewportHeight := m.height - 3 // Reserve space for input and footer
	if viewportHeight < 1 {
		viewportHeight = 1
	}
	m.viewport.Height = viewportHeight
	m.viewport.Width = m.width

	// If viewport is empty (first render), populate it
	if m.viewport.View() == "" && len(m.messages) > 0 {
		m.viewport.SetContent(m.updateViewportContent())
		m.viewport.GotoBottom()
	}

	b.WriteString(m.viewport.View())

	// Prompt and input section
	if m.agentExecuting {
		elapsed := time.Since(m.agentStartTime)
		promptStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F39C12")). // Orange
			Bold(true)
		b.WriteString("\n" + promptStyle.Render(fmt.Sprintf("> [Executing... %v]", elapsed.Round(time.Second))))
	} else {
		promptStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3498DB")). // Blue
			Bold(true)
		b.WriteString("\n" + promptStyle.Render("> ") + m.textInput.View())
	}

	// Footer with instructions
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7F8C8D")). // Gray
		Faint(true)
	b.WriteString("\n" + footerStyle.Render("Ctrl+C: Quit | Esc: Cancel Agent | Enter: Submit | ↑/↓: History"))

	return b.String()
}

// NewProgramWithContext creates a new Bubbletea program with the TUI model.
//
// This is the main entry point for running the TUI with agent integration.
// It also sets up the message channel for AgentMessenger integration.
//
// Usage:
//
//	messageChan := make(chan tui.Message, 100)
//	p := tui.NewProgramWithContext(ctx, agent, tui.WithMessageChannel(messageChan))
//	if _, err := p.Run(); err != nil {
//	    log.Fatal(err)
//	}
func NewProgramWithContext(ctx context.Context, agent AgentExecutor, opts ...func(*Model)) *tea.Program {
	m := NewModel(ctx, agent)
	for _, opt := range opts {
		opt(&m)
	}
	return tea.NewProgram(m, tea.WithContext(ctx), tea.WithAltScreen())
}

// WithMessageChannel is an option for NewProgramWithContext that sets the message channel.
// This allows AgentMessenger to send messages to the TUI instead of printing to stdout.
//
// Usage:
//
//	p := tui.NewProgramWithContext(ctx, agent, tui.WithMessageChannel())
func WithMessageChannel() func(*Model) {
	return func(m *Model) {
		// Create the internal message channel for the TUI
		ch := make(chan Message, 100)
		m.SetMessageChannel(ch)

		// Create and set the messenger channel for AgentMessenger integration
		// This allows AgentMessenger to send messages to the TUI
		adapterChan := make(chan MessageAdapter, 100)
		SetMessengerChannel(adapterChan)

		// Start a goroutine to bridge MessageAdapter to Message
		go func() {
			for adapterMsg := range adapterChan {
				msg := adapterMsg.ToMessage()
				ch <- msg
			}
			close(ch)
		}()
	}
}
