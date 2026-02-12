// Package agents provides agent implementations and factory functions.
package agents

import (
	"context"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/llm"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
)

type (
	defaultAgent struct {
		base           *gollem.Agent
		logService     logger.LoggerService
		configService  config.ConfigService
		clientProvider llm.ClientProvider
		registry       registry.AgentRegistry
		id             uuid.UUID
		config         *shared.AgentConfig
	}
)

func (p *defaultAgent) Execute(ctx context.Context, input ...gollem.Input) (*gollem.ExecuteResponse, error) {
	return p.base.Execute(ctx, input...)
}

func (p *defaultAgent) Session() gollem.Session {
	return p.base.Session()
}

func (p *defaultAgent) GetID() uuid.UUID {
	return p.id
}

func (p *defaultAgent) GetConfig() *shared.AgentConfig {
	return p.config
}

// GetMessageHistory retrieves the agent's message history from its session
func (p *defaultAgent) GetMessageHistory(ctx context.Context) (*gollem.History, error) {
	// Handle nil base agent
	if p.base == nil {
		return nil, nil
	}

	session := p.base.Session()
	if session == nil {
		return nil, nil
	}

	history, err := session.History()
	if err != nil {
		return nil, err
	}

	return history, nil
}
