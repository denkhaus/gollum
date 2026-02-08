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
//  2. Status bar (if enabled)
//  3. Prompt with current text input or agent execution status
//  4. Instructions footer
func (m Model) View() string {
	if m.quit {
		return ""
	}

	var b strings.Builder

	// Viewport section with scrollable messages
	viewportHeight := m.height - 4
	if m.config.StatusEnabled {
		viewportHeight-- // Extra line for status bar
	}
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

	// Status bar (if enabled)
	if m.config.StatusEnabled {
		b.WriteString("\n" + m.renderStatusBar())
	}

	// Render input section based on current mode
	switch {
	case m.multiLineInput && m.config.MultiLineEnabled:
		// Multi-line input indicator
		multiLineStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F39C12")). // Orange
			Bold(true)
		lineCount := len(m.multiLineBuffer)
		if current := m.textInput.Value(); current != "" {
			lineCount++
		}
		b.WriteString("\n" + multiLineStyle.Render(fmt.Sprintf("[Multi-line: %d lines] (Alt+Enter to add line, Enter to submit, Esc to cancel)", lineCount)))
		b.WriteString("\n" + m.textInput.View())

	case m.searchState.active:
		// Search mode indicator
		searchStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9B59B6")). // Purple
			Bold(true)
		matchedCount := len(m.searchState.results)
		if matchedCount > 0 {
			b.WriteString("\n" + searchStyle.Render(fmt.Sprintf("[Search: %s] (%d matches, %d/%d) Ctrl+S/R: nav, Enter: accept, Esc: exit]",
				m.searchState.query, matchedCount, m.searchState.matchedIdx+1, matchedCount)))
		} else {
			b.WriteString("\n" + searchStyle.Render(fmt.Sprintf("[Search: %s] (no matches) Esc: exit", m.searchState.query)))
		}

	default:
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
	}

	// Footer with instructions
	b.WriteString("\n" + m.renderFooter())

	return b.String()
}

// renderStatusBar renders the status bar with agent status and message count.
func (m Model) renderStatusBar() string {
	// Build status bar content
	statusParts := []string{}

	// Agent status
	if m.agentExecuting {
		statusParts = append(statusParts, lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F39C12")). // Orange
			Bold(true).
			Render("⚡ Executing"))
	} else {
		statusParts = append(statusParts, lipgloss.NewStyle().
			Foreground(lipgloss.Color("#2ECC71")). // Green
			Render("● Idle"))
	}

	// Message count
	statusParts = append(statusParts, lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7F8C8D")). // Gray
		Render(fmt.Sprintf("Messages: %d", m.getMessageCount())))

	// History size
	statusParts = append(statusParts, lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7F8C8D")). // Gray
		Render(fmt.Sprintf("History: %d/%d", len(m.inputHistory), m.config.HistoryMaxSize)))

	// Join parts with separator
	statusBar := lipgloss.JoinHorizontal(lipgloss.Left, statusParts...)

	// Apply status bar styling
	statusBarStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#2C3E50")). // Dark blue-gray
		Foreground(lipgloss.Color("#ECF0F1")). // Light gray
		Padding(0, 1).
		Width(m.width)

	return statusBarStyle.Render(statusBar)
}

// renderFooter renders the footer with keyboard shortcuts.
func (m Model) renderFooter() string {
	shortcuts := []string{
		"Ctrl+C: Quit",
		"Esc: Cancel/Exit",
		"Enter: Submit",
	}

	if m.config.MultiLineEnabled {
		shortcuts = append(shortcuts, "Alt+Enter: Multi-line")
	}

	shortcuts = append(shortcuts, "↑/↓: History", "Ctrl+R: Search")

	// Add multi-line specific shortcut if in multi-line mode
	if m.multiLineInput {
		shortcuts = []string{"Alt+Enter: New line", "Enter: Submit", "Esc: Cancel"}
	}

	// Add search specific shortcuts if in search mode
	if m.searchState.active {
		shortcuts = []string{"Ctrl+S/R: Navigate", "Enter: Accept", "Esc: Exit"}
	}

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7F8C8D")). // Gray
		Faint(true)

	// Join shortcuts with proper spacing
	footerText := strings.Join(shortcuts, " | ")
	return footerStyle.Render(footerText)
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
