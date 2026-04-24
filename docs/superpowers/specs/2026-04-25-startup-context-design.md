# Startup Context Design

**Date:** 2026-04-25
**Status:** Design Approved
**Author:** Claude & denkhaus

## Problem Statement

The startup flow executes when the application starts, but its output is currently unused. The startup flow could act as a powerful pre-processor that prepares context for the main agent, enriching its understanding of the workspace and project.

## Solution

Introduce a `StartupContextService` that captures the startup flow's natural language output and automatically injects it into the main agent's system prompt via `PromptManager`.

## Architecture

```
┌─────────────────────────┐
│  ApplicationService     │
│  .Run()                 │
└────────┬────────────────┘
         │
         │ 1. Execute startup flow
         ▼
┌─────────────────────────┐
│  StartupContextService  │
│  .SetContext(outputs)   │
└────────┬────────────────┘
         │ (singleton via DI)
         │
         │ 2. PromptManager reads context
         ▼
┌─────────────────────────┐
│  PromptManager          │
│  .BuildSystemPrompt()   │
│  reads startup context  │
└─────────────────────────┘
```

## Components

### StartupContextService (New)

```go
// pkg/startup/service.go
package startup

import (
    "fmt"
    "sync"
)

// StartupContextService stores and provides startup flow context
type StartupContextService interface {
    // SetContext stores the startup flow outputs
    SetContext(outputs map[string]any)

    // GetContextText returns formatted context for prompts
    GetContextText() string

    // HasContent returns true if startup context is available
    HasContent() bool

    // Clear removes stored context (for testing)
    Clear()
}

type startupContextServiceImpl struct {
    outputs map[string]any
    mu      sync.RWMutex
}

func NewStartupContextService() StartupContextService {
    return &startupContextServiceImpl{
        outputs: make(map[string]any),
    }
}

func (s *startupContextServiceImpl) SetContext(outputs map[string]any) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.outputs = outputs
}

func (s *startupContextServiceImpl) GetContextText() string {
    s.mu.RLock()
    defer s.mu.RUnlock()

    if len(s.outputs) == 0 {
        return ""
    }

    // If there's a dedicated "context" or "text" field, use it directly
    if text, ok := s.outputs["context"].(string); ok {
        return text
    }
    if text, ok := s.outputs["text"].(string); ok {
        return text
    }

    // Otherwise, format all key-value pairs
    return s.formatOutputs()
}

func (s *startupContextServiceImpl) HasContent() bool {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return len(s.outputs) > 0
}

func (s *startupContextServiceImpl) Clear() {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.outputs = make(map[string]any)
}

func (s *startupContextServiceImpl) formatOutputs() string {
    var result string
    for key, value := range s.outputs {
        result += fmt.Sprintf("%s: %v\n", key, value)
    }
    return result
}
```

### ApplicationService Changes

```go
// pkg/app/service.go

type applicationServiceImpl struct {
    // ... existing fields ...
    startupContextService startup.StartupContextService
}

func (p *applicationServiceImpl) Run(ctx context.Context) error {
    // ... existing initialization ...

    // Execute startup flow
    startupFlow, err := p.flowRegistry.GetStartupFlow()
    if err == nil && startupFlow != nil {
        p.logService.Info("Startup flow found, executing...")

        result, execErr := p.flowExecutorService.Execute(ctx, startupFlow, nil)
        if execErr != nil {
            p.logService.Errorf("Startup flow failed: %v", execErr)
            // Continue anyway - startup flow failure should not block the app
        } else if result != nil && len(result.Outputs) > 0 {
            // Store outputs in StartupContextService
            p.startupContextService.SetContext(result.Outputs)
            p.logService.Info("Startup flow completed successfully, context stored")
        }
    }

    // Run TUI or other channels
    return p.runChannel(ctx, tui.Identifier)
}
```

### PromptManager Changes

```go
// pkg/prompts/manager.go

type promptManagerImpl struct {
    // ... existing fields ...
    startupContextService startup.StartupContextService
}

func (p *promptManagerImpl) BuildSystemPrompt(basePrompt string, ...) string {
    prompt := p.buildBasePrompt(basePrompt)

    // ONLY append startup context if content exists
    if p.startupContextService.HasContent() {
        contextText := p.startupContextService.GetContextText()
        if contextText != "" {
            prompt += fmt.Sprintf("\n\n<StartupContext>\n%s\n</StartupContext>", contextText)
        }
    }

    return prompt
}
```

### DI Registration

```go
// pkg/di/container.go

func Register(container *do.Container) {
    // ... existing providers ...

    // Startup context service
    do.Provide(container, startup.NewStartupContextService)
}
```

## Data Flow

