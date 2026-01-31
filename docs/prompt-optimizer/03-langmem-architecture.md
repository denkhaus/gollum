# LangMEM Architecture Overview

**Source:** [langchain-ai/langmem](https://github.com/langchain-ai/langmem)
**Extracted:** 2025-01-31
**Purpose:** Understanding LangMEM's architecture for Go implementation

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                         LangMEM Prompt Optimization System                     │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │                         PUBLIC API LAYER                               │   │
│  │                                                                         │   │
│  │  ┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐  │   │
│  │  │create_prompt_    │    │create_multi_     │    │Trajectory        │  │   │
│  │  │optimizer()       │    │prompt_optimizer()│    │Formats           │  │   │
│  │  └────────▲─────────┘    └────────▲─────────┘    └──────────────────┘  │   │
│  └───────────┼─────────────────────┼────────────────────────────────────────┘   │
│              │                     │                                           │
│  ┌───────────┴─────────────────────┴───────────────────────────────────────┐   │
│  │                         STRATEGY LAYER                                  │   │
│  │                                                                         │   │
│  │  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐              │   │
│  │  │Gradient      │    │Meta-Prompt   │    │Prompt        │              │   │
│  │  │Optimizer     │    │Optimizer     │    │Memory        │              │   │
│  │  │              │    │              │    │Optimizer     │              │   │
│  │  │ think/critique│    │Reflect+     │    │Single-shot   │              │   │
│  │  │ → recommend   │    │Update        │    │              │              │   │
│  │  └──────────────┘    └──────────────┘    └──────────────┘              │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│              │                     │                     │                     │
│  ┌───────────┴─────────────────────┴─────────────────────┴───────────────────┐   │
│  │                         TOOLING LAYER                                    │   │
│  │                                                                         │   │
│  │  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐              │   │
│  │  │create_       │    │Prompt        │    │format_       │              │   │
│  │  │extractor()   │    │Extraction    │    │sessions()    │              │   │
│  │  │              │    │Schema        │    │              │              │   │
│  │  └──────────────┘    └──────────────┘    └──────────────┘              │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│              │                                                                  │
│  ┌───────────┴───────────────────────────────────────────────────────────────┐   │
│  │                         LLM INTEGRATION                                   │   │
│  │                                                                         │   │
│  │  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐              │   │
│  │  │LangChain     │    │BaseChatModel │    │LangSmith     │              │   │
│  │  │Core          │    │              │    │Tracing       │              │   │
│  │  └──────────────┘    └──────────────┘    └──────────────┘              │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## Package Structure

```
langmem/
├── src/langmem/
│   ├── prompts/
│   │   ├── optimization.py      # Main optimizer factory
│   │   ├── gradient.py           # Gradient strategy
│   │   ├── metaprompt.py         # Meta-prompt strategy
│   │   ├── prompt.py             # Prompt memory strategy
│   │   ├── stateless.py          # State-based optimization
│   │   ├── types.py              # Type definitions
│   │   └── utils.py              # Utility functions
│   ├── utils.py                  # Shared utilities
│   └── graphs/
│       └── prompts.py            # Graph-based optimization
│
├── docs/docs/
│   ├── guides/
│   │   └── optimize_memory_prompt.md
│   └── reference/
│       └── prompt_optimization.md
│
└── examples/
    └── intro_videos/
        └── procedural_memory.ipynb
```

---

## Core Components

### 1. Optimizer Factory (`optimization.py`)

```python
def create_prompt_optimizer(
    model: str | BaseChatModel,
    kind: Literal["gradient", "metaprompt", "prompt_memory"] = "gradient",
    config: Optional[OptimizerConfig] = None,
) -> Runnable[OptimizerInput, str]:
    """Factory function creating appropriate optimizer instance."""
```

**Responsibilities:**
- Strategy selection based on `kind` parameter
- Configuration validation and defaults
- Returns initialized optimizer instance

---

### 2. Strategy Implementations

#### Gradient Optimizer (`gradient.py`)

```python
class GradientPromptOptimizer(Runnable[OptimizerInput, str]):
    def __init__(self, model, config):
        self.just_think_chain = create_extractor([think, critique])
        self.any_chain = create_extractor([think, critique, recommend])
        self.final_chain = create_extractor([recommend])
```

**Flow:**
```
1. Reflection Loop (min to max steps)
   - think(): Generate hypothesis
   - critique(): Critique previous reasoning
   - recommend(): Decide if adjustment needed

2. If warrants_adjustment:
   - Update prompt with hypotheses and recommendations
3. Else:
   - Return original prompt
```

#### Meta-Prompt Optimizer (`metaprompt.py`)

```python
class MetaPromptOptimizer(Runnable[OptimizerInput, str]):
    def __init__(self, model, config):
        self.reflect_chain = create_extractor([think, critique])
```

**Flow:**
```
1. Combined reflection + update prompt
2. Reflection loop with tool calls
3. Direct prompt update in final step
```

#### Prompt Memory Optimizer

```python
class PromptMemoryMultiple(Runnable[OptimizerInput, str]):
    """Simple single-shot optimization."""
```

**Flow:**
```
1. Single LLM call with trajectories
2. Direct prompt update
```

---

### 3. Tool Definition

All strategies use **function calling** via `trustcall.create_extractor()`:

```python
def think(thought: str) -> str:
    """Reflection tool for reasoning."""
    return ""

def critique(criticism: str) -> str:
    """Critique tool for evaluating reasoning."""
    return ""

def recommend(
    warrants_adjustment: bool,
    hypotheses: Optional[str] = None,
    full_recommendations: Optional[str] = None,
) -> str:
    """Decision tool for applying updates."""
    return ""
```

**Go Translation:**

```go
type OptimizerTool struct {
    Name        string
    Description string
    Parameters  interface{}
}

var (
    ThinkTool = OptimizerTool{
        Name:        "think",
        Description: "Generate hypothesis about prompt issues",
        Parameters:  ThinkParameters{},
    }
    // ... other tools
)
```

---

### 4. Data Flow

```
┌────────────┐
│ Trajectories│
│  + Feedback │
└──────┬─────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│  format_sessions() - Format trajectories for LLM            │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│  Reflection Phase                                           │
│  ┌────────────────────────────────────────────────────┐     │
│  │  Loop (min to max steps):                          │     │
│  │    1. invoke think/critique tools                 │     │
│  │    2. build message history                       │     │
│  │    3. check for recommend() response              │     │
│  └────────────────────────────────────────────────────┘     │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│  Decision Point                                             │
│  ┌─────────────┬────────────────────┐                       │
│  │ warrants_   │ warrants_          │                       │
│  │ adjustment  │ = False            │                       │
│  │ = True      │                    │                       │
│  ▼             ▼                    │                       │
│ Update Prompt  Return Original      │                       │
└─────────────────────────────────────┴───────────────────────┘
```

---

## Key Design Patterns

### 1. Runnable Protocol

```python
class Runnable[Input, Output]:
    def invoke(self, input: Input) -> Output: ...
    def ainvoke(self, input: Input) -> Awaitable[Output]: ...
```

**Go Equivalent:**
```go
type Runnable[Input any, Output any] interface {
    Invoke(ctx context.Context, input Input) (Output, error)
}
```

### 2. Factory Pattern

```python
# Strategy selection factory
def create_prompt_optimizer(model, kind, config):
    if kind == "gradient":
        return GradientPromptOptimizer(model, config)
    elif kind == "metaprompt":
        return MetaPromptOptimizer(model, config)
    # ...
```

### 3. Configuration Builder

```python
class GradientOptimizerConfig(TypedDict, total=False):
    gradient_prompt: str
    metaprompt: str
    max_reflection_steps: int
    min_reflection_steps: int
```

**Go Equivalent:**
```go
type OptimizerConfig struct {
    Kind               OptimizerStrategy
    GradientPrompt     string
    MetaPrompt         string
    MaxReflectionSteps int
    MinReflectionSteps int
}

func (c *OptimizerConfig) WithDefaults() *OptimizerConfig {
    if c.MaxReflectionSteps == 0 {
        c.MaxReflectionSteps = 5
    }
    // ...
    return c
}
```

---

## Go Implementation Architecture

### Proposed Package Structure

```
pkg/prompt/optimizer/
├── types.go              # Core types (Trajectory, Feedback, etc.)
├── optimizer.go          # Optimizer interface + factory
├── strategies/
│   ├── gradient.go       # Gradient strategy
│   ├── metaprompt.go     # Meta-prompt strategy
│   └── memory.go         # Prompt memory strategy
├── templates.go          # Prompt templates
├── tools.go              # Tool definitions (think, critique, recommend)
├── utils.go              # Utilities (format trajectories, etc.)
└── optimizer_test.go     # Tests
```

### Interface Design

```go
// Optimizer defines the prompt optimizer interface
type Optimizer interface {
    // Optimize improves a prompt based on conversation trajectories
    Optimize(ctx context.Context, input *OptimizerInput) (string, error)
}

// MultiOptimizer handles multiple prompts with credit assignment
type MultiOptimizer interface {
    // OptimizeMulti optimizes multiple prompts
    OptimizeMulti(ctx context.Context, input *MultiOptimizerInput) ([]*Prompt, error)
}

// Factory function
func NewOptimizer(
    llm gollem.LLMClient,
    config *OptimizerConfig,
) (Optimizer, error) {
    switch config.Kind {
    case StrategyGradient:
        return NewGradientOptimizer(llm, config)
    case StrategyMetaPrompt:
        return NewMetaPromptOptimizer(llm, config)
    case StrategyPromptMemory:
        return NewPromptMemoryOptimizer(llm)
    default:
        return nil, fmt.Errorf("unknown strategy: %s", config.Kind)
    }
}
```

---

## Dependencies Analysis

| LangMEM Dependency | Go Equivalent | Purpose |
|-------------------|---------------|---------|
| `langchain_core` | `github.com/m-mizutani/gollem` | LLM abstraction |
| `trustcall` | Custom implementation | Tool calling |
| `langsmith` | `go.opentelemetry.io/otel` | Tracing/observability |
| `pydantic` | Go struct tags | Validation |
| `langchain_core.messages` | Custom Message types | Message handling |

---

## See Also

- [LangMEM Prompt Templates](./01-langmem-prompt-templates.md)
- [LangMEM Optimizer Reference](./02-langmem-optimizer-reference.md)
- [Original Repository](https://github.com/langchain-ai/langmem)
