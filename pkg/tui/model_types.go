// Package tui provides Bubbletea-based terminal user interface components for Gollum.
//
// This file contains type definitions and message types for the TUI.
package tui

import (
	"context"
	"time"

	"github.com/denkhaus/gollum/pkg/shared"

	"github.com/m-mizutani/gollem"
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

// newMessageMsg is sent when a new message should be added to the viewport.
type newMessageMsg struct {
	message shared.Message
}

// exportMsg is sent when /export command is invoked.
type exportMsg struct{}

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

// TUI timing and sizing constants
const (
	// DefaultMouseDebounceDuration is the default debounce time for mouse wheel events.
	DefaultMouseDebounceDuration = 130 * time.Millisecond

	// DefaultKeyDebounceDuration is the default debounce time for arrow key events.
	DefaultKeyDebounceDuration = 16 * time.Millisecond

	// DefaultDoubleClickThreshold is the default time window for double-click detection.
	DefaultDoubleClickThreshold = 500 * time.Millisecond

	// DefaultMaxCacheSize is the default maximum number of cached formatted messages.
	DefaultMaxCacheSize = 1000
)
