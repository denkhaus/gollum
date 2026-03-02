# Session Mutation API Design

## Overview

This design enables runtime modification of an agent's system prompt and message history. This is essential for:

1. **Dynamic Skill Loading** - When working directory changes, new skills are discovered and must be reflected in the system prompt immediately
2. **Observed Memory Pattern** - Background agents can compress large token-intensive messages and replace them on-the-fly

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     defaultAgent                            │
│                                                             │
│  ┌─────────────┐     ┌─────────────────────────────────┐   │
│  │ LLM Client  │     │       gollem.Agent (base)        │   │
│  │ (cached)    │────▶│  ┌───────────┐  ┌─────────────┐  │   │
│  └─────────────┘     │  │ Session   │  │ History     │  │   │
│                      │  │ (recreated)│  │ (preserved) │  │   │
│  ┌─────────────┐     │  └───────────┘  └─────────────┘  │   │
│  │ Config      │     │         ▲                        │   │
│  │ (updated)   │     └─────────│────────────────────────┘   │
│  └─────────────┘               │                            │
│                                │                            │
│  ┌─────────────────────────────▼─────────────────────────┐ │
│  │  UpdateSystemPrompt(newPrompt)                        │ │
│  │  UpdateHistory(modifier func)                         │ │
│  └───────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## Key Insight

The system prompt is **NOT** part of the history in `gollem`:
- **History** contains only `user`, `assistant`, and `tool` messages
- **System prompt** is passed separately to the LLM API on each request

This means we can:
1. Extract history from existing session
2. Create new session with NEW system prompt + OLD history
3. Old system prompt is not "baked into" the history

## Interface Changes

### pkg/shared/agent.go

```go
type Agent interface {
    // ... existing methods ...
    GetID() uuid.UUID
    GetConfig() *AgentConfig
    Session() gollem.Session
    Execute(ctx context.Context, input ...gollem.Input) (*gollem.ExecuteResponse, error)
    GetMessageHistory(ctx context.Context) (*gollem.History, error)

    // NEW: Session Mutation Methods

    // UpdateSystemPrompt replaces the system prompt immediately.
    // Blocks until the new session is created.
    // Used when directory changes and new skills are discovered.
    UpdateSystemPrompt(ctx context.Context, newPrompt string) error

    // UpdateHistory replaces parts of the history using a modifier function.
    // Non-blocking - can be called on-the-fly.
    // Used by Observed Memory Pattern for compaction.
    UpdateHistory(ctx context.Context, modifier func(*gollem.History) (*gollem.History, error)) error
}
```

## Implementation

### pkg/agents/default.go - Struct Changes

```go
type defaultAgent struct {
    base           *gollem.Agent
    logService     logger.LoggerService
    configService  config.ConfigService
    clientProvider llm.ClientProvider
    registry       registry.AgentRegistry
    id             uuid.UUID
    config         *shared.AgentConfig

    // NEW: Required for session recreation
    llmClient        gollem.LLMClient
    displayProvider  middleware.DisplayMiddlewareProvider
    summaryProvider  middleware.SummaryMiddlewareProvider
    promptManager    manager.PromptManager
}
```

### pkg/agents/default.go - New Methods