1. **Application starts** → `ApplicationService.Run(ctx)`
2. **Check for startup flow** → `flowRegistry.GetStartupFlow()`
3. **Execute startup flow** (if found) → `flowExecutorService.Execute(ctx, flow, nil)`
4. **Store outputs** → `startupContextService.SetContext(result.Outputs)`
5. **Start main loop** → TUI channel starts
6. **Agent creation** → `PromptManager.BuildSystemPrompt()` called
7. **Inject context** → If available, adds `<StartupContext>...</StartupContext>` section
8. **Agent runs** → With enriched understanding of workspace/project

## Context Formatting

The `GetContextText()` method formats outputs as natural language:

### Strategy

1. **Dedicated fields first:** If outputs contain `context` or `text` key, use directly
2. **Fallback to formatting:** Format all key-value pairs as readable text
3. **Handle nested structures:** Gracefully format nested maps

### Examples

```go
// Simple text field
map[string]any{"context": "Project is a payment system"}
→ "Project is a payment system"

// Structured data
map[string]any{
    "project": "E-commerce",
    "focus": "Security",
}
→ "project: E-commerce\nfocus: Security"

// Nested structure
map[string]any{
    "info": map[string]any{
        "name": "MyApp",
        "stage": "production",
    }
}
→ "info: map[name:MyApp stage:production]"
```

## Error Handling

| Scenario | Behavior |
|----------|----------|
| No startup flow found | Application continues normally, no context set |
| Startup flow execution fails | Log error, continue without context |
| Startup flow has no outputs | `HasContent()` returns `false`, no injection |
| Outputs is empty map | No context section added to prompt |

**Principle:** Startup flow failures should never prevent the application from starting.

## Prompt Behavior Guarantee

When there is no startup context:
- No `<StartupContext>` tags are added to the system prompt
- The prompt remains completely clean
- The agent has no knowledge of the startup flow mechanism

## Files to Create

```
pkg/startup/
├── service.go           # StartupContextService interface and impl
└── service_test.go      # Unit tests
```

## Files to Modify

```
pkg/app/service.go                    # Execute flow, set context
pkg/prompts/manager.go                # Inject context into system prompt
pkg/di/container.go                   # Register StartupContextService
pkg/di/container_test.go              # Add tests for new provider
```

## Testing

### Unit Tests for StartupContextService

```go
func TestStartupContextService_SetAndGet(t *testing.T) {
    service := startup.NewStartupContextService()

    outputs := map[string]any{
        "context": "Test context",
    }
    service.SetContext(outputs)

    assert.True(t, service.HasContent())
    assert.Equal(t, "Test context", service.GetContextText())
}

func TestStartupContextService_HasContent_Empty(t *testing.T) {
    service := startup.NewStartupContextService()
    assert.False(t, service.HasContent())
}

func TestStartupContextService_Clear(t *testing.T) {
    service := startup.NewStartupContextService()
    service.SetContext(map[string]any{"test": "value"})
    service.Clear()
    assert.False(t, service.HasContent())
}
```

### Integration Test

Create a test startup flow and verify:
1. Flow executes on startup
2. Context is stored in StartupContextService
3. PromptManager includes context in system prompt
4. No context tags when startup flow is absent

## Example Startup Flow

```xml
<!-- .gollum/flows/startup/main.xml -->
<flow name="startup">
    <input>
        <strings>
            <field name="workspace_path" type="string" />
        </strings>
    </input>

    <states>
        <state name="analyze" initial="true">
            <steps>
                <!-- Read README -->
                <func function="read_file">
                    <params>
                        <param name="path" value="${input.workspace_path}/README.md" />
                    </params>
                </func>

                <!-- Analyze git status -->
                <func function="git_status">
                    <params>
                        <param name="path" value="${input.workspace_path}" />
                    </params>
                </func>

                <!-- Format context for LLM -->
                <func builtin="assign">
                    <params>
                        <param name="from" value="Project Analysis: Based on README and git state, this is a payment processing system requiring security focus." />
                        <param name="to" value="${output.context}" />
                    </params>
                </func>
            </steps>
            <transitions>
                <transition event="done" target="done" />
            </transitions>
        </state>

        <state name="done" final="true" />
    </states>

    <output>
        <strings>
            <field name="context" type="string" />
        </strings>
    </output>
</flow>
```

## Success Criteria

- [ ] StartupContextService created and registered in DI
- [ ] ApplicationService executes startup flow and stores context
- [ ] PromptManager injects context into system prompt when available
- [ ] No context tags appear when startup flow is absent
- [ ] Unit tests for StartupContextService pass
- [ ] Integration test verifies end-to-end flow
- [ ] Application continues normally if startup flow fails

## References

- `pkg/app/service.go` - Application service
- `pkg/flows/registry/service.go` - Flow registry with GetStartupFlow()
- `pkg/prompts/manager.go` - Prompt manager
- `pkg/di/container.go` - DI container
