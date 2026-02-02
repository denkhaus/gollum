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

#### Gradient Descent (Default)
Multi-phase reflection with think/critique/recommend tools. Best for comprehensive analysis.
- Uses multiple reflection iterations (configurable via MAX_REFLECTION/MIN_REFLECTION)
- Produces structured analysis with hypotheses and recommendations
- Most comprehensive but slower than other strategies

#### Meta-Prompt
Single-phase combined reflection and update. Faster, less comprehensive.
- Combines analysis and prompt update in single LLM call
- Good balance between speed and quality
- Suitable for most optimization scenarios

#### Prompt Memory
Direct single-shot optimization. Fastest for minor adjustments.
- Single-pass optimization without explicit reflection
- Best for quick prompt tweaks
- Less detailed analysis than other strategies

### Best Practices

- Run optimization at session end, not during execution
- Use gradient strategy for critical agent prompts requiring thorough analysis
- Collect feedback from actual user interactions when possible
- Monitor optimization results for quality degradation
- Keep prompt history for rollback if needed
- Provide clear UpdateInstructions to guide optimization direction
- Use structured Feedback (with scores) for more consistent results
- Test optimized prompts before deploying to production