```go
// UpdateSystemPrompt replaces the system prompt immediately (blocking).
func (p *defaultAgent) UpdateSystemPrompt(ctx context.Context, newPrompt string) error {
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
        "agent_id", p.id,
        "history_messages", len(history.Messages))

    return nil
}

// UpdateHistory modifies the history using a modifier function.
func (p *defaultAgent) UpdateHistory(ctx context.Context, modifier func(*gollem.History) (*gollem.History, error)) error {
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
        "agent_id", p.id,
        "messages_before", len(currentHistory.Messages),
        "messages_after", len(modifiedHistory.Messages))

    return nil
}

// buildOptionsWithHistory consolidates options for agent recreation.
func (p *defaultAgent) buildOptionsWithHistory(history *gollem.History) []gollem.Option {
    options := []gollem.Option{
        gollem.WithStrategy(p.config.Strategy),
        gollem.WithTools(p.config.Tools...),
        gollem.WithToolSets(p.config.ToolSets...),
        gollem.WithSystemPrompt(p.config.SystemPrompt),
    }

    // Add history if present
    if history != nil && len(history.Messages) > 0 {
        options = append(options, gollem.WithHistory(history))
    }

    // Recreate middlewares based on OutputMode
    switch p.config.OutputMode {
    case shared.OutputModeSummary:
        summaryMW := p.summaryProvider.CreateSummaryMiddleware(p.id, p.config.Role)
        options = append(options,
            gollem.WithContentBlockMiddleware(summaryMW.ContentBlockMiddleware),
            gollem.WithToolMiddleware(summaryMW.ToolMiddleware),
        )
    case shared.OutputModeFull:
        displayMW := p.displayProvider.CreateDisplayMiddleware(p.id, p.config.Role)
        options = append(options,
            gollem.WithContentBlockMiddleware(displayMW.ContentBlockMiddleware),
            gollem.WithToolMiddleware(displayMW.ToolMiddleware),
        )
    // OutputModeSilent: no middlewares
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
```

### pkg/agents/factory.go - Updated Creation

```go
func (f *defaultAgentFactory) CreateAgent(ctx context.Context, config *shared.AgentConfig) (shared.Agent, error) {
    // ... existing validation and tool setup ...

    // Get LLM client
    client, err := f.clientProvider.GetClient(ctx, config.LLMProvider)
    if err != nil {
        return nil, errs.Wrap(err, errs.TypeInternal, "failed to create llm client").
            WithContext("agent_id", config.ID).
            WithContext("llm_provider", config.LLMProvider)
    }

    defAgent := &defaultAgent{
        logService:       f.logService,
        configService:    f.configService,
        clientProvider:   f.clientProvider,
        registry:         f.registry,
        id:               config.ID,
        config:           config,
        llmClient:        client,                 // NEW: Cache for recreation
        displayProvider:  f.displayProvider,      // NEW: For middleware recreation
        summaryProvider:  f.summaryProvider,      // NEW: For middleware recreation
        promptManager:    f.promptManager,        // NEW: For compacter recreation
    }

    // ... rest of existing code ...

    return defAgent, nil
}
```

## Usage Examples

### Updating System Prompt on Directory Change

```go
// When new skills are discovered after directory change
func (s *AppService) onDirectoryChange(newDir string) error {
    // Discover new skills
    err := s.skillService.AddSearchPathAndDiscover(ctx, newDir)
    if err != nil {
        return err
    }

    // Build new system prompt with updated skills
    newPrompt, err := s.promptManager.GetSystemPrompt(ctx, skillList)
    if err != nil {
        return err
    }

    // Update supervisor agent immediately
    return s.supervisorAgent.UpdateSystemPrompt(ctx, newPrompt)
}
```

### Observed Memory Pattern (Future)

```go
// Background agent compresses large messages
func (p *MemoryObserver) compressAndReplace(agent shared.Agent, largeMsgIndex int) error {
    return agent.UpdateHistory(ctx, func(h *gollem.History) (*gollem.History, error) {
        // Generate summary of large message
        summary := p.generateSummary(h.Messages[largeMsgIndex])

        // Replace with compressed version
        h.Messages[largeMsgIndex] = gollem.Message{
            Role: gollem.RoleAssistant,
            Contents: []gollem.MessageContent{
                // ... summary content ...
            },
        }

        return h, nil
    })
}
```

## Implementation Checklist

- [ ] Add new methods to `pkg/shared/agent.go` interface
- [ ] Add new fields to `pkg/agents/default.go` struct
- [ ] Implement `UpdateSystemPrompt()` in `defaultAgent`
- [ ] Implement `UpdateHistory()` in `defaultAgent`
- [ ] Implement `buildOptionsWithHistory()` helper in `defaultAgent`
- [ ] Update `pkg/agents/factory.go` to populate new fields
- [ ] Add unit tests for both methods
- [ ] Add integration test with directory change scenario

## Notes

- Middlewares are stateless and only need agent ID and role, so they can be recreated safely
- The system prompt is passed separately to the LLM API, not stored in history
- History only contains `user`, `assistant`, and `tool` messages
