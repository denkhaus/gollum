// Package ui provides UI components and agent messenger implementations.
package ui

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
)

var (
	// Styles for different message types
	messageTypeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#9B59B6")). // Purple
				Bold(true).
				Width(10).
				Align(lipgloss.Center)

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3498DB")). // Blue border
			Padding(0, 1)

	timestampStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7F8C8D")). // Gray
			Faint(true).
			Width(10)

	headerStyle = lipgloss.NewStyle().
			Width(82) // Matches border width (80 + 2 padding)
)

type (
	// AgentMessenger handles visual display of agent communication
	AgentMessenger interface {
		// DisplayAgentMessage shows an agent's own message (LLM output or tool usage)
		DisplayAgentMessage(agentID uuid.UUID, agentRole, message string, isTool bool)
		// DisplayUserMessage shows a user's input message
		DisplayUserMessage(message string)
		// DisplaySystemInfo shows system-level information
		DisplaySystemInfo(message string)
		// DisplayWelcome shows the welcome message
		DisplayWelcome()
	}

	// agentMessengerImpl is the private implementation of AgentMessenger
	agentMessengerImpl struct {
		mu sync.Mutex
	}
)

// Ensure agentMessengerImpl implements AgentMessenger
var _ AgentMessenger = (*agentMessengerImpl)(nil)

// NewAgentMessenger creates a new agent messenger for dependency injection.
func NewAgentMessenger(_ do.Injector) (AgentMessenger, error) {
	return &agentMessengerImpl{}, nil
}

// DisplayAgentMessage shows an agent's own message (LLM output or tool usage)
func (p *agentMessengerImpl) DisplayAgentMessage(agentID uuid.UUID, agentRole, message string, isTool bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	agentShort := p.shortenAgentName(agentID, agentRole)

	var messageType, icon string
	if isTool {
		messageType = "Tool"
		icon = "⚡"
	} else {
		messageType = "Response"
		icon = "🤖"
	}

	// Create the header with proper spacing
	headerLeft := lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Foreground(lipgloss.Color("#2ECC71")).Bold(true).Render(fmt.Sprintf("%s %s", icon, agentShort)),
		lipgloss.NewStyle().Padding(0, 1).Render(messageTypeStyle.Render(messageType)),
	)
	header := lipgloss.JoinHorizontal(
		lipgloss.Top,
		headerLeft,
		timestampStyle.Render(time.Now().Format("15:04:05")),
	)
	header = headerStyle.Width(82).Render(header)

	// Create the full message
	fullMessage := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		borderStyle.Render(message),
	)

	// Print with spacing
	if _, err := fmt.Fprint(os.Stdout, "\n"+fullMessage+"\n\n"); err != nil {
		// Ignore write errors to stdout in UI code
		_ = err
	}
}

// DisplayUserMessage shows a user's input message
func (p *agentMessengerImpl) DisplayUserMessage(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	userStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3498DB")). // Blue
		Bold(true)

	header := headerStyle.Width(82).Render(lipgloss.JoinHorizontal(
		lipgloss.Top,
		userStyle.Render("👤 You"),
		timestampStyle.Render(time.Now().Format("15:04:05")),
	))

	fullMessage := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		borderStyle.Render(message),
	)

	if _, err := fmt.Fprint(os.Stdout, "\n"+fullMessage+"\n\n"); err != nil {
		// Ignore write errors to stdout in UI code
		_ = err
	}
}

// DisplaySystemInfo shows system-level information
func (p *agentMessengerImpl) DisplaySystemInfo(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F39C12")). // Orange
		Bold(true).
		Render("🚀 System")

	fullMessage := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		borderStyle.Render(message),
	)

	if _, err := fmt.Fprint(os.Stdout, "\n"+fullMessage+"\n\n"); err != nil {
		// Ignore write errors to stdout in UI code
		_ = err
	}
}

// shortenAgentName creates a short display name for agents
func (p *agentMessengerImpl) shortenAgentName(agentID uuid.UUID, role string) string {
	if role != "" && len(role) <= 10 {
		return role
	}

	// Use first 4 characters of ID as fallback
	idStr := agentID.String()
	return idStr[:4]
}

// formatContent formats the message content for display
func (p *agentMessengerImpl) formatContent(content map[string]any) string {
	var parts []string

	// Extract response if it exists
	if response, ok := content["response"].(string); ok {
		parts = append(parts, response)
	}

	// Extract agent info if it exists
	if agentRole, ok := content["agent_role"].(string); ok {
		parts = append(parts, fmt.Sprintf("Role: %s", agentRole))
	}

	// Extract error if it exists
	if err, ok := content["error"].(string); ok {
		parts = append(parts, fmt.Sprintf("❌ Error: %s", err))
	}

	// Handle other content types
	if len(parts) == 0 {
		// Fallback to string representation
		parts = append(parts, fmt.Sprintf("%v", content))
	}

	return strings.Join(parts, "\n")
}

// DisplayWelcome shows the welcome message
func (p *agentMessengerImpl) DisplayWelcome() {
	p.mu.Lock()
	defer p.mu.Unlock()

	welcomeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#2ECC71")).
		Bold(true).
		Padding(0, 2).
		Background(lipgloss.Color("#1E1E1E"))

	welcome := welcomeStyle.Render("🚀 Gollum Agent System - Interactive Mode")
	if _, err := fmt.Fprint(os.Stdout, "\n"+welcome+"\n\n"); err != nil {
		// Ignore write errors to stdout in UI code
		_ = err
	}
	if _, err := fmt.Fprint(os.Stdout, lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7F8C8D")).
		Render("Type 'quit' to exit • Agent communication will be displayed below\n")); err != nil {
		// Ignore write errors to stdout in UI code
		_ = err
	}
}
