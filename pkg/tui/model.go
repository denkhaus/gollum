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
	tag       int      // Unique ID to identify this debounce batch
	direction int      // -1 for up, 1 for down
	viewport  Viewport // ViewportMain or ViewportLogs
}

// keyDebounceMsg is sent after arrow key debouncing to perform the actual scroll.
// This prevents the UI from being overwhelmed by rapid keyboard events.
type keyDebounceMsg struct {
	tag       int      // Unique ID to identify this debounce batch
	direction int      // -1 for up, 1 for down
	viewport  Viewport // ViewportMain or ViewportLogs
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

	// Collapsed indicates whether this message is collapsed (tool messages only).
	// When collapsed, only a summary header is shown. Click to expand/collapse.
	Collapsed bool
}

// newMessageMsg is sent when a new message should be added to the viewport.
type newMessageMsg struct {
	message Message
}

// Config holds TUI configuration from environment variables.
type Config struct {
	// HistoryMaxSize is the maximum number of history entries to keep
	HistoryMaxSize int

	// MaxMessages is the maximum number of messages to keep in the conversation history
	// When exceeded, oldest messages are removed (ring buffer behavior)
	MaxMessages int

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
		MaxMessages:      500, // Cap message history to prevent unbounded growth
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
	activeViewport Viewport

	// messageChan receives messages from AgentMessenger (optional, for TUI mode)
	messageChan chan Message

	// messengerChan is the adapter channel for AgentMessenger integration
	// This must be stored for cleanup when the TUI shuts down
	messengerChan chan MessageAdapter

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

	// keyDebounceTag is incremented on each arrow key event for debouncing
	keyDebounceTag int

	// keyDebounceDuration controls how long to wait before processing arrow key scroll events
	keyDebounceDuration time.Duration

	// formatCache caches formatted messages to avoid expensive re-formatting on every viewport update
	// Key: message ID (uuid.UUID), Value: formatted string
	formatCache map[uuid.UUID]string

	// cacheWidth tracks the terminal width when cache was built
	// Used to invalidate cache on window resize (since formatting depends on width)
	cacheWidth int

	// maxCacheSize limits cache size to prevent unbounded growth
	maxCacheSize int

	// lastRenderedCount tracks the number of messages that were rendered in the last viewport update
	// Used for differential rendering to only append new messages
	lastRenderedCount int

	// cachedContent stores the complete viewport content string to avoid rebuilding
	// Invalidated on width change or message deletion
	cachedContent string

	// lineToMessage is a direct mapping from content line number to message index.
	// lineToMessage[lineNum] = messageIdx means that line `lineNum` belongs to message `messageIdx`.
	// This is built during content rendering and provides O(1) click detection.
	// When content changes (expand/collapse, new messages), this array is rebuilt.
	lineToMessage []int

	// selectedMessageIndex tracks which message is currently selected (-1 = none)
	// A selected message displays a bold border to indicate selection
	selectedMessageIndex int

	// lastClickTime tracks when the last click occurred for double-click detection
	lastClickTime time.Time

	// lastClickedMessageIndex tracks which message was last clicked for double-click detection
	// Double-clicks must be on the same message as the first click
	// Using message index instead of Y position makes this robust to content changes
	lastClickedMessageIndex int

	// doubleClickThreshold is the maximum time between clicks to count as double-click
	doubleClickThreshold time.Duration
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
		activeViewport:        ViewportInput, // Start with input focus for history navigation
		mouseDebounceTag:      0,
		mouseDebounceDuration: 130 * time.Millisecond, // 130ms debounce for smooth scrolling
		keyDebounceTag:        0,
		keyDebounceDuration:   16 * time.Millisecond, // 16ms debounce for arrow keys (one frame)
		formatCache:           make(map[uuid.UUID]string),
		cacheWidth:            0,                      // Will be set on first update
		maxCacheSize:          1000,                   // Match HistoryMaxSize default
		selectedMessageIndex:  -1,                     // No message selected initially
		doubleClickThreshold:  500 * time.Millisecond, // 500ms for double-click detection
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

// addMessage appends a message to the history while enforcing the MaxMessages limit.
// When the limit is exceeded, oldest messages are removed (ring buffer behavior).
// This also cleans up the format cache and differential rendering cache for evicted messages.
func (m *Model) addMessage(msg Message) {
	m.messages = append(m.messages, msg)

	// Enforce message limit - remove oldest messages if exceeded
	maxMsgs := m.config.MaxMessages
	if maxMsgs <= 0 {
		maxMsgs = 500 // Fallback to default if not configured
	}

	if len(m.messages) > maxMsgs {
		// Calculate how many messages to remove
		evictCount := len(m.messages) - maxMsgs

		// Clean up format cache for evicted messages
		for i := 0; i < evictCount; i++ {
			delete(m.formatCache, m.messages[i].ID)
		}

		// Remove oldest messages (ring buffer behavior)
		m.messages = m.messages[evictCount:]

		// Invalidate differential rendering cache when messages are evicted
		// This ensures the viewport is rebuilt correctly
		m.cachedContent = ""
		m.lastRenderedCount = 0
	}
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
	return tea.Tick(time.Millisecond*750, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// logTickCmd returns a command that sends log tick messages to fetch new logs.
func (m Model) logTickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*1000, func(t time.Time) tea.Msg {
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

// SetMessengerChannel sets the adapter channel for AgentMessenger integration.
// This stores the channel for cleanup when the TUI shuts down.
func (m *Model) SetMessengerChannel(ch chan MessageAdapter) {
	m.messengerChan = ch
}

// GetMessengerChannel returns the adapter channel for AgentMessenger integration.
func (m Model) GetMessengerChannel() chan MessageAdapter {
	return m.messengerChan
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
// Uses differential rendering to only append new messages instead of rebuilding everything.
// This is a performance optimization to avoid O(n) re-formatting on every update.
// Also tracks message line positions for mouse click detection.
func (m *Model) updateViewportContent() string {
	currentCount := len(m.messages)

	// Check if we need a full rebuild (cache invalidation scenarios)
	// 1. Width changed - formatting depends on width
	// 2. Messages were removed (e.g., /clear) - need to rebuild from scratch
	// 3. No cached content yet - first call
	// 4. Message collapse state changed - need to rebuild for updated formatting
	needsRebuild := m.cacheWidth != m.width ||
		currentCount < m.lastRenderedCount ||
		m.cachedContent == ""

	if needsRebuild {
		// Full rebuild: iterate through all messages
		var b strings.Builder
		// Reset line-to-message mapping for O(1) click detection
		m.lineToMessage = make([]int, 0)

		for i, msg := range m.messages {
			// Add blank lines BEFORE each message (except the first)
			// Map them to -1 (no message) so viewport line indices match array indices
			if i > 0 {
				b.WriteString("\n")                           // First blank line
				m.lineToMessage = append(m.lineToMessage, -1) // Map blank line to -1
				b.WriteString("\n")                           // Second blank line
				m.lineToMessage = append(m.lineToMessage, -1) // Map blank line to -1
			}

			formatted := m.formatMessage(i, msg)
			b.WriteString(formatted)

			// Build direct line-to-message mapping
			// Each line of this message maps to message index i
			linesInMsg := countLines(formatted)
			for line := 0; line < linesInMsg; line++ {
				m.lineToMessage = append(m.lineToMessage, i)
			}
		}
		m.cachedContent = b.String()
		m.lastRenderedCount = currentCount
		m.cacheWidth = m.width
		return m.cachedContent
	}

	// Differential update: only append new messages
	// This is the hot path for normal operation
	if currentCount > m.lastRenderedCount {
		var b strings.Builder
		// Start with existing content
		b.WriteString(m.cachedContent)

		// Append only new messages
		for i := m.lastRenderedCount; i < currentCount; i++ {
			// Add blank lines BEFORE each message (except if this is the first message)
			// Map them to -1 (no message) so viewport line indices match array indices
			if i > 0 {
				b.WriteString("\n")                           // First blank line
				m.lineToMessage = append(m.lineToMessage, -1) // Map blank line to -1
				b.WriteString("\n")                           // Second blank line
				m.lineToMessage = append(m.lineToMessage, -1) // Map blank line to -1
			}

			formatted := m.formatMessage(i, m.messages[i])
			b.WriteString(formatted)

			// Build direct line-to-message mapping
			// Each line of this message maps to message index i
			linesInMsg := countLines(formatted)
			for line := 0; line < linesInMsg; line++ {
				m.lineToMessage = append(m.lineToMessage, i)
			}
		}

		m.cachedContent = b.String()
		m.lastRenderedCount = currentCount
	}

	return m.cachedContent
}

// clearFormatCache clears the format cache and cached content.
// This is called when cache becomes invalid (e.g., width change, message update).
func (m *Model) clearFormatCache() {
	m.formatCache = make(map[uuid.UUID]string)
	m.cachedContent = ""
	m.lastRenderedCount = 0
	m.lineToMessage = nil
}

// getMessageAtLine finds the message index at the given line number in the viewport content.
// Returns -1 if no message is found at that line.
// The line number is 0-indexed from the top of the viewport content.
// Uses direct O(1) lookup via lineToMessage array.
func (m Model) getMessageAtLine(lineNum int) int {
	// Direct lookup - O(1)
	if lineNum >= 0 && lineNum < len(m.lineToMessage) {
		return m.lineToMessage[lineNum]
	}
	return -1
}

// getMessageStartLine finds the first line number where a given message appears.
// Returns -1 if the message is not found in the lineToMessage mapping.
// This is used for tests that need to find the starting line of a specific message.
func (m Model) getMessageStartLine(msgIdx int) int {
	for line, idx := range m.lineToMessage {
		if idx == msgIdx {
			return line
		}
	}
	return -1
}

// toggleMessageCollapse toggles the collapsed state of a tool message at the given index.
// Returns true if the message was toggled, false if it wasn't a tool message.
func (m *Model) toggleMessageCollapse(msgIdx int) bool {
	if msgIdx < 0 || msgIdx >= len(m.messages) {
		return false
	}

	msg := &m.messages[msgIdx]
	// Only tool messages can be collapsed
	if !msg.IsTool && msg.Type != MessageTypeTool {
		return false
	}

	// Toggle collapsed state
	msg.Collapsed = !msg.Collapsed

	// Invalidate cache for this message to force re-render
	delete(m.formatCache, msg.ID)

	// Invalidate cached content since collapse state affects line positions
	m.cachedContent = ""
	m.lineToMessage = nil

	return true
}

// isMessageSelected returns true if the message at the given index is currently selected.
func (m Model) isMessageSelected(msgIdx int) bool {
	return m.selectedMessageIndex >= 0 && msgIdx == m.selectedMessageIndex
}

// selectMessage sets the selected message index and invalidates cache for re-render.
// Pass -1 to deselect all messages.
func (m *Model) selectMessage(msgIdx int) {
	// Invalidate cache for previously selected message
	if m.selectedMessageIndex >= 0 && m.selectedMessageIndex < len(m.messages) {
		delete(m.formatCache, m.messages[m.selectedMessageIndex].ID)
	}

	// Invalidate cache for newly selected message
	if msgIdx >= 0 && msgIdx < len(m.messages) {
		delete(m.formatCache, m.messages[msgIdx].ID)
	}

	m.selectedMessageIndex = msgIdx

	// Invalidate cached content since selection affects border rendering
	m.cachedContent = ""
	// Also invalidate line positions to force recalculation
	// This ensures click detection uses correct positions after selection changes
	m.lineToMessage = nil
}

// invalidateCacheFor removes a specific message from the cache.
// This is designed for future use when messages can be updated or deleted.
// Currently not called in the main code path but kept for API completeness
// and tested in TestFormatCacheInvalidationOnMessageUpdate.
func (m *Model) invalidateCacheFor(msgID uuid.UUID) {
	delete(m.formatCache, msgID)
}

// formatMessage formats a single message for display in the viewport.
// Uses lipgloss styling to match the AgentMessenger appearance.
// For agent messages, it uses the markdown renderer if available.
//
// The message is displayed with a unified frame with 3-column header:
//
//	╭──────────┬──────────┬──────────────────────────────────────╮
//	│ 🤖 7ac5  │ Response │ 23:10:16                            │
//	├──────────┴──────────┴──────────────────────────────────────┤
//	│                                                            │
//	│ Hello! How can I help you today?                           │
//	│                                                            │
//	╰────────────────────────────────────────────────────────────╯
//
// When selected, the message uses double-line borders (╔═╗║╚╝) to indicate selection.
func (m *Model) formatMessage(msgIdx int, msg Message) string {
	// Check cache first - return cached formatted message if available
	// Cache key: message ID
	// Cache invalidation: width change, message update, selection change
	if cached, ok := m.formatCache[msg.ID]; ok && m.cacheWidth == m.width {
		return cached
	}

	// Enforce cache size limit to prevent unbounded growth
	// Check BEFORE formatting to avoid unnecessary work
	if len(m.formatCache) >= m.maxCacheSize {
		// Clear cache when limit reached (simple eviction strategy)
		// Alternative: LRU eviction would be better but more complex
		m.clearFormatCache()
	}

	// Cache miss - format the message and cache the result
	selected := m.isMessageSelected(msgIdx)
	formatted := m.formatMessageImpl(msgIdx, msg, selected)

	// Cache the formatted message
	m.formatCache[msg.ID] = formatted
	m.cacheWidth = m.width

	return formatted
}

// formatMessageImpl implements the actual message formatting logic.
// This is separated from formatMessage to enable caching.
// The selected parameter determines whether to use bold/double-line borders.
func (m *Model) formatMessageImpl(_ int, msg Message, selected bool) string {
	// For collapsed tool messages, render a compact header with click indicator
	if msg.IsTool && msg.Type == MessageTypeTool && msg.Collapsed {
		return m.formatCollapsedToolMessage(msg, selected)
	}

	timestamp := msg.Timestamp.Format("15:04:05")

	// Build header row content with 3 columns
	// Column 1: Agent/User identifier (text format, no emoji)
	// Column 2: Message type (Response, Tool, etc.)
	// Column 3: Timestamp (left-aligned as per plan)
	var col1, col2, col3 string

	switch msg.Type {
	case MessageTypeUser:
		col1 = "You"
		col2 = "User"
		col3 = timestamp

	case MessageTypeAgent, MessageTypeTool:
		agentName := formatAgentName(msg.AgentID, msg.AgentRole)
		if msg.IsTool || msg.Type == MessageTypeTool {
			col1 = fmt.Sprintf("Agent: %s", agentName)
			col2 = "Tool"
		} else {
			col1 = fmt.Sprintf("Agent: %s", agentName)
			col2 = "Response"
		}
		col3 = timestamp

	case MessageTypeSystem:
		col1 = "System"
		col2 = "Info"
		col3 = timestamp

	case MessageTypeError:
		col1 = "Error"
		col2 = "Error"
		col3 = timestamp

	default:
		col1 = "Unknown"
		col2 = "Unknown"
		col3 = timestamp
	}

	// Calculate total width and column widths
	// Total width minus margin (2 chars) to avoid edge overflow
	const minWidth = 50
	totalWidth := max(m.width-2, minWidth)

	// Column widths for the 3-column header
	// We need: totalWidth >= col1Width + col2Width + col3Width + 2
	col2Width := 10    // Message type (fixed)
	col3MinWidth := 8  // Minimum for timestamp
	col1MinWidth := 14 // Minimum for icon + agent ID

	// Calculate col3Width first (remaining space after col1 and col2)
	col3Width := totalWidth - col1MinWidth - col2Width - 2
	col3Width = max(col3Width, col3MinWidth)

	// Now calculate col1Width with remaining space
	col1Width := totalWidth - col2Width - col3Width - 2
	if col1Width < col1MinWidth {
		// If still too small, scale down proportionally
		excess := col1MinWidth - col1Width
		col1Width = col1MinWidth
		col3Width = max(col3Width-excess, col3MinWidth)
	}

	// Ensure all widths are positive
	col1Width = max(col1Width, 1)
	col2Width = max(col2Width, 1)
	col3Width = max(col3Width, 1)

	// Content width should use the same width as the border total
	// The border total is col1Width + col2Width + col3Width (for the dashes)
	// Content width = border total + 2 (for the │ on each side)
	contentWidth := col1Width + col2Width + col3Width + 2
	contentWidth = max(contentWidth, 20)

	// For agent messages, use markdown renderer if available
	var contentLines []string
	if msg.Type == MessageTypeAgent && m.markdownRenderer != nil {
		rendered, err := m.markdownRenderer.Render(context.Background(), msg.Content, contentWidth)
		if err != nil {
			contentLines = wrapText(msg.Content, contentWidth)
		} else {
			contentLines = strings.Split(rendered, "\n")
		}
	} else {
		contentLines = wrapText(msg.Content, contentWidth)
	}

	// If no content, add an empty line for spacing
	if len(contentLines) == 0 {
		contentLines = []string{""}
	}

	// Build the border string manually to get proper T-junctions
	var b strings.Builder

	// Helper to repeat a string (with safety check)
	repeat := func(s string, count int) string {
		if count <= 0 {
			return ""
		}
		return strings.Repeat(s, count)
	}

	// Select border characters based on selection state
	// Selected messages use double-line borders (bold appearance)
	var (
		topLeft, topMid, topRight string
		midLeft, midMid, midRight string
		bottomLeft, bottomRight   string
		vertical, horizontal      string
	)
	if selected {
		// Double-line borders for selected messages
		topLeft, topMid, topRight = "╔", "╦", "╗"
		midLeft, midMid, midRight = "╠", "╩", "╣"
		bottomLeft, bottomRight = "╚", "╝"
		vertical, horizontal = "║", "═"
	} else {
		// Single-line borders for normal messages
		topLeft, topMid, topRight = "╭", "┬", "╮"
		midLeft, midMid, midRight = "├", "┴", "┤"
		bottomLeft, bottomRight = "╰", "╯"
		vertical, horizontal = "│", "─"
	}

	// Top border with T-junctions for column separators
	b.WriteString(fmt.Sprintf("%s%s%s%s%s%s%s\n",
		topLeft, repeat(horizontal, col1Width), topMid, repeat(horizontal, col2Width), topMid, repeat(horizontal, col3Width), topRight))

	// Pad and truncate columns to fit
	// Adds 1 space of padding on each side of the text
	padCol := func(text string, width int, alignRight bool) string {
		// Use rune count for proper width calculation with emojis
		runes := []rune(text)
		textLen := len(runes)

		// Account for 1 space padding on each side
		availableWidth := max(width-2, 1)

		if textLen > availableWidth {
			// Truncate by runes (not bytes) to preserve emoji
			return " " + string(runes[:availableWidth]) + " "
		}

		padding := max(availableWidth-textLen, 0)

		if alignRight {
			return " " + strings.Repeat(" ", padding) + text + " "
		}
		return " " + text + strings.Repeat(" ", padding) + " "
	}

	// Header row with vertical separators
	// IMPORTANT: Apply padding first, then write the raw string with borders
	// Do NOT use lipgloss styles on the header row as they can interfere with alignment
	paddedCol1 := padCol(col1, col1Width, false)
	paddedCol2 := padCol(col2, col2Width, false)
	paddedCol3 := padCol(col3, col3Width, false) // LEFT-align timestamp (not right)

	// Write the header row with proper borders
	b.WriteString(fmt.Sprintf("%s%s%s%s%s%s%s\n", vertical, paddedCol1, vertical, paddedCol2, vertical, paddedCol3, vertical))

	// Separator line
	b.WriteString(fmt.Sprintf("%s%s%s%s%s%s%s\n",
		midLeft, repeat(horizontal, col1Width), midMid, repeat(horizontal, col2Width), midMid, repeat(horizontal, col3Width), midRight))

	// Content rows with border
	// Add 1 space of padding on each side for content
	const contentPadding = 1
	availableContentWidth := max(contentWidth-2*contentPadding, 1)

	for _, line := range contentLines {
		// Use visual width (ignoring ANSI escape codes) for proper alignment
		// This is critical for markdown-rendered content that contains color codes
		lineVisWidth := visualWidth(line)

		if lineVisWidth > availableContentWidth {
			// Truncate long lines by visual width (preserving ANSI codes at start)
			truncated := truncateVisual(line, availableContentWidth)
			b.WriteString(fmt.Sprintf("%s %s%s %s\n", vertical, truncated, strings.Repeat(" ", availableContentWidth-visualWidth(truncated)), vertical))
		} else {
			// Pad short lines with spaces on the right
			padding := availableContentWidth - lineVisWidth
			b.WriteString(fmt.Sprintf("%s %s%s %s\n", vertical, line, strings.Repeat(" ", padding), vertical))
		}
	}

	// Bottom border - needs to match top border width
	borderLineWidth := col1Width + col2Width + col3Width + 2
	b.WriteString(fmt.Sprintf("%s%s%s\n", bottomLeft, repeat(horizontal, borderLineWidth), bottomRight))

	return b.String()
}

// formatCollapsedToolMessage renders a collapsed tool message with a click-to-expand indicator.
// Shows a compact header: "⚡ Tool Output [Click to expand]" with agent name and timestamp.
// When selected, uses double-line borders to indicate selection.
func (m *Model) formatCollapsedToolMessage(msg Message, selected bool) string {
	timestamp := msg.Timestamp.Format("15:04:05")
	agentName := formatAgentName(msg.AgentID, msg.AgentRole)

	// Calculate total width matching the full message format
	const minWidth = 50
	totalWidth := max(m.width-2, minWidth)

	// Build a compact single-line header
	// Format: ⚡ Tool (AgentName) [Click to expand] · timestamp
	headerContent := fmt.Sprintf("⚡ Tool (%s) [Click to expand] · %s", agentName, timestamp)

	// Truncate if too long
	headerRunes := []rune(headerContent)
	availableWidth := totalWidth - 2 // Account for borders
	if len(headerRunes) > availableWidth {
		headerContent = string(headerRunes[:availableWidth])
	}

	// Select border characters based on selection state
	var topLeft, topRight, bottomLeft, bottomRight, vertical, horizontal string
	if selected {
		topLeft, topRight = "╔", "╗"
		bottomLeft, bottomRight = "╚", "╝"
		vertical, horizontal = "║", "═"
	} else {
		topLeft, topRight = "╭", "╮"
		bottomLeft, bottomRight = "╰", "╯"
		vertical, horizontal = "│", "─"
	}

	// Build the collapsed border
	var b strings.Builder
	borderWidth := totalWidth - 2

	// Top border
	b.WriteString(fmt.Sprintf("%s%s%s\n", topLeft, strings.Repeat(horizontal, borderWidth), topRight))

	// Header content with padding
	padding := borderWidth - len([]rune(headerContent))
	if padding < 0 {
		padding = 0
	}
	b.WriteString(fmt.Sprintf("%s %s%s %s\n", vertical, headerContent, strings.Repeat(" ", padding), vertical))

	// Bottom border (with trailing newline for consistency)
	b.WriteString(fmt.Sprintf("%s%s%s\n", bottomLeft, strings.Repeat(horizontal, borderWidth), bottomRight))

	return b.String()
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
		m.messages = []Message{}
		m.clearFormatCache()
		m.viewport.SetContent("")
		systemMsg := Message{
			ID:        uuid.New(),
			Type:      MessageTypeSystem,
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
		goodbyeMsg := Message{
			ID:        uuid.New(),
			Type:      MessageTypeSystem,
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
		helpMsg := Message{
			ID:        uuid.New(),
			Type:      MessageTypeSystem,
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
		errorMsg := Message{
			ID:        uuid.New(),
			Type:      MessageTypeError,
			Content:   fmt.Sprintf("Unknown command: %s. Type /help for available commands.", command),
			Timestamp: time.Now(),
		}
		m.addMessage(errorMsg)
		m.viewport.SetContent(m.updateViewportContent())
		m.viewport.GotoTop()
		return m, nil
	}
}

// exportMsg is sent when /export command is invoked.
type exportMsg struct{}
