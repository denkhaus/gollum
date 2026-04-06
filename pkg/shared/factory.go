package shared

import (
	"context"

	"github.com/google/uuid"
)

// SupervisorAgentOption is a functional option for CreateSupervisorAgent
type SupervisorAgentOption func(*AgentConfig)

// WithAgentID sets a specific agent ID instead of generating a new one
func WithAgentID(id uuid.UUID) SupervisorAgentOption {
	return func(cfg *AgentConfig) {
		cfg.ID = id
	}
}

// WithSessionID sets the session ID for the agent
func WithSessionID(sessionID string) SupervisorAgentOption {
	return func(cfg *AgentConfig) {
		cfg.SessionID = sessionID
	}
}

// WithChannelID sets the channel ID for the agent
func WithChannelID(channelID uuid.UUID) SupervisorAgentOption {
	return func(cfg *AgentConfig) {
		cfg.ChannelID = channelID
	}
}

// AgentFactory creates agents in a centralized way
// This allows both AgentProvider and SpawnAgentTool to use the same logic
type AgentFactory interface {
	CreateAgent(ctx context.Context, config *AgentConfig) (Agent, error)
	CreateSupervisorAgent(ctx context.Context, opts ...SupervisorAgentOption) (Agent, *AgentConfig, error)
}
