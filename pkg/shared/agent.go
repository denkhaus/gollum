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
}
