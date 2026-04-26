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
	"time"

	"github.com/denkhaus/gollum/pkg/shared"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/markdown"
)

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
	messages []shared.Message

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
	messageChan chan shared.Message

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

	// tuiChannel is the TUIChannel instance for channel system integration
	// This allows the channel facade to send messages to the TUI
	tuiChannel *TUIChannel
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
		messages:              []shared.Message{},
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
		mouseDebounceDuration: DefaultMouseDebounceDuration,
		keyDebounceTag:        0,
		keyDebounceDuration:   DefaultKeyDebounceDuration,
		formatCache:           make(map[uuid.UUID]string),
		cacheWidth:            0, // Will be set on first update
		maxCacheSize:          DefaultMaxCacheSize,
		selectedMessageIndex:  -1, // No message selected initially
		doubleClickThreshold:  DefaultDoubleClickThreshold,
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
func (m *Model) addMessage(msg shared.Message) {
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
func (m *Model) SetMessageChannel(ch chan shared.Message) {
	m.messageChan = ch
}

// GetMessageChannel returns the message channel for AgentMessenger to send messages.
func (m Model) GetMessageChannel() chan shared.Message {
	return m.messageChan
}

// SetTUIChannel sets the TUIChannel instance for channel system integration.
// This allows the channel facade to send messages to the TUI.
func (m *Model) SetTUIChannel(ch *TUIChannel) {
	m.tuiChannel = ch
}

// GetTUIChannel returns the TUIChannel instance for channel system integration.
// This allows registering the TUI as a channel in the channel facade.
func (m Model) GetTUIChannel() *TUIChannel {
	return m.tuiChannel
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
