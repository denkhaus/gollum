# LLMClientConfig Integration Design

**Date:** 2026-03-10
**Status:** Approved
**Author:** Claude (brainstorming session)

## Overview

Introduce `LLMClientConfig` as a unified configuration structure for creating LLM clients. This replaces the previous `LLMProvider`-only approach with a more flexible "provider/model" format that allows specifying model names and optional LLM parameters at creation time.

## Motivation

1. **Unified client creation**: Flow steps and the agent factory can use the same config format
2. **Provider/model format**: Support `anthropic/claude-3-5-sonnet-20241022` style model specification
3. **Optional parameters**: Allow overriding temperature, max tokens, and top-p per-client
4. **Sane defaults**: Fall back to config service defaults when optional values are not provided

## Design

### 1. LLMClientConfig Methods (`pkg/shared/agent.go`)

Add parsing methods to `LLMClientConfig`:

```go
// Provider parses and returns the LLM provider from the Model field.
// Model must be in "provider/model" format (e.g., "anthropic/claude-3-5-sonnet-20241022").
// Returns shared.ErrInvalidModelFormat if no slash is present.
func (c *LLMClientConfig) Provider() (LLMProvider, error)

// ModelName returns just the model name portion (after the slash).
// Returns shared.ErrInvalidModelFormat if no slash is present.
func (c *LLMClientConfig) ModelName() (string, error)
```

**Error handling**: Add `ErrInvalidModelFormat` to `pkg/shared/shared.go`:
```go
ErrInvalidModelFormat = errors.New("model must be in 'provider/model' format")
```

### 2. ClientProvider.GetClient (`pkg/llm/provider.go`)

Update implementation to:

1. Parse provider and model from `cnf.Model`
2. Validate provider has API key configured
3. Use `cnf` values when provided, fall back to config service defaults

```go
func (p *clientProvider) GetClient(ctx context.Context, cnf *shared.LLMClientConfig) (gollem.LLMClient, error) {
    provider, err := cnf.Provider()
    if err != nil {
        return nil, err
    }

    modelName, err := cnf.ModelName()
    if err != nil {
        return nil, err
    }

    switch provider {
    case shared.LLMProviderAnthropic:
        cfg := p.configService.GetAnthropicConfig()
        if cfg.APIKey == "" {
            return nil, fmt.Errorf("anthropic provider not configured: missing API key")
        }

        temp := cfg.Temperature
        if cnf.Temperature != nil {
            temp = *cnf.Temperature
        }
        maxTokens := cfg.MaxTokens
        if cnf.MaxTokens != nil {
            maxTokens = *cnf.MaxTokens
        }
        topP := cfg.TopP
        if cnf.TopP != nil {
            topP = *cnf.TopP
        }

        return claude.New(ctx, cfg.APIKey,
            claude.WithModel(modelName),
            claude.WithTemperature(temp),
            claude.WithMaxTokens(int64(maxTokens)),
            claude.WithTopP(topP),
            claude.WithBaseURL(cfg.BaseURL),
        )

    case shared.LLMProviderOpenAI:
        // Similar pattern

    case shared.LLMProviderGemini:
        // Similar pattern

    default:
        return nil, shared.ErrLLMProviderNotSupported
    }
}
```

### 3. AgentConfig (`pkg/shared/shared.go`)

Replace `LLMProvider LLMProvider` field with:

```go
type AgentConfig struct {
    ID              uuid.UUID        `json:"id"`
    ParentID        *uuid.UUID       `json:"parent_id,omitempty"`
    SystemPrompt    string           `json:"system_prompt"`
    Role            string           `json:"role"`
    Description     string           `json:"description"`
    Strategy        gollem.Strategy  `json:"-"`
    Tools           []gollem.Tool    `json:"-"`
    ToolSets        []gollem.ToolSet `json:"-"`
    LLMClientConfig *LLMClientConfig `json:"llm_client_config"`
    OutputMode      OutputMode       `json:"output_mode"`
    AllowCompaction bool             `json:"allow_compaction"`
    History         *gollem.History  `json:"history,omitempty"`
}
```

### 4. AgentFactory (`pkg/agents/factory.go`)

Update `CreateAgent` to use `LLMClientConfig`:

```go
func (f *defaultAgentFactory) CreateAgent(ctx context.Context, config *shared.AgentConfig) (shared.Agent, error) {
    // ... existing validation and tool setup ...

    // Get LLM client
    client, err := f.clientProvider.GetClient(ctx, config.LLMClientConfig)
    if err != nil {
        return nil, errs.Wrap(err, errs.TypeInternal, "failed to create llm client").
            WithContext("agent_id", config.ID)
    }

    // ... rest of existing code ...
}
```

### 5. ClientProvider Interface (`pkg/llm/provider.go`)

```go
type ClientProvider interface {
    GetClient(ctx context.Context, cnf *shared.LLMClientConfig) (gollem.LLMClient, error)
}
```

## Components Affected

| File | Changes |
|------|---------|
| `pkg/shared/agent.go` | Add `Provider()` and `ModelName()` methods to `LLMClientConfig` |
| `pkg/shared/shared.go` | Add `ErrInvalidModelFormat`, replace `LLMProvider` field in `AgentConfig` |
| `pkg/llm/provider.go` | Rewrite `GetClient` to use `*shared.LLMClientConfig` |
| `pkg/agents/factory.go` | Update `CreateAgent` to pass `LLMClientConfig` |
| `pkg/app/service.go` | Update agent config construction |
| Tests | Update all test code that creates `AgentConfig` |

## Error Handling

1. **Invalid model format**: Return `ErrInvalidModelFormat` if `Model` lacks "/" separator
2. **Unknown provider**: Return `ErrLLMProviderNotSupported` for invalid provider names
3. **Missing API key**: Return descriptive error when provider not configured

## Testing Strategy

1. Unit tests for `Provider()` and `ModelName()` methods
2. Unit tests for `GetClient` with:
   - Full config (all parameters provided)
   - Partial config (some parameters nil)
   - Missing API key scenarios
   - Invalid model format
3. Integration tests for agent creation with new config format

## Migration Notes

This is a breaking change. All code constructing `AgentConfig` must be updated to provide `LLMClientConfig` instead of `LLMProvider`.
