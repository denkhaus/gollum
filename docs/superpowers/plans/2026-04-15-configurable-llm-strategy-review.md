# Plan Review: Configurable LLM Strategy Implementation

**Date:** 2026-04-15
**Reviewer:** Claude (Code Review)
**Status:** Needs Revisions

## Executive Summary

The plan addresses a critical infinite loop issue with the `simple` strategy by migrating to `react` strategy with configurable parameters. However, several gaps and inconsistencies with the actual codebase need to be addressed before implementation.

## Critical Gaps

### 1. Architecture Mismatch: Two-Tier Executor Pattern

**Issue:** The plan assumes a single `flowExecutorImpl` struct, but the actual codebase has a **two-tier service pattern**:

- `flowExecutorServiceImpl` - DI service that creates instances
- `flowExecutorImpl` - Per-flow execution instance

**Impact:** The plan's Chunk 6 incorrectly targets `flowExecutorImpl` for DI injection. StrategyBuilder must be injected into `flowExecutorServiceImpl` and passed to instances.

**Required Change:**

```go
// CORRECT: Inject into service
type flowExecutorServiceImpl struct {
    logService        logger.LoggerService
    bashToolProvider  tools.BashToolProvider
    extService        extensions.ExtensionService
    flowRegistry      flowregistry.FlowRegistry
    hookManager       hooks.HookManager
    flowToolsProvider tools.FlowToolsProvider
    mcpRegistry       mcpregistry.MCPRegistry
    agentFactory      shared.AgentFactory
    strategyBuilder   shared.StrategyBuilder  // NEW
}

// Pass to instances during creation
func (p *flowExecutorServiceImpl) New(flow *flows.Flow) FlowExecutorInstance {
    return &flowExecutorImpl{
        // ...
        strategyBuilder: p.strategyBuilder,  // Pass through
    }
}
```

### 2. Missing DI Registration Step

**Gap:** The plan never registers `StrategyBuilder` in the DI container.

**Required Addition:** Add to Chunk 3:

```go
// In pkg/di/container.go RegisterServices()
do.Provide(p.injector, shared.NewStrategyBuilder)
```

### 3. Config Service Interface Not Extended

**Gap:** The plan adds methods to `ConfigService` interface but doesn't specify where.

**Clarification Needed:** The getters should be added after line 291 in `pkg/config/service.go`:

```go
type ConfigService interface {
    // ... existing methods ...
    GetSubAgentConfig() *SubAgentConfig  // NEW
    GetSupervisorConfig() *SupervisorConfig  // NEW
}
```

### 4. Shared Package StrategyConfig Duplication

**Issue:** The plan defines `StrategyConfig` in both `pkg/config` AND `pkg/shared/strategy.go`, causing duplication and potential circular dependency.

**Better Approach:** Keep the canonical definition in `pkg/config` and import it in `shared`:

```go
// pkg/shared/strategy.go
package shared

import (
    "github.com/denkhaus/gollum/pkg/config"
    "github.com/m-mizutani/gollem/strategy/react"
)

type StrategyBuilder interface {
    BuildReact(cfg *config.StrategyConfig) *react.Strategy
    BuildDefaultReact() *react.Strategy
}
```

### 5. Missing React Strategy Import Verification

**Gap:** The plan assumes `github.com/m-mizutani/gollem/strategy/react` exists and has specific API (`WithMaxIterations`, `WithMaxRepeatedActions`).

**Required Verification Step:** Before implementation, verify:

```bash
# Check if react strategy exists and has expected API
go doc github.com/m-mizutani/gollem/strategy/react
go doc github.com/m-mizutani/gollem/strategy/react.WithMaxIterations
```

## Design Concerns (Karpathy Guidelines)

### 1. Over-Engineering Risk: Three Configuration Layers

**Concern:** The plan introduces THREE separate configuration mechanisms:
1. Flow XML `<strategy>` element
2. Env vars for subagents (`GOLLUM_SUBAGENT_STRATEGY_*`)
3. Env vars for supervisors (`GOLLUM_SUPERVISOR_STRATEGY_*`)

**Question:** Is this complexity necessary? Could a single default configuration suffice?

**Simplification Proposal:** Consider a single global default with optional per-agent override:

```go
// Simplified approach
type StrategyConfig struct {
    MaxIterations      int `envconfig:"STRATEGY_MAX_ITERATIONS" default:"20"`
    MaxRepeatedActions int `envconfig:"STRATEGY_MAX_REPEATED_ACTIONS" default:"3"`
}
```

### 2. YAGNI Violation: Separate SubAgentConfig/SupervisorConfig

**Issue:** The plan creates separate `SubAgentConfig` and `SupervisorConfig` structs, but both contain identical `Strategy` fields.

**Simplification:** Use a single `StrategyConfig` that applies to all agents unless overridden in flow XML:

```go
type serviceImpl struct {
    // ...
    Strategy StrategyConfig `envconfig:"STRATEGY"`
}

func (s *serviceImpl) GetStrategyConfig() *StrategyConfig {
    return &s.Strategy
}
```

### 3. Missing Error Handling for Zero Values

**Gap:** The plan says "zero values handled gracefully" but doesn't specify how.

**Required Specification:** Document behavior when MaxIterations=0 or MaxRepeatedActions=0:

