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
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/markdown"
)

// AgentExecutor defines the interface for executing agent commands.
// This allows the TUI to interact with different agent implementations.
type AgentExecutor interface {
	Execute(ctx context.Context, input string) (*gollem.ExecuteResponse, error)
}

// tickMsg is sent periodically to update the UI (for agent execution timer).
type tickMsg time.Time

// logTickMsg is sent periodically to fetch new log entries.
type logTickMsg time.Time

// mouseDebounceMsg is sent after mouse wheel debouncing to perform the actual scroll.
// This prevents the UI from being overwhelmed by rapid mouse events.
type mouseDebounceMsg struct {
	tag       int    // Unique ID to identify this debounce batch
	direction int    // -1 for up, 1 for down
	viewport  string // "main" or "logs"
}

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

// Config holds TUI configuration from environment variables.
type Config struct {
	// HistoryMaxSize is the maximum number of history entries to keep
	HistoryMaxSize int

	// EnableTimestamps controls whether timestamps are shown in messages
	EnableTimestamps bool

	// EnableColors controls whether colors are used in the UI
	EnableColors bool

	// StatusEnabled controls whether the status bar is shown
	StatusEnabled bool

	// MultiLineEnabled controls whether multi-line input is enabled
	MultiLineEnabled bool
}

// DefaultConfig returns the default TUI configuration.
func DefaultConfig() Config {
	return Config{
		HistoryMaxSize:   1000,
		EnableTimestamps: true,
		EnableColors:     true,
		StatusEnabled:    true,
		MultiLineEnabled: true,
	}
}

// searchState represents the current state of history search (Ctrl+R).
type searchState struct {
	active     bool
	query      string
	matchedIdx int
	results    []int // Indices of matching history entries
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

	// currentCancel allows canceling the active agent execution
	currentCancel context.CancelFunc

	// agentStartTime tracks when the current agent execution started
	agentStartTime time.Time

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

	// activeViewport indicates which viewport receives keyboard scroll events
	// "main" or "logs"
	activeViewport string

	// messageChan receives messages from AgentMessenger (optional, for TUI mode)
	messageChan chan Message

	// viewport manages scrollable message display
	viewport viewport.Model

	// logViewport manages scrollable log output
	logViewport viewport.Model

	// lastLogFetchSeq tracks the last fetched log sequence number to avoid duplicates
	lastLogFetchSeq int64

	// logEntries holds the formatted log lines for display
	logEntries []string

	// logService provides access to fetch logs (injected via DI)
	logService logger.LoggerService

	// config holds TUI configuration
	config Config

	// markdownRenderer provides markdown rendering for agent responses
	markdownRenderer markdown.Renderer

	// searchState tracks history search state (Ctrl+R)
	searchState searchState

	// multiLineInput indicates whether multi-line input mode is active
	multiLineInput bool

	// multiLineBuffer stores the current multi-line input being composed
	multiLineBuffer []string

	// mouseDebounceTag is incremented on each mouse wheel event for debouncing
	mouseDebounceTag int

	// mouseDebounceDuration controls how long to wait before processing scroll events
	mouseDebounceDuration time.Duration
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
	ti.Prompt = ""   // Remove default prompt to avoid duplication with custom "> " in view
	ti.CharLimit = 0 // No limit for agent input

	// Initialize viewport with default size (will be updated on WindowSizeMsg)
	vp := viewport.New(0, 0)
	logVp := viewport.New(0, 0)

	return Model{
		textInput:             ti,
		quit:                  false,
		messages:              []Message{},
		agent:                 agent,
		ctx:                   ctx,
		inputHistory:          []string{},
		inputHistoryIndex:     -1,
		messageChan:           nil, // Will be set by SetMessageChannel
		viewport:              vp,
		logViewport:           logVp,
		lastLogFetchSeq:       -1, // -1 means fetch all logs initially
		logEntries:            []string{},
		config:                DefaultConfig(),
		searchState:           searchState{},
		multiLineInput:        false,
		multiLineBuffer:       []string{},
		activeViewport:        "main", // Start with main viewport active
		mouseDebounceTag:      0,
		mouseDebounceDuration: 130 * time.Millisecond, // 30ms debounce for smooth scrolling
	}
}

// SetLoggerService sets the logger service for fetching logs.
func (m *Model) SetLoggerService(service logger.LoggerService) {
	m.logService = service
}

// SetConfig sets the TUI configuration.
func (m *Model) SetConfig(config Config) {
	m.config = config
}

// SetMarkdownRenderer sets the markdown renderer for rich text display.
func (m *Model) SetMarkdownRenderer(renderer markdown.Renderer) {
	m.markdownRenderer = renderer
}

// Init initializes the TUI application.
//
// This function is called once at the start of the program.
// It returns the initial command (textinput focus, tick for timer updates, log tick, and message listener).
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{textinput.Blink, m.tickCmd(), m.logTickCmd()}
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

