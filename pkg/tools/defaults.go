// Package tools provides default values and constants for tool implementations.
package tools

import "time"

// Timeout constants for Bash tool
const (
	// DefaultBashTimeout is the default timeout for bash commands in seconds.
	DefaultBashTimeout = 60.0

	// MaxBashTimeout is the maximum allowed timeout for bash commands in seconds (5 minutes).
	MaxBashTimeout = 300.0
)

// Timeout constants for AgentOutput tool
const (
	// DefaultAgentOutputTimeout is the default timeout for waiting on agent output in milliseconds.
	DefaultAgentOutputTimeout = 30000

	// MaxAgentOutputTimeout is the maximum allowed timeout for agent output in milliseconds (10 minutes).
	MaxAgentOutputTimeout = 600000

	// MinAgentOutputTimeout is the minimum allowed timeout for agent output in milliseconds.
	MinAgentOutputTimeout = 1
)

// Default limits for file operations
const (
	// DefaultReadFileLimit is the default maximum number of lines to read.
	DefaultReadFileLimit = 200

	// DefaultReadFileOffset is the default line number to start reading from.
	DefaultReadFileOffset = 1
)

// Default limits for session logs
const (
	// DefaultSessionLogCount is the default number of log entries to return.
	DefaultSessionLogCount = 100
)

// Timeout durations (pre-converted for convenience)
var (
	// DefaultAgentOutputTimeoutDuration is the default timeout as a time.Duration.
	DefaultAgentOutputTimeoutDuration = time.Duration(DefaultAgentOutputTimeout) * time.Millisecond

	// MaxAgentOutputTimeoutDuration is the maximum timeout as a time.Duration.
	MaxAgentOutputTimeoutDuration = time.Duration(MaxAgentOutputTimeout) * time.Millisecond
)
