# Configurable LLM Strategy Design

**Date:** 2026-04-15
**Author:** Claude (Planning)
**Status:** Approved

## Problem Statement

The `simple` strategy used by default for LLM steps, subagents, and supervisors has a critical flaw: it continues as long as there is input, with no maximum iteration limit. This causes infinite loops when the LLM uses tools like `emit_log`.

**Root Cause:** The `simple` strategy only terminates when the LLM makes no tool calls. If the LLM keeps calling tools, the loop never ends.

## Solution

Replace the `simple` strategy with `react` strategy and make its parameters configurable via:
1. **Flow YAML** — For LLM step agents (per-agent configuration)
2. **Environment Variables** — For subagents and supervisors (global defaults)

## Architecture

### Configuration Layer (`pkg/config/service.go`)

New config structs:

```go
// StrategyConfig defines iteration limits for LLM agent strategies
type StrategyConfig struct {
    MaxIterations      int `envconfig:"MAX_ITERATIONS" default:"20"`
    MaxRepeatedActions int `envconfig:"MAX_REPEATED_ACTIONS" default:"3"`
}

// SubAgentConfig holds subagent-specific configuration
type SubAgentConfig struct {
    Strategy StrategyConfig `envconfig:"STRATEGY"`
}

// SupervisorConfig holds supervisor-specific configuration  
type SupervisorConfig struct {
    Strategy StrategyConfig `envconfig:"STRATEGY"`
}
```

Environment variables:
- `GOLLUM_SUBAGENT_STRATEGY_MAX_ITERATIONS` (default: 20)
- `GOLLUM_SUBAGENT_STRATEGY_MAX_REPEATED_ACTIONS` (default: 3)
- `GOLLUM_SUPERVISOR_STRATEGY_MAX_ITERATIONS` (default: 20)
- `GOLLUM_SUPERVISOR_STRATEGY_MAX_REPEATED_ACTIONS` (default: 3)

### Shared Strategy Builder (`pkg/shared/`)

**Interface** (`pkg/shared/strategy.go`):

```go
package shared

import "github.com/m-mizutani/gollem/strategy/react"

// StrategyBuilder provides strategy construction as a DI service
type StrategyBuilder interface {
    BuildReact(cfg *config.StrategyConfig) *react.Strategy
    BuildDefaultReact() *react.Strategy
}
```

**Implementation** (`pkg/shared/strategy_builder.go`):

```go
package shared

import (
    "github.com/denkhaus/gollum/pkg/config"
    "github.com/m-mizutani/gollem/strategy/react"
    "github.com/samber/do/v2"
)

type strategyBuilderImpl struct{}

func NewStrategyBuilder(_ do.Injector) (StrategyBuilder, error) {
    return &strategyBuilderImpl{}, nil
}

func (b *strategyBuilderImpl) BuildReact(cfg *config.StrategyConfig) *react.Strategy {
    opts := []react.Option{}
    
    if cfg.MaxIterations > 0 {
        opts = append(opts, react.WithMaxIterations(cfg.MaxIterations))
    }
    if cfg.MaxRepeatedActions > 0 {
        opts = append(opts, react.WithMaxRepeatedActions(cfg.MaxRepeatedActions))
    }
    
    return react.New(opts...)
}

func (b *strategyBuilderImpl) BuildDefaultReact() *react.Strategy {
    return b.BuildReact(&config.StrategyConfig{
        MaxIterations:      20,
        MaxRepeatedActions: 3,
    })
}
```

### Agent Factory Integration (`pkg/agents/factory.go`)

1. Add `strategyBuilder shared.StrategyBuilder` field to `defaultAgentFactory`
2. Inject via DI in `NewAgentFactory`
3. Update `CreateAgent` to use `BuildDefaultReact()` when `config.Strategy == nil`
4. Update `CreateSupervisorAgent` to use supervisor config from config service

### Flow Types (`pkg/flows/types.go`)

```go
// Agent defines an LLM agent configuration within a flow
type Agent struct {
    Name     string         `xml:"name,attr"`
    Prompt   string         `xml:"prompt,attr"`
    LLM      AgentLLM       `xml:"llm"`
    Strategy *AgentStrategy `xml:"strategy"` // NEW
}

// AgentStrategy defines strategy parameters for flow agents
type AgentStrategy struct {
    MaxIterations      int `xml:"maxIterations,attr"`
    MaxRepeatedActions int `xml:"maxRepeatedActions,attr"`
}
```

### Flow Executor (`pkg/flows/executor/llm_step.go`)

1. Add `strategyBuilder shared.StrategyBuilder` to `flowExecutorImpl`
2. In `executeLLMStep`, check if `agentConfig.Strategy` is defined
3. If defined, use `BuildReact()` with values from YAML
4. If not defined, use `BuildDefaultReact()`
5. Pass the constructed strategy to `shared.AgentConfig` instead of `simple.New()`

### Flow YAML Syntax

```xml
<agents>
    <!-- Agent with custom strategy -->
    <agent name="summarizer">
        <strategy maxIterations="10" maxRepeatedActions="2" />
        <llm model="anthropic/opus-4.6" />
    </agent>
    
    <!-- Agent with default strategy -->
    <agent name="analyst">
        <llm model="anthropic/sonnet-4.6" />
    </agent>
</agents>
```

## Data Flow

### For Subagents/Supervisors:
1. Config service reads env vars on startup
2. Agent factory injects StrategyBuilder via DI
3. `CreateSupervisorAgent` gets config from ConfigService
4. StrategyBuilder constructs react strategy with config values
5. Agent created with configured strategy

### For LLM Steps:
1. Flow XML parsed into `flows.Agent` structs
2. Flow executor reads agent config
3. If `<strategy>` element present, use those values
4. If absent, use StrategyBuilder defaults
5. Agent created with appropriate strategy

## Error Handling

- Invalid env vars: Use defaults (validation happens at config layer)
- Invalid XML values: Will be zero, handled gracefully by BuildReact (skips zero values)
- Missing config: StrategyBuilder always returns valid react strategy with defaults

## Testing Plan

### Unit Tests
1. **Config Service**: Verify env var parsing and defaults
2. **Strategy Builder**: Test BuildReact with various configs
3. **Agent Factory**: Test strategy integration for subagents/supervisors
4. **Flow Executor**: Test YAML parsing and strategy selection

### Integration Test
- Create flow with agents using different strategy configurations
- Verify tools work without infinite loops
- Verify iteration limits are respected

## Migration Notes

- **Breaking Change**: Flows that relied on `simple` strategy will now use `react` with defaults
- **Backwards Compatible**: Existing flows without `<strategy>` element will use react defaults
- **Upgrade Path**: Users can add `<strategy>` elements to flows to customize behavior
