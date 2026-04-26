// Package shared provides common types and interfaces used across the Gollum agent system.
package shared

import (
	"context"
	"strings"

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

	// ToSessionContext creates a SessionContext from the agent's configuration.
	// This provides session, channel, and agent context for unified logging.
	ToSessionContext() SessionContext
}

// LLMClientConfig holds configuration for LLM client initialization
type LLMClientConfig struct {
	// Model describes a llm-provider/model combination in the format "<provider>/model"
	Model string
	// Temperature for model initialization (optional)
	Temperature *float64
	// MaxTokens for model initialization (optional)
	MaxTokens *int
	// TopP for model initialization (optional)
	TopP *float64
}

// Provider parses and returns the LLM provider from the Model field.
// Model must be in "provider/model" format (e.g., "anthropic/claude-3-5-sonnet-20241022").
// Returns ErrInvalidModelFormat if no slash is present or if slash is at the start/end.
// Returns ErrLLMProviderNotSupported if the provider is not recognized.
func (c *LLMClientConfig) Provider() (LLMProvider, error) {
	idx := strings.Index(c.Model, "/")
	if idx == -1 || idx == 0 || idx == len(c.Model)-1 {
		return "", ErrInvalidModelFormat
	}
	providerStr := c.Model[:idx]
	switch LLMProvider(providerStr) {
	case LLMProviderAnthropic, LLMProviderOpenAI, LLMProviderGemini:
		return LLMProvider(providerStr), nil
	default:
		return "", ErrLLMProviderNotSupported
	}
}

// ModelName returns just the model name portion (after the slash).
// Returns ErrInvalidModelFormat if format is invalid (no slash, or slash at start/end).
func (c *LLMClientConfig) ModelName() (string, error) {
	idx := strings.Index(c.Model, "/")
	if idx == -1 || idx == 0 || idx == len(c.Model)-1 {
		return "", ErrInvalidModelFormat
	}
	return c.Model[idx+1:], nil
}