- Option A: Treat zero as "use default"
- Option B: Treat zero as "no limit" (dangerous!)
- Option C: Return validation error

**Recommendation:** Option A for safety.

## Missing Implementation Details

### 1. AgentConfig.Strategy Field Type

**Gap:** The plan modifies `AgentConfig` to use strategy but doesn't specify the type.

**Clarification Needed:**

```go
// In shared.AgentConfig
type AgentConfig struct {
    // ...
    Strategy gollem.Strategy  // Interface from gollem package
}
```

### 2. Flow Agent Model vs LLM Field Mismatch

**Issue:** The plan references `<llm model="..." />` in flow XML, but the current `Agent` struct uses a `Model` attribute directly.

**Current Code:**
```go
type Agent struct {
    Name        string  `xml:"name,attr"`
    Model       string  `xml:"model,attr"`  // Direct attribute
    Prompt      string  `xml:"prompt"`
    // ...
}
```

**Plan's Example:**
```xml
<agent name="summarizer">
    <strategy maxIterations="10" />
    <llm model="anthropic/opus-4.6" />  <!-- This doesn't match -->
</agent>
```

**Resolution:** Either update the struct to support nested `<llm>` element OR update examples to use direct attribute.

### 3. Test Setup Patterns Undefined

**Gap:** The plan references `setupTestInjector(t)` but doesn't define the pattern.

**Required:** Add a test helpers section showing how to set up DI for testing:

```go
// pkg/shared/factory_test.go or test_helpers.go
func setupTestInjector(t *testing.T) do.Injector {
    injector := do.New()
    do.Provide(injector, config.NewService)
    do.Provide(injector, shared.NewStrategyBuilder)
    // ... minimal providers for test
    return injector
}
```

## Testing Gaps

### 1. No Integration Test for Actual Infinite Loop Prevention

**Gap:** The plan has a stub test marked `t.Skip("Requires full setup")` but doesn't provide a path to enable it.

**Required:** Either:
- Remove the stub test if not feasible
- OR provide a complete integration test with mock LLM

### 2. Missing Negative Test Cases

**Gap:** No tests for:
- Invalid env var values (negative numbers, non-integers)
- Missing react strategy package (fallback behavior)
- Zero value handling

### 3. No Test for Strategy Override Priority

**Gap:** When both env vars AND flow XML specify strategy, which wins?

**Required Specification:** Document priority order:
1. Flow XML `<strategy>` (highest priority)
2. Environment variables
3. Hardcoded defaults (lowest priority)

## Documentation Gaps

### 1. No User Migration Guide

**Gap:** The plan updates `llm-step-limitations.md` but doesn't provide user-facing documentation.

**Required:** Add a user guide showing:
- How to add `<strategy>` to existing flows
- Recommended values for different use cases
- Behavior changes from simple→react migration

### 2. Missing Performance Impact Analysis

**Gap:** The `react` strategy has different performance characteristics than `simple`.

**Question:** Will MaxIterations=20 cause premature termination for legitimate multi-step tasks?

**Required:** Document trade-offs and provide guidance on choosing values.

## Recommended Improvements

### 1. Simplify Configuration Architecture

**Before Implementation:** Decide between:
- **Option A (Current Plan):** Separate configs for subagents/supervisors
- **Option B (Simpler):** Single global config with per-agent XML override

### 2. Add Configuration Validation

**Add to Chunk 1:**

```go
func (c *StrategyConfig) Validate() error {
    if c.MaxIterations < 0 {
        return fmt.Errorf("MAX_ITERATIONS must be >= 0, got %d", c.MaxIterations)
    }
    if c.MaxRepeatedActions < 0 {
        return fmt.Errorf("MAX_REPEATED_ACTIONS must be >= 0, got %d", c.MaxRepeatedActions)
    }
    if c.MaxIterations == 0 && c.MaxRepeatedActions == 0 {
        return fmt.Errorf("at least one limit must be set")
    }
    return nil
}
```

### 3. Explicit Breaking Changes Section

**Add to Summary:**

```markdown
## Breaking Changes

1. **Default Behavior Change:** All agents now use `react` strategy instead of `simple`
   - Impact: Agents will stop after MaxIterations even if making progress
   - Mitigation: Set higher MaxIterations if needed

2. **Flow XML Changes:** Flows can now include `<strategy>` elements
   - Impact: None (backwards compatible)
   - Benefit: Per-agent customization

3. **Environment Variables:** New GOLLUM_STRATEGY_* vars
   - Impact: None (defaults provided)
   - Migration: Optional, for customization only
```

## Pre-Implementation Checklist

Before executing this plan:

- [ ] Verify `github.com/m-mizutani/gollem/strategy/react` exists and has expected API
- [ ] Decide on simplified vs. complex configuration architecture
- [ ] Document priority order for conflicting config sources
- [ ] Specify zero value handling behavior
- [ ] Define approach for integration tests (mock vs. real)
- [ ] Create user-facing migration guide
- [ ] Analyze performance impact of react vs. simple strategy

## Conclusion

The plan addresses a critical bug but needs revisions for:
1. Architecture alignment with two-tier executor pattern
2. Simplification of configuration layer
3. Completion of missing test and documentation artifacts
4. Clarification of ambiguous design decisions

**Recommendation:** Address critical gaps (especially #1 and #2) before implementation. Consider simplification proposal for configuration architecture.
