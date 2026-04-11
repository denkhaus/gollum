// Package agents provides agent implementations and factory functions.
package agents

import (
	"context"

	"github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/errs"
	"github.com/denkhaus/gollum/pkg/llm"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/prompt"
	"github.com/denkhaus/gollum/pkg/prompt/manager"
	"github.com/denkhaus/gollum/pkg/registry"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/middleware/compacter"
	"go.uber.org/zap"
)

type (
	// DefaultAgent is the default implementation of shared.Agent
	// It can be type-asserted from shared.Agent to access additional methods like AddTools
	DefaultAgent struct {
		base           *gollem.Agent
		logService     logger.LoggerService
		configService  config.ConfigService
		clientProvider llm.ClientProvider
		registry       registry.AgentRegistry
		id             uuid.UUID
		config         *shared.AgentConfig

		// Required for session recreation
		llmClient     gollem.LLMClient
		promptManager manager.PromptManager
		tools         []gollem.Tool // Resolved tools for agent recreation
	}
)

func (p *DefaultAgent) Execute(ctx context.Context, input ...gollem.Input) (*gollem.ExecuteResponse, error) {
	return p.base.Execute(ctx, input...)
}

func (p *DefaultAgent) Session() gollem.Session {
	return p.base.Session()
}

func (p *DefaultAgent) GetID() uuid.UUID {
	return p.id
}

func (p *DefaultAgent) GetConfig() *shared.AgentConfig {
	return p.config
}

// ToLoggingContext creates a LoggingContext from the agent's configuration.
// This provides session, channel, and agent context for unified logging.
func (p *DefaultAgent) ToLoggingContext() shared.LoggingContext {
	return shared.LoggingContext{
		SessionID: p.config.SessionID,
		ChannelID: p.config.ChannelID,
		AgentID:   p.id,
	}
}

// GetMessageHistory retrieves the agent's message history from its session
func (p *DefaultAgent) GetMessageHistory(ctx context.Context) (*gollem.History, error) {
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

// UpdateSystemPrompt replaces the system prompt immediately (blocking).
// It preserves the existing message history while updating the system prompt.
func (p *DefaultAgent) UpdateSystemPrompt(ctx context.Context, newPrompt string) error {
	// 1. Preserve existing history
	history, err := p.GetMessageHistory(ctx)
	if err != nil {
		return errs.Wrap(err, errs.TypeInternal, "failed to get message history").
			WithContext("agent_id", p.id)
	}

	// 2. Update config with new prompt
	p.config.SystemPrompt = newPrompt

	// 3. Build new options with preserved history
	newOptions := p.buildOptionsWithHistory(history)

	// 4. Create new agent (blocking)
	p.base = gollem.New(p.llmClient, newOptions...)

	p.logService.GetLogger().Info("System prompt updated",
		zap.String("agent_id", p.id.String()),
		zap.Int("history_messages", len(history.Messages)))

	return nil
}

// UpdateHistory modifies the history using a modifier function.
// This allows flexible transformations like summarization, filtering, or replacement.
func (p *DefaultAgent) UpdateHistory(ctx context.Context, modifier func(*gollem.History) (*gollem.History, error)) error {
	// 1. Get current history
	currentHistory, err := p.GetMessageHistory(ctx)
	if err != nil {
		return errs.Wrap(err, errs.TypeInternal, "failed to get message history").
			WithContext("agent_id", p.id)
	}

	// 2. Apply modifier (transformation)
	modifiedHistory, err := modifier(currentHistory)
	if err != nil {
		return errs.Wrap(err, errs.TypeInternal, "history modifier failed").
			WithContext("agent_id", p.id)
	}

	// 3. Recreate agent with modified history
	newOptions := p.buildOptionsWithHistory(modifiedHistory)
	p.base = gollem.New(p.llmClient, newOptions...)

	p.logService.GetLogger().Info("History updated",
		zap.String("agent_id", p.id.String()),
		zap.Int("messages_before", len(currentHistory.Messages)),
		zap.Int("messages_after", len(modifiedHistory.Messages)))

	return nil
}

// buildOptionsWithHistory consolidates options for agent recreation.
// It rebuilds all gollem options including middlewares based on current config.
func (p *DefaultAgent) buildOptionsWithHistory(history *gollem.History) []gollem.Option {
	options := []gollem.Option{
		gollem.WithStrategy(p.config.Strategy),
		gollem.WithSystemPrompt(p.config.SystemPrompt),
	}

	// Add tools if they exist
	if len(p.tools) > 0 {
		options = append(options, gollem.WithTools(p.tools...))
	}

	// Add history if present
	if history != nil && len(history.Messages) > 0 {
		options = append(options, gollem.WithHistory(history))
	}

	// Add compacter middleware if enabled
	if p.config.AllowCompaction {
		compacterPrompt, err := p.promptManager.GetPromptByID(context.Background(), prompt.PromptIDCompacter)
		if err == nil {
			contextCompacter := compacter.NewContentBlockMiddleware(p.llmClient,
				compacter.WithSummaryPrompt(compacterPrompt.Content),
			)
			options = append(options,
				gollem.WithContentBlockMiddleware(contextCompacter),
			)
		}
	}

	return options
}

// AddTools adds additional tools to the agent's tool set.
// This is useful for adding flow-specific tools after agent creation.
func (p *DefaultAgent) AddTools(additionalTools []gollem.Tool) {
	p.tools = append(p.tools, additionalTools...)

	// Recreate agent with updated tools
	newOptions := p.buildOptionsWithHistory(nil)
	p.base = gollem.New(p.llmClient, newOptions...)
}
