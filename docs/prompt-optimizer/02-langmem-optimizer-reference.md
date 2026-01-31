# LangMEM Optimizer Reference

**Source:** [langchain-ai/langmem Documentation](https://github.com/langchain-ai/langmem)
**Extracted:** 2025-01-31
**Purpose:** API reference for implementing Go-based Prompt Optimizer

---

## Overview

LangMEM provides three optimization strategies for improving prompts through conversation analysis:

1. **Gradient Optimizer** - Separates analysis from application (2-10 LLM calls)
2. **Meta-Prompt Optimizer** - Combined reflection and update (1-5 LLM calls)
3. **Prompt Memory Optimizer** - Simple single-shot optimization (1 LLM call)

---

## API Signatures

### Single Prompt Optimizer

```python
def create_prompt_optimizer(
    model: str | BaseChatModel,
    /,
    *,
    kind: Literal["gradient", "prompt_memory", "metaprompt"] = "gradient",
    config: Optional[OptimizerConfig] = None,
) -> Runnable[OptimizerInput, str]:
    """Create a prompt optimizer that improves prompt effectiveness."""
```

### Multi Prompt Optimizer

```python
def create_multi_prompt_optimizer(
    model: str | BaseChatModel,
    /,
    *,
    kind: Literal["gradient", "prompt_memory", "metaprompt"] = "gradient",
    config: Optional[dict] = None,
) -> Runnable[MultiPromptOptimizerInput, list[Prompt]]:
    """Create optimizer for multiple prompts with credit assignment."""
```

---

## Input Types

### OptimizerInput (Single Prompt)

```python
class OptimizerInput(TypedDict):
    trajectories: Union[Sequence[AnnotatedTrajectory], str]
    prompt: Union[str, Prompt]

class Prompt(TypedDict, total=False):
    prompt: str              # Required
    update_instructions: str  # Optional
    feedback: str            # Optional
    when_to_update: str      # Optional

class AnnotatedTrajectory(Tuple):
    # Format: (conversation_messages, feedback)
    # conversation_messages: List[dict] with "role" and "content"
    # feedback: Optional - can be dict with score/comment, string, or None
```

### MultiPromptOptimizerInput

```python
class MultiPromptOptimizerInput(TypedDict):
    trajectories: Union[Sequence[AnnotatedTrajectory], str]
    prompts: list[Prompt]  # Each with "name" field
```

---

## Strategy Details

### 1. Gradient Optimizer

**Flow:**
```
Reflection Phase (think/critique) → Recommendation → Update Phase
```

**Configuration:**
```python
class GradientOptimizerConfig(TypedDict, total=False):
    gradient_prompt: str           # Custom reflection prompt
    metaprompt: str               # Custom update prompt
    max_reflection_steps: int     # Default: 5
    min_reflection_steps: int     # Default: 1
```

**Characteristics:**
- Most thorough analysis
- Separates "what to improve" from "how to improve"
- Uses think/critique tools for reflection
- Higher cost but better quality

---

### 2. Meta-Prompt Optimizer

**Flow:**
```
Reflection + Update in Single Call
```

**Configuration:**
```python
class MetapromptOptimizerConfig(TypedDict, total=False):
    metaprompt: str               # Combined instructions
    max_reflection_steps: int     # Default: 5
    min_reflection_steps: int     # Default: 1
```

**Characteristics:**
- Balance of speed and quality
- Direct prompt updates
- Moderate cost (1-5 LLM calls)

---

### 3. Prompt Memory Optimizer

**Flow:**
```
Single LLM Call → Direct Update
```

**Configuration:**
- No configuration required

**Characteristics:**
- Fastest option
- Simple metaprompt
- Limited ability to learn complex patterns

---

## Multi-Prompt Optimization (Credit Assignment)

The multi-prompt optimizer adds **credit assignment** - determining which prompts in a system need updates based on trajectory analysis.

```python
optimizer = create_multi_prompt_optimizer(
    "anthropic:claude-3-5-sonnet-latest",
    kind="gradient"
)

trajectories = [...]  # Conversation data
prompts = [
    {"name": "research", "prompt": "..."},
    {"name": "summarize", "prompt": "..."},
]

updated = await optimizer.ainvoke({
    "trajectories": trajectories,
    "prompts": prompts
})
```

**Process:**
1. Analyze trajectories
2. Classify which prompts contributed to failures
3. Update only the identified prompts
4. Return all prompts (updated + unchanged)

---

## Usage Examples

### Basic Usage

```python
from langmem import create_prompt_optimizer

optimizer = create_prompt_optimizer(
    "anthropic:claude-3-5-sonnet-latest",
    kind="gradient",
    config={"max_reflection_steps": 3, "min_reflection_steps": 1}
)

trajectories = [
    (
        [
            {"role": "user", "content": "Tell me about Mars"},
            {"role": "assistant", "content": "Mars is..."},
        ],
        {"score": 0.7, "comment": "Needs more structure"}
    )
]

result = optimizer.invoke({
    "trajectories": trajectories,
    "prompt": "You are a helpful assistant"
})
```

### With Update Instructions

```python
result = optimizer.invoke({
    "trajectories": trajectories,
    "prompt": {
        "prompt": "You are a helpful assistant",
        "update_instructions": "Maintain friendly tone",
        "when_to_update": "Only if score < 0.8"
    }
})
```

### Async Usage

```python
result = await optimizer.ainvoke({
    "trajectories": trajectories,
    "prompt": "You are a helpful assistant"
})
```

---

## Go Implementation Considerations

### 1. Type System

```go
// OptimizerInput mirrors Python's OptimizerInput
type OptimizerInput struct {
    Trajectories interface{} // Either string or []*Trajectory
    Prompt       interface{} // Either string or *Prompt
}

type Prompt struct {
    Prompt              string
    UpdateInstructions  string
    Feedback            string
    WhenToUpdate        string
}

type Trajectory struct {
    Messages []Message
    Feedback interface{} // Can be nil, string, or Feedback struct
}

type Feedback struct {
    Score   float64
    Comment string
    Revised string // For edit-based feedback
}
```

### 2. Strategy Pattern

```go
type OptimizerStrategy string

const (
    StrategyGradient     OptimizerStrategy = "gradient"
    StrategyMetaPrompt   OptimizerStrategy = "metaprompt"
    StrategyPromptMemory OptimizerStrategy = "prompt_memory"
)

type OptimizerConfig struct {
    Kind                OptimizerStrategy
    MaxReflectionSteps  int
    MinReflectionSteps  int
    GradientPrompt      string
    MetaPrompt          string
}
```

### 3. Runnable Interface

Instead of Python's `Runnable`, use a Go interface:

```go
type PromptOptimizer interface {
    Optimize(ctx context.Context, input *OptimizerInput) (string, error)
}

type MultiPromptOptimizer interface {
    OptimizeMulti(ctx context.Context, input *MultiOptimizerInput) ([]*Prompt, error)
}
```

---

## Performance Comparison

| Strategy | LLM Calls | Quality | Speed | Best For |
|----------|-----------|---------|-------|----------|
| `prompt_memory` | 1 | Basic | Fast | Simple adjustments |
| `metaprompt` | 1-5 | Good | Medium | Balanced optimization |
| `gradient` | 2-10 | Best | Slow | Complex improvements |

---

## See Also

- [LangMEM Prompt Templates](./01-langmem-prompt-templates.md)
- [LangMEM Architecture](./03-langmem-architecture.md)
- [Original Repository](https://github.com/langchain-ai/langmem)
