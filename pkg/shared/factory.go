package shared

import (
	"context"
)

// AgentFactory creates agents in a centralized way
// This allows both AgentProvider and SpawnAgentTool to use the same logic
type AgentFactory interface {
	CreateAgent(ctx context.Context, config *AgentConfig) (Agent, error)
}