// logTickCmd returns a command that sends log tick messages to fetch new logs.
func (m Model) logTickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
		return logTickMsg(t)
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
	var b strings.Builder
	for _, msg := range m.messages {
		b.WriteString(m.formatMessage(msg))
		b.WriteString("\n\n")
	}
	return b.String()
}

// formatMessage formats a single message for display in the viewport.
// Uses lipgloss styling to match the AgentMessenger appearance.
// For agent messages, it uses the markdown renderer if available.
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

	// Calculate render width accounting for borders and glamour gutter.
	// Following Charmbracelet best practices:
	// https://github.com/charmbracelet/bubbletea/blob/main/examples/glamour/main.go
	//
	// The border style uses RoundedBorder() which adds 2 chars on each side.
	// Glamour also adds a 2-char gutter for line numbers/indentation.
	const (
		glamourGutter = 2 // Glamour's internal left gutter
		borderPadding = 4 // RoundedBorder: 2 left + 2 right
	)

	width := m.width - borderPadding - glamourGutter
	if width < 20 {
		width = 20 // Minimum width
	}

	// For agent messages, use markdown renderer if available
	// Tool messages and other types use plain word wrapping
	var contentText string
	if msg.Type == MessageTypeAgent && m.markdownRenderer != nil {
		// Use markdown renderer for rich agent responses
		rendered, err := m.markdownRenderer.Render(context.Background(), msg.Content, width)
		if err != nil {
			// Fallback to plain text if rendering fails
			contentText = msg.Content
		} else {
			contentText = rendered
		}
	} else {
		// Use plain text with word wrapping for other message types
		contentText = msg.Content
	}

	// Apply word wrapping for plain text content
	// (markdown content is already wrapped by the renderer)
	contentStyle := lipgloss.NewStyle().
		Width(width).
		MaxWidth(width)

	var wrappedContent string
	if msg.Type == MessageTypeAgent && m.markdownRenderer != nil {
		// Markdown renderer already handles wrapping
		wrappedContent = contentText
	} else {
		// Apply word wrapping for plain text
		wrappedContent = contentStyle.Render(contentText)
	}

	// Apply minimal border (no padding to save space)
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3498DB")) // Blue border

	content := borderStyle.Render(wrappedContent)

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

// searchHistory performs a case-insensitive search through history.
// Returns indices of matching entries.
func (m Model) searchHistory(query string) []int {
	var results []int
	query = strings.ToLower(query)

	for i, entry := range m.inputHistory {
		if strings.Contains(strings.ToLower(entry), query) {
			results = append(results, i)
		}
	}

	return results
}

// nextSearchResult navigates to the next search result.
func (m *Model) nextSearchResult() {
	if len(m.searchState.results) == 0 {
		return
	}

	m.searchState.matchedIdx = (m.searchState.matchedIdx + 1) % len(m.searchState.results)
	m.textInput.SetValue(m.inputHistory[m.searchState.results[m.searchState.matchedIdx]])
	m.textInput.CursorEnd()
}

// prevSearchResult navigates to the previous search result.
func (m *Model) prevSearchResult() {
	if len(m.searchState.results) == 0 {
		return
	}

	m.searchState.matchedIdx = (m.searchState.matchedIdx - 1 + len(m.searchState.results)) % len(m.searchState.results)
	m.textInput.SetValue(m.inputHistory[m.searchState.results[m.searchState.matchedIdx]])
	m.textInput.CursorEnd()
}

// exitSearch exits history search mode.
func (m *Model) exitSearch() {
	m.searchState = searchState{}
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
		// Clear all messages
		m.messages = []Message{}
		m.viewport.SetContent("")
		systemMsg := Message{
			ID:        uuid.New(),
			Type:      MessageTypeSystem,
			Content:   "Messages cleared",
			Timestamp: time.Now(),
		}
		m.messages = append(m.messages, systemMsg)
		m.viewport.SetContent(m.updateViewportContent())
		m.viewport.GotoTop()
		return m, nil

	case "/quit":
		// Quit the TUI
		m.quit = true
		goodbyeMsg := Message{
			ID:        uuid.New(),
			Type:      MessageTypeSystem,
			Content:   "👋 Goodbye!",
			Timestamp: time.Now(),
		}
		m.messages = append(m.messages, goodbyeMsg)
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
		helpMsg := Message{
			ID:        uuid.New(),
			Type:      MessageTypeSystem,
			Content:   helpText,
			Timestamp: time.Now(),
		}
		m.messages = append(m.messages, helpMsg)
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
		errorMsg := Message{
			ID:        uuid.New(),
			Type:      MessageTypeError,
			Content:   fmt.Sprintf("Unknown command: %s. Type /help for available commands.", command),
			Timestamp: time.Now(),
		}
		m.messages = append(m.messages, errorMsg)
		m.viewport.SetContent(m.updateViewportContent())
		m.viewport.GotoTop()
		return m, nil
	}
}

// exportMsg is sent when /export command is invoked.
type exportMsg struct{}
