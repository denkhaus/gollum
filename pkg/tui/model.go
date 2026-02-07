// Package tui provides Bubbletea-based terminal user interface components for Gollum.
//
// This package establishes the foundation for a modern, event-driven TUI that will
// eventually replace the current stdinReader-based input system. It uses Bubbletea's
// Model-Update-View architecture for clean separation of concerns.
//
// # Architecture
//
// The TUI follows Bubbletea's elm architecture:
//   - Model: Application state (textInput, quit flag, output buffer)
//   - Init: Initial command that starts the application
//   - Update: Event handler that returns new model + commands
//   - View: Renderer that converts model to string output
//
// # Usage
//
// To run the TUI:
//
//	p := tui.NewProgramWithContext(ctx, agent)
//	if _, err := p.Run(); err != nil {
//	    log.Fatal(err)
//	}
//
// # Phase Scope
//
//   - Phase 1: Basic Bubbletea model with text input
//   - Phase 2: Replace stdinReader with Bubbletea input (current)
//   - Phase 3: Integrate AgentMessenger with Bubbletea viewport
//   - Phase 4: Enhanced features and polish
package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
)

// AgentExecutor defines the interface for executing agent commands.
// This allows the TUI to interact with different agent implementations.
type AgentExecutor interface {
	Execute(ctx context.Context, input string) (*gollem.ExecuteResponse, error)
}

// tickMsg is sent periodically to update the UI (for agent execution timer).
type tickMsg time.Time

// agentCompleteMsg is sent when agent execution completes.
type agentCompleteMsg struct {
	response *gollem.ExecuteResponse
	err      error
}

// MessageType represents the type of message being displayed.
type MessageType int

const (
	// MessageTypeUser represents a message from the user.
	MessageTypeUser MessageType = iota
	// MessageTypeAgent represents a message from an agent (LLM output).
	MessageTypeAgent
	// MessageTypeTool represents a tool execution message.
	MessageTypeTool
	// MessageTypeSystem represents a system-level message.
	MessageTypeSystem
	// MessageTypeError represents an error message.
	MessageTypeError
)

// String returns the string representation of the MessageType.
func (mt MessageType) String() string {
	switch mt {
	case MessageTypeUser:
		return "user"
	case MessageTypeAgent:
		return "agent"
	case MessageTypeTool:
		return "tool"
	case MessageTypeSystem:
		return "system"
	case MessageTypeError:
		return "error"
	default:
		return "unknown"
	}
}

// Message represents a single message in the conversation history.
// Messages are displayed in the viewport with styling based on type.
type Message struct {
	// ID is the unique identifier for this message.
	ID uuid.UUID

	// Type is the category of message (user, agent, tool, system, error).
	Type MessageType

	// Content is the message text to display.
	Content string

	// Timestamp is when the message was created.
	Timestamp time.Time

	// AgentID is the ID of the agent that sent this message (for agent/tool messages).
	AgentID uuid.UUID

	// AgentRole is the display role/name of the agent.
	AgentRole string

	// IsTool indicates whether this is a tool execution message.
	IsTool bool
}

// newMessageMsg is sent when a new message should be added to the viewport.
type newMessageMsg struct {
	message Message
}

// Model represents the application state for the TUI.
//
// In Bubbletea's architecture, the Model is a pure data structure that
// contains all application state. The Model is immutable between Update cycles.
type Model struct {
	// textInput is the Bubbletea text input component for user entry
	textInput textinput.Model

	// quit indicates when the user wants to exit the application
	quit bool

	// messages stores conversation history with full message metadata
	messages []Message

	// agent executes user commands
	agent AgentExecutor

	// ctx is the application context for cancellation
	ctx context.Context

	// agentExecuting indicates whether an agent is currently running
	agentExecuting bool

	// agentStartTime tracks when the current agent execution started
	agentStartTime time.Time

	// currentCancel allows canceling the active agent execution
	currentCancel context.CancelFunc

	// err stores the last agent error
	err error

	// inputHistory stores previous user inputs for up/down navigation
	inputHistory []string

	// inputHistoryIndex tracks the current position in input history
	inputHistoryIndex int

	// preservedInput stores the input when canceling during execution
	preservedInput string

	// cancelRequested indicates if the user requested cancellation
	cancelRequested bool

	// width and height store the terminal dimensions
	width  int
	height int

	// messageChan receives messages from AgentMessenger (optional, for TUI mode)
	messageChan chan Message

	// viewport manages scrollable message display
	viewport viewport.Model
}

