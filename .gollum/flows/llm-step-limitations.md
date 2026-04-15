# LLM Step Limitations

## Known Issue: Infinite Tool Calling Loop

The `simple` strategy used by default for LLM steps,subagents and the supervisor has a critical flaw: it continues as long as there is input, with no maximum iteration limit. This causes infinite loops when:

1. The LLM is asked to use tools (like `emit_log`)
2. The LLM calls the tool successfully
3. The strategy gives the LLM another turn
4. The LLM generates a variation and calls the tool again
5. Loop repeats indefinitely...

### Root Cause

The `simple` strategy only terminates when the LLM makes **no tool calls**. If the LLM keeps calling tools, the loop never ends.

**Code reference:** [github.com/m-mizutani/gollem/blob/main/strategy/simple/simple.go](https://github.com/m-mizutani/gollem/blob/main/strategy/simple/simple.go)

```go
// Terminates only when NO tool calls are made
if len(state.LastResponse.FunctionCalls) == 0 {
    return nil, executeResponse, nil  // Stop
}
return state.NextInput, nil, nil  // Continue - infinite loop if tools used
```

## Workaround

For LLM steps that need to capture output, avoid using tools. Instead:

```xml
<!-- ✅ GOOD: Capture text response directly -->
<step type="llm" agent="summarizer">
    <prompt>Provide a brief summary...</prompt>
    <result assignTo="output.summary" />
</step>

<!-- ❌ BAD: Tool calling causes infinite loop -->
<step type="llm" agent="summarizer">
    <prompt>Use emit_log to provide a summary...</prompt>
    <tools>emit_log</tools>
</step>
```

## Future Solution

Replace the `simple` strategy with `react` strategy which provides:

1. **MaxIterations** (default: 20) - Prevents infinite loops
2. **MaxRepeatedActions** (default: 3) - Stops if same action repeated
3. **Consecutive error tracking** - Stops on repeated failures

**Implementation required in:** `pkg/agents/factory.go`

```go
import "github.com/m-mizutani/gollem/strategy/react"

// Configure react strategy with safeguards
strategy := react.New(llmClient,
    react.WithMaxIterations(10),
    react.WithMaxRepeatedActions(2),
)
```

**Reference:** [github.com/m-mizutani/gollem/blob/main/strategy/react/react.go](https://github.com/m-mizutani/gollem/blob/main/strategy/react/react.go)
