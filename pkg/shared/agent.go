// Package shared provides common types and interfaces used across the Gollum agent system.
package shared

import (
	"context"

	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
)

// Agent interface to avoid import cycles
// This interface contains only the methods that registry needs to know about
type Agent interface {
	GetID() uuid.UUID
	GetConfig() *AgentConfig
	Session() gollem.Session
	Execute(ctx context.Context, input ...gollem.Input) (*gollem.ExecuteResponse, error)
	// GetMessageHistory retrieves the agent's message history
	GetMessageHistory(ctx context.Context) (*gollem.History, error)

	// UpdateSystemPrompt replaces the system prompt immediately.
	// Blocks until the new session is created.
	// Used when directory changes and new skills are discovered.
	UpdateSystemPrompt(ctx context.Context, newPrompt string) error

	// UpdateHistory replaces parts of the history using a modifier function.
	// Non-blocking - can be called on-the-fly.
	// Used by Observed Memory Pattern for compaction.
	UpdateHistory(ctx context.Context, modifier func(*gollem.History) (*gollem.History, error)) error
}