// NewModel creates a new TUI model with initial state.
//
// The model is initialized with:
//   - A focused text input component
//   - Quit flag set to false
//   - Empty messages buffer
//   - Agent executor and context for command execution
//
// Returns a Model ready for use with tea.Program.
func NewModel(ctx context.Context, agent AgentExecutor) Model {
	ti := textinput.New()
	ti.Focus()
	ti.Placeholder = ""
	ti.CharLimit = 0 // No limit for agent input

	// Initialize viewport with default size (will be updated on WindowSizeMsg)
	vp := viewport.New(0, 0)

	return Model{
		textInput:         ti,
		quit:              false,
		messages:          []Message{},
		agent:             agent,
		ctx:               ctx,
		inputHistory:      []string{},
		inputHistoryIndex: -1,
		messageChan:       nil, // Will be set by SetMessageChannel
		viewport:          vp,
	}
}

// Init initializes the TUI application.
//
// This function is called once at the start of the program.
// It returns the initial command (textinput focus, tick for timer updates, and message listener).
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{textinput.Blink, m.tickCmd()}
	if msgCmd := m.waitForMessages(); msgCmd != nil {
		cmds = append(cmds, msgCmd)
	}
	return tea.Batch(cmds...)
}

// tickCmd returns a command that sends tick messages for UI updates.
func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// SetMessageChannel sets the channel for receiving messages from AgentMessenger.
// This enables the TUI to receive messages instead of AgentMessenger printing to stdout.
func (m *Model) SetMessageChannel(ch chan Message) {
	m.messageChan = ch
}

// GetMessageChannel returns the message channel for AgentMessenger to send messages.
func (m Model) GetMessageChannel() chan Message {
	return m.messageChan
}

// waitForMessages returns a command that waits for messages on the message channel.
// This should be included in tea.Batch to listen for incoming messages.
func (m Model) waitForMessages() tea.Cmd {
	if m.messageChan == nil {
		return nil
	}
	return func() tea.Msg {
		msg, ok := <-m.messageChan
		if !ok {
			return nil // Channel closed
		}
		return newMessageMsg{message: msg}
	}
}

// updateViewportContent updates the viewport with the current messages.
func (m Model) updateViewportContent() string {
	var content string
	for _, msg := range m.messages {
		content += m.formatMessage(msg) + "\n\n"
	}
	return content
}

// formatMessage formats a single message for display in the viewport.
// Uses lipgloss styling to match the AgentMessenger appearance.
func (m Model) formatMessage(msg Message) string {
	timestamp := msg.Timestamp.Format("15:04:05")

	var header string
	switch msg.Type {
	case MessageTypeUser:
		userStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3498DB")). // Blue
			Bold(true)
		header = lipgloss.JoinHorizontal(
			lipgloss.Top,
			userStyle.Render("👤 You"),
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7F8C8D")). // Gray
				Faint(true).
				Render(" "+timestamp),
		)

	case MessageTypeAgent, MessageTypeTool:
		agentName := formatAgentName(msg.AgentID, msg.AgentRole)
		var icon, messageType string
		if msg.IsTool || msg.Type == MessageTypeTool {
			messageType = "Tool"
			icon = "⚡"
		} else {
			messageType = "Response"
			icon = "🤖"
		}

		// Match AgentMessenger styling
		messageTypeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9B59B6")). // Purple
			Bold(true).
			Padding(0, 1)

		headerLeft := lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#2ECC71")). // Green
				Bold(true).
				Render(fmt.Sprintf("%s %s", icon, agentName)),
			messageTypeStyle.Render(messageType),
		)

		header = lipgloss.JoinHorizontal(
			lipgloss.Top,
			headerLeft,
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7F8C8D")). // Gray
				Faint(true).
				Render(" "+timestamp),
		)

	case MessageTypeSystem:
		header = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F39C12")). // Orange
			Bold(true).
			Render("🚀 System " + timestamp)

	case MessageTypeError:
		header = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E74C3C")). // Red
			Bold(true).
			Render("❌ Error " + timestamp)

	default:
		header = fmt.Sprintf("❓ Unknown %s", timestamp)
	}

	// Render content with border styling (simplified from AgentMessenger)
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3498DB")). // Blue border
		Padding(0, 1)

	content := borderStyle.Render(msg.Content)

	return header + "\n" + content
}

// formatAgentName creates a short display name for agents.
func formatAgentName(agentID uuid.UUID, role string) string {
	if role != "" && len(role) <= 10 {
		return role
	}
	// Use first 4 characters of ID as fallback
	idStr := agentID.String()
	if len(idStr) >= 4 {
		return idStr[:4]
	}
	return "agent"
}
