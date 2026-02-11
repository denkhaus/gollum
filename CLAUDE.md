# Gollum Project - Development Guidelines

## Mock Generation with MockGen

When working with interfaces in tests, use the centralized mock generation approach.
Maintain a centralized `pkg/mocks/generate.go` file where all mocks generation commands live


### Use Centralized Mocks
Import and use mocks from the centralized location:
```go
import "github.com/denkhaus/gollum/pkg/mocks"

ctrl := gomock.NewController(t)
mockRegistry := mocks.NewMockAgentRegistry(ctrl)
```

### Benefits
- Single source of truth for all mocks
- Reusable across all packages
- Easy regeneration with go generate
- Consistent mock patterns

## Project Structure

- `pkg/registry/` - Agent registry and management
- `pkg/tools/` - Tool implementations (SpawnAgent, SendMessage, RemoveAgent, etc.)
- `pkg/agents/` - Agent implementations and providers
- `pkg/mocks/` - Centralized mock files for testing
- `pkg/di/` - Dependency injection container

## Agent Tools

### Available Tools
- **SpawnAgentTool**: Creates new subagents with custom system prompts
- **DEPRECATED SendMessageTool**: Sends messages between agents
- **RemoveAgentTool**: Removes agents and all their subagents recursively
- **CurrentTimeTool**: Provides current time information

### Tool Implementation Pattern
1. Create tool struct with registry and senderID
2. Implement provider interface for DI
3. Add to DI container in `pkg/di/container.go`
4. Register in default agent configuration in `pkg/agents/default.go`

### Testing Tools
- Use centralized mocks from `pkg/mocks/`
- Follow GoMock patterns with `gomock.Controller`
- Test both success and error cases
- Validate permissions and edge cases

## Prompt Optimizer

The Prompt Optimizer automatically improves agent prompts using execution trajectory feedback.

### Configuration

Environment variables for optimizer configuration:

- `GOLLUM_OPTIMIZER_STRATEGY`: Optimization strategy (default: "gradient")
  - Options: "gradient", "meta-prompt", "prompt-memory"
- `GOLLUM_OPTIMIZER_PROVIDER`: LLM provider for optimization (default: "anthropic")
  - Options: "anthropic", "openai", "gemini"
- `GOLLUM_OPTIMIZER_MAX_REFLECTION`: Maximum reflection steps for gradient (default: "5")
- `GOLLUM_OPTIMIZER_MIN_REFLECTION`: Minimum reflection steps for gradient (default: "2")

Example:
```bash
export GOLLUM_OPTIMIZER_STRATEGY=gradient
export GOLLUM_OPTIMIZER_PROVIDER=anthropic
export GOLLUM_OPTIMIZER_MAX_REFLECTION=5
```

### Usage

#### DI Integration

The PromptOptimizer is automatically registered in the DI container:

```go
import "github.com/denkhaus/gollum/pkg/di"

// In your DI setup
optimizer := do.MustInvoke[optimizer.PromptOptimizer](container)
```

#### Optimizing a Prompt

```go
import "github.com/denkhaus/gollum/pkg/prompt/optimizer"

// Create trajectory from execution
trajectory := &optimizer.Trajectory{
    Messages: agentMessages, // From agent session
    Feedback: "The agent struggled with multi-step reasoning",
}

// Prepare optimization input
input := optimizer.OptimizerInput{
    Prompt:             "You are a helpful assistant...",
    Trajectories:       []*optimizer.Trajectory{trajectory},
    UpdateInstructions: "Improve reasoning capabilities",
}

// Run optimizer
result, err := opt.Optimize(ctx, &input)
if err != nil {
    log.Fatal(err)
}

// Result contains improved prompt
fmt.Printf("New prompt: %s\n", result.NewPrompt)
fmt.Printf("Changes: %s\n", result.ChangeDescription)
fmt.Printf("Warrants adjustment: %v\n", result.WarrantsAdjustment)
```

#### Feedback Types

The optimizer accepts multiple feedback formats:

```go
// Simple string feedback
trajectory := &optimizer.Trajectory{
    Messages: messages,
    Feedback: "Agent failed to consider edge cases",
}

// Structured feedback with scoring
trajectory := &optimizer.Trajectory{
    Messages: messages,
    Feedback: &optimizer.Feedback{
        Score:        0.6,
        Comment:      "Performance below expectations",
        FailureModes: []string{"missing context", "incomplete reasoning"},
        Outcome:      "partial_success",
    },
}

// Edit feedback with specific revisions
trajectory := &optimizer.Trajectory{
    Messages: messages,
    Feedback: &optimizer.EditFeedback{
        Revised: "The corrected response should be...",
        Edits: []optimizer.TextEdit{
            {OldText: "old approach", NewText: "new approach", Reason: "clarity"},
        },
    },
}
```

### Optimization Strategies

