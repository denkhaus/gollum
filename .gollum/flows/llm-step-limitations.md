# LLM Step Limitations

## Status

✅ **RESOLVED** - Implemented in commits 600237e and earlier

The `react` strategy is now used by default with configurable parameters:
- Flow YAML: `<strategy maxIterations="N" maxRepeatedActions="N" />` in agent
- Env vars: `GOLLUM_SUBAGENT_STRATEGY_*`, `GOLLUM_SUPERVISOR_STRATEGY_*`
- Default: MaxIterations=20, MaxRepeatedActions=3

## Implementation Details

See: `docs/superpowers/specs/2026-04-15-configurable-llm-strategy-design.md`

---

## Known Issue: Infinite Tool Calling Loop (RESOLVED)

**Before the fix:** The `simple` strategy used by default for LLM steps, subagents and the supervisor had a critical flaw: it continued as long as there was input, with no maximum iteration limit. This caused infinite loops when:

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

## Solution (IMPLEMENTED)

The `simple` strategy has been replaced with `react` strategy which provides:

1. **MaxIterations** (default: 20) - Prevents infinite loops
2. **MaxRepeatedActions** (default: 3) - Stops if same action repeated
3. **Consecutive error tracking** - Stops on repeated failures

**Implementation:** See `pkg/strategy/` for strategy builder, `pkg/flows/executor/llm_step.go` for LLM step integration.

### Configuration Options

**Via environment variables:**
```bash
# For subagents
GOLLUM_SUBAGENT_STRATEGY_MAX_ITERATIONS=20
GOLLUM_SUBAGENT_STRATEGY_MAX_REPEATED_ACTIONS=3

# For supervisors
GOLLUM_SUPERVISOR_STRATEGY_MAX_ITERATIONS=20
GOLLUM_SUPERVISOR_STRATEGY_MAX_REPEATED_ACTIONS=3
```

**Via flow YAML:**
```xml
<agent name="summarizer">
    <strategy maxIterations="10" maxRepeatedActions="2" />
    <llm model="anthropic/sonnet-4.6" />
</agent>
```

**Reference:** [github.com/m-mizutani/gollem/blob/main/strategy/react/react.go](https://github.com/m-mizutani/gollem/blob/main/strategy/react/react.go)