The Prompt Optimizer implements three strategies adapted from [LangMEM](https://github.com/langchain-ai/langmem).
Choose the appropriate strategy based on your requirements:

#### Strategy Selection Guide

| Strategy | LLM Calls | Best For | Cost | Speed |
|----------|-----------|----------|-------|-------|
| **Gradient** | 2-10 | Complex improvements, thorough analysis | Highest | Slowest |
| **Meta-Prompt** | 1-5 | Balanced optimization | Medium | Medium |
| **Prompt Memory** | 1 | Simple adjustments | Lowest | Fastest |

#### Prompt Memory (`prompt-memory`)
**When to use:**
- Fast, single-shot optimization needed (1 LLM call)
- Simple adjustments are sufficient
- Cost/latency is a concern
- Rapid prototyping or quick iterations

**Limitations:**
- Limited ability to learn from complex patterns
- Only analyzes the first trajectory
- No iterative refinement

**Configuration:**
```bash
export GOLLUM_OPTIMIZER_STRATEGY=prompt-memory
```

#### Meta-Prompt (`metaprompt`)
**When to use:**
- Balance between speed and quality needed
- Moderate cost is acceptable (1-5 LLM calls)
- Direct pattern learning from examples works well
- General-purpose optimization for most use cases

**Characteristics:**
- Combines analysis and prompt update in iterative process
- Good default choice when unsure which strategy to use
- Each iteration is a single LLM call

**Configuration:**
```bash
export GOLLUM_OPTIMIZER_STRATEGY=metaprompt
export GOLLUM_OPTIMIZER_MAX_REFLECTION=5  # Max iterations
export GOLLUM_OPTIMIZER_MIN_REFLECTION=2  # Min iterations
```

#### Gradient (`gradient`) - **Default**
**When to use:**
- Most thorough analysis required
- Complex improvements needed
- Cost is not a primary constraint
- Production-quality optimization for critical prompts
- Separation of concerns (reflection phase + update phase) beneficial
- Extracting feedback from conversational context is important

**Characteristics:**
- Two-phase approach: Reflection Loop → Update Phase
- Each reflection step generates hypotheses and recommendations
- Most comprehensive but most expensive strategy

**Configuration:**
```bash
export GOLLUM_OPTIMIZER_STRATEGY=gradient
export GOLLUM_OPTIMIZER_MAX_REFLECTION=5  # Max reflection steps
export GOLLUM_OPTIMIZER_MIN_REFLECTION=2  # Min reflection steps
```

**Best Practices:**
- Use higher MaxReflectionSteps (5-10) for complex, critical prompts
- Use MinReflectionSteps of at least 2 to ensure convergence
- Consider cost implications: this strategy can be expensive

### Best Practices

- Run optimization at session end, not during execution
- Use gradient strategy for critical agent prompts requiring thorough analysis
- Collect feedback from actual user interactions when possible
- Monitor optimization results for quality degradation
- Keep prompt history for rollback if needed
- Provide clear UpdateInstructions to guide optimization direction
- Use structured Feedback (with scores) for more consistent results
- Test optimized prompts before deploying to production

## Langfuse Tracing

The LangfuseHook provides comprehensive tracing for LLM interactions, tool executions, and agent lifecycle events through Langfuse SDK integration.

### Configuration

Environment variables for Langfuse tracing:

- `GOLLUM_LANGFUSE_ENABLED`: Enable/disable tracing (default: "false")
- `GOLLUM_LANGFUSE_HOST`: Langfuse server URL (default: "https://cloud.langfuse.com")
- `GOLLUM_LANGFUSE_PUBLIC_KEY`: Langfuse public key for authentication
- `GOLLUM_LANGFUSE_SECRET_KEY`: Langfuse secret key for authentication
- `GOLLUM_LANGFUSE_FLUSH_INTERVAL`: Flush interval in milliseconds (default: "1000")
- `GOLLUM_LANGFUSE_MAX_QUEUE_SIZE`: Max queue size before flush (default: "100")

Example:
```bash
export GOLLUM_LANGFUSE_ENABLED=true
export GOLLUM_LANGFUSE_HOST=https://cloud.langfuse.com
export GOLLUM_LANGFUSE_PUBLIC_KEY=pk-lf-...
export GOLLUM_LANGFUSE_SECRET_KEY=sk-lf-...
```

### Hook Registration in main.go

Register LangfuseHook with HookManager during initialization:

```go
import (
    "github.com/denkhaus/gollum/pkg/builtin"
    "github.com/denkhaus/gollum/pkg/hooks"
)

func main() {
    // ... DI container setup ...

    // Get LangfuseHook from DI
    langfuseHook := do.MustInvoke[*builtin.LangfuseHook](injector)

    // Get HookManager from DI
    hm := do.MustInvoke[HookManager](injector)

    // Register all Langfuse tracing hooks
    if err := builtin.RegisterLangfuseHooks(hm, langfuseHook); err != nil {
        log.Fatal("Failed to register Langfuse hooks", zap.Error(err))
    }
}
```

### Tracing Enable/Disable Control

- Set `GOLLUM_LANGFUSE_ENABLED=false` to disable tracing
- When disabled, hooks are no-ops (no performance overhead)
- Enable at runtime by updating config (if supported)

### Viewing Traces in Langfuse UI

1. Navigate to your Langfuse instance (e.g., https://cloud.langfuse.com)
2. Select your project/org
3. View traces in the "Traces" tab
4. Filter by session ID, agent ID, or time range
5. Inspect individual spans for LLM requests, tool calls, and agent events

## See Also

- [Langfuse Tracing Guide](guides/guide.golang.langfuse-tracing.md) - Comprehensive guidance on Langfuse integration patterns
- [Prompt Optimization](#prompt-optimizer) - Automatic prompt improvement using execution feedback
- [Go Testing Guide](guides/guide.golang.testing.md) - Unit and integration testing patterns
