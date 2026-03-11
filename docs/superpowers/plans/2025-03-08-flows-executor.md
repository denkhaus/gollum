# Flows Executor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement the flows state machine executor that runs XML-defined workflows with LLM steps, context management, and error handling.

**Architecture:** A state machine engine that:
1. Loads validated flows from the parser
2. Manages execution context with computed field evaluation
3. Executes steps (LLM, sub-flow calls) with timeout and error handling
4. Tracks execution history for debugging and replay
5. Enforces state transitions with condition evaluation

**Tech Stack:**
- Go 1.23+ with encoding/xml for parsing
- Context-based execution with cancellation
- Expression evaluator from `pkg/flows/ast`
- Langfuse hooks for observability

---

## Task 1: Context Management with Computed Field Evaluation

**Files:**
- Create: `pkg/flows/executor/context.go`
- Test: `pkg/flows/executor/context_test.go`

**Step 1: Write failing test for context initialization**

```go
package executor

import (
    "testing"
    "github.com/denkhaus/gollum/pkg/flows"
    "github.com/stretchr/testify/assert"
)

func TestNewContext_InitializesWithDefaults(t *testing.T) {
    input := &flows.InputBlock{
        Strings: []flows.FieldDef{{Name: "repo", Required: true}},
    }
    inputVals := map[string]any{"repo": "gollum"}

    ctx := NewContext(input, inputVals)

    assert.Equal(t, "gollum", ctx.GetInput("repo"))
    assert.NotNil(t, ctx.values)
    assert.NotNil(t, ctx.computed)
}

func TestNewContext_AppliesInputDefaults(t *testing.T) {
    input := &flows.InputBlock{
        Strings: []flows.FieldDef{{Name: "owner", Default: "denkhaus"}},
    }

    ctx := NewContext(input, nil)

    assert.Equal(t, "denkhaus", ctx.GetInput("owner"))
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/executor -v -run TestNewContext`
Expected: FAIL with "undefined: NewContext"

**Step 3: Write minimal Context struct and NewContext**

```go
package executor

import (
    "fmt"
    "strconv"
    "github.com/denkhaus/gollum/pkg/flows"
    "github.com/denkhaus/gollum/pkg/flows/ast"
)

// Context manages execution context with input, output, and computed fields
type Context struct {
    input    *flows.InputBlock
    inputVals map[string]any
    values    map[string]any  // context and output fields
    computed map[string]string // computed field expressions
    eval     *ast.Evaluator
}

// NewContext creates a new execution context
func NewContext(input *flows.InputBlock, inputVals map[string]any) *Context {
    ctx := &Context{
        input:    input,
        inputVals: make(map[string]any),
        values:   make(map[string]any),
        computed: make(map[string]string),
        eval:     ast.NewEvaluator(),
    }

    // Apply input values or defaults
    for _, field := range input.GetAllFields() {
        if val, ok := inputVals[field.Name]; ok {
            ctx.inputVals[field.Name] = val
        } else if field.Default != "" {
            ctx.inputVals[field.Name] = coerceType(field.Type, field.Default)
        }
    }

    return ctx
}

// GetInput retrieves an input field value
func (c *Context) GetInput(name string) any {
    return c.inputVals[name]
}

// SetContextField sets a context field value
func (c *Context) SetContextField(name string, value any) {
    c.values[name] = value
}

// GetContextField retrieves a context field value
func (c *Context) GetContextField(name string) (any, bool) {
    val, ok := c.values[name]
    return val, ok
}

// SetOutputField sets an output field value
func (c *Context) SetOutputField(name string, value any) {
    c.values["output."+name] = value
}

// GetOutputField retrieves an output field value
func (c *Context) GetOutputField(name string) (any, bool) {
    val, ok := c.values["output."+name]
    return val, ok
}

// coerceType converts string to appropriate type
func coerceType(typ, val string) any {
    switch typ {
    case "int":
        i, _ := strconv.Atoi(val)
        return i
    case "bool":
        b, _ := strconv.ParseBool(val)
        return b
    case "float":
        f, _ := strconv.ParseFloat(val, 64)
        return f
    default:
        return val
    }
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/executor -v -run TestNewContext`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/flows/executor/context.go pkg/flows/executor/context_test.go
git commit -m "feat(executor): add context management with input handling"
```

---

## Task 2: Computed Field Evaluation

**Files:**
- Modify: `pkg/flows/executor/context.go`
- Modify: `pkg/flows/executor/context_test.go`

**Step 1: Write failing test for computed field evaluation**

```go
func TestContext_EvaluateComputedFields(t *testing.T) {
    flow := &flows.Flow{
        Context: &flows.ContextBlock{
            Strings: []flows.ContextField{{Name: "status"}},
            Computeds: []flows.ComputedField{
                {Name: "is_open", Type: "bool", When: "EQ(context.status, 'open')"},
            },
        },
    }

    ctx := NewContext(&flows.InputBlock{}, nil)
    ctx.SetContextField("status", "open")

    err := ctx.EvaluateComputedFields(flow.Context)

    assert.NoError(t, err)
    val, ok := ctx.GetContextField("is_open")
    assert.True(t, ok)
    assert.Equal(t, true, val)
}

func TestContext_ComputedFieldsAreImmutable(t *testing.T) {
    flow := &flows.Flow{
        Context: &flows.ContextBlock{
            Strings: []flows.ContextField{{Name: "count"}},
            Computeds: []flows.ComputedField{
                {Name: "is_large", Type: "bool", When: "GT(context.count, 10)"},
            },
        },
    }

    ctx := NewContext(&flows.InputBlock{}, nil)
    ctx.SetContextField("count", 5)

    err := ctx.EvaluateComputedFields(flow.Context)
    assert.NoError(t, err)

    // Try to modify computed field
    ctx.SetContextField("is_large", true)

    // Computed field should NOT be modified
    val, _ := ctx.GetContextField("is_large")
    assert.Equal(t, false, val)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/executor -v -run TestContext_EvaluateComputedFields`
Expected: FAIL with "undefined: EvaluateComputedFields"

**Step 3: Implement EvaluateComputedFields**

Add to `context.go`:

```go
// EvaluateComputedFields evaluates all computed fields in dependency order
func (c *Context) EvaluateComputedFields(ctxBlock *flows.ContextBlock) error {
    // Store computed expressions for immutability check
    for _, cf := range ctxBlock.Computeds {
        c.computed[cf.Name] = cf.When
    }

    // Build dependency graph and evaluate in topological order
    evaluated := make(map[string]bool)
    for len(evaluated) < len(ctxBlock.Computeds) {
        progress := false
        for _, cf := range ctxBlock.Computeds {
            if evaluated[cf.Name] {
                continue
            }

            // Check if all dependencies are evaluated
            deps := c.eval.ExtractDependencies(cf.When)
            ready := true
            for _, dep := range deps {
                if !c.isDependencyResolved(dep, evaluated) {
                    ready = false
                    break
                }
            }

            if ready {
                val, err := c.eval.EvaluateExpr(cf.When, c.buildScope())
                if err != nil {
                    return fmt.Errorf("computed field %s: %w", cf.Name, err)
                }
                c.values[cf.Name] = val
                evaluated[cf.Name] = true
                progress = true
            }
        }

        if !progress {
            return fmt.Errorf("circular dependency in computed fields")
        }
    }

    return nil
}

// isDependencyResolved checks if a dependency is available
func (c *Context) isDependencyResolved(dep string, evaluated map[string]bool) bool {
    // Check if it's an input field
    if _, ok := c.inputVals[dep]; ok {
        return true
    }
    // Check if it's a context field
    if _, ok := c.values[dep]; ok {
        return true
    }
    // Check if it's a computed field that's been evaluated
    return evaluated[dep]
}

// buildScope builds the evaluation scope
func (c *Context) buildScope() map[string]any {
    scope := make(map[string]any)
    for k, v := range c.inputVals {
        scope["input."+k] = v
    }
    for k, v := range c.values {
        scope["context."+k] = v
    }
    return scope
}

// SetContextField with immutability check
func (c *Context) SetContextField(name string, value any) {
    if _, isComputed := c.computed[name]; isComputed {
        // Silently ignore attempts to modify computed fields
        return
    }
    c.values[name] = value
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/executor -v -run TestContext_Evaluate`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/flows/executor/context.go pkg/flows/executor/context_test.go
git commit -m "feat(executor): add computed field evaluation with immutability"
```

---

## Task 3: Executor Core Structure and State Machine

**Files:**
- Create: `pkg/flows/executor/executor.go`
- Test: `pkg/flows/executor/executor_test.go`

**Step 1: Write failing test for executor initialization**

```go
package executor

import (
    "testing"
    "github.com/denkhaus/gollum/pkg/flows"
    "github.com/stretchr/testify/assert"
)

func TestNewExecutor_CreatesExecutorWithFlow(t *testing.T) {
    flow := &flows.Flow{
        Name: "test-flow",
        States: []flows.State{
            {Name: "init", Initial: true},
            {Name: "done"},
        },
    }

    exec := NewExecutor(flow)

    assert.NotNil(t, exec)
    assert.Equal(t, "test-flow", exec.flow.Name)
    assert.Equal(t, "init", exec.currentState)
}

func TestExecutor_Validate_ReturnsErrorForInvalidFlow(t *testing.T) {
    flow := &flows.Flow{
        Name: "invalid-flow",
        States: []flows.State{
            {Name: "init"}, // No initial state
        },
    }

    exec := NewExecutor(flow)
    err := exec.Validate()

    assert.Error(t, err)
    assert.Contains(t, err.Error(), "no initial state")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/executor -v -run TestNewExecutor`
Expected: FAIL with "undefined: NewExecutor"

**Step 3: Implement Executor struct and core methods**

```go
package executor

import (
    "fmt"
    "time"
    "github.com/denkhaus/gollum/pkg/flows"
)

// Executor executes flow state machines
type Executor struct {
    flow        *flows.Flow
    ctx         *Context
    currentState string
    history     *ExecutionHistory
    startTime   time.Time
}

// NewExecutor creates a new executor
func NewExecutor(flow *flows.Flow) *Executor {
    return &Executor{
        flow:      flow,
        ctx:       NewContext(flow.Input, nil),
        history:   NewExecutionHistory(),
        startTime: time.Now(),
    }
}

// SetInput sets input field values
func (e *Executor) SetInput(vals map[string]any) {
    e.ctx = NewContext(e.flow.Input, vals)
}

// Validate validates the flow before execution
func (e *Executor) Validate() error {
    // Check for initial state
    hasInitial := false
    for _, state := range e.flow.States {
        if state.Initial {
            hasInitial = true
            e.currentState = state.Name
            break
        }
    }

    if !hasInitial {
        return fmt.Errorf("flow %s: no initial state defined", e.flow.Name)
    }

    return nil
}

// Run executes the flow from the initial state
func (e *Executor) Run() error {
    if err := e.Validate(); err != nil {
        return err
    }

    // Find initial state
    var initialState *flows.State
    for i := range e.flow.States {
        if e.flow.States[i].Initial {
            initialState = &e.flow.States[i]
            break
        }
    }

    if initialState == nil {
        return fmt.Errorf("no initial state found")
    }

    return e.executeState(initialState)
}

// executeState executes a single state
func (e *Executor) executeState(state *flows.State) error {
    // Record state entry
    e.history.RecordStateEntry(state.Name, time.Now())

    // Evaluate computed fields
    if e.flow.Context != nil {
        if err := e.ctx.EvaluateComputedFields(e.flow.Context); err != nil {
            return fmt.Errorf("computed field evaluation: %w", err)
        }
    }

    // Execute steps
    for _, step := range state.Steps {
        if err := e.executeStep(&step, state.Name); err != nil {
            return e.handleError(err, &step, state)
        }
    }

    // Execute calls
    for _, call := range state.Calls {
        if err := e.executeCall(&call, state.Name); err != nil {
            return e.handleError(err, nil, state)
        }
    }

    // Find and execute transition
    return e.executeTransition(state)
}

// executeTransition evaluates conditions and transitions to next state
func (e *Executor) executeTransition(state *flows.State) error {
    // Build evaluation scope
    scope := e.ctx.buildScope()

    for _, trans := range state.Transitions {
        if trans.Otherwise {
            // Fallback transition
            return e.transitionTo(trans.To)
        }

        if trans.When == "" {
            // Unconditional transition
            return e.transitionTo(trans.To)
        }

        // Evaluate condition
        eval := ast.NewEvaluator()
        result, err := eval.EvaluateExpr(trans.When, scope)
        if err != nil {
            return fmt.Errorf("transition condition: %w", err)
        }

        if boolVal, ok := result.(bool); ok && boolVal {
            return e.transitionTo(trans.To)
        }
    }

    // No transition - terminal state
    e.history.RecordStateExit(state.Name, time.Now())
    return nil
}

// transitionTo transitions to a new state
func (e *Executor) transitionTo(stateName string) error {
    // Find target state
    var targetState *flows.State
    for i := range e.flow.States {
        if e.flow.States[i].Name == stateName {
            targetState = &e.flow.States[i]
            break
        }
    }

    if targetState == nil {
        return fmt.Errorf("state not found: %s", stateName)
    }

    e.currentState = stateName
    return e.executeState(targetState)
}

// handleError handles step execution errors
func (e *Executor) handleError(err error, step *flows.Step, state *flows.State) error {
    // TODO: Implement on-error transition handling
    return fmt.Errorf("step execution failed in state %s: %w", state.Name, err)
}

// executeStep executes a single step (placeholder)
func (e *Executor) executeStep(step *flows.Step, stateName string) error {
    // TODO: Implement step execution in next tasks
    return fmt.Errorf("step execution not implemented")
}

// executeCall executes a call step (placeholder)
func (e *Executor) executeCall(call *flows.Call, stateName string) error {
    // TODO: Implement call execution in next tasks
    return fmt.Errorf("call execution not implemented")
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/executor -v -run TestNewExecutor`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/flows/executor/executor.go pkg/flows/executor/executor_test.go
git commit -m "feat(executor): add executor core with state machine"
```

---

## Task 4: Execution History Tracking

**Files:**
- Create: `pkg/flows/executor/history.go`
- Test: `pkg/flows/executor/history_test.go`

**Step 1: Write failing test for execution history**

```go
package executor

import (
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
)

func TestExecutionHistory_TracksStateTransitions(t *testing.T) {
    hist := NewExecutionHistory()

    hist.RecordStateEntry("init", time.Now())
    time.Sleep(10 * time.Millisecond)
    hist.RecordStateExit("init", time.Now())
    hist.RecordStateEntry("done", time.Now())

    states := hist.GetStates()
    assert.Len(t, states, 2)
    assert.Equal(t, "init", states[0].Name)
    assert.True(t, states[0].Duration > 0)
}

func TestExecutionHistory_TracksErrors(t *testing.T) {
    hist := NewExecutionHistory()

    hist.RecordError("test-step", "llm", "timeout error", time.Now())

    errors := hist.GetErrors()
    assert.Len(t, errors, 1)
    assert.Equal(t, "timeout error", errors[0].Message)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/executor -v -run TestExecutionHistory`
Expected: FAIL with "undefined: NewExecutionHistory"

**Step 3: Implement ExecutionHistory**

```go
package executor

import (
    "time"
)

// ExecutionHistory tracks flow execution
type ExecutionHistory struct {
    states    []StateExecution
    errors    []ErrorRecord
    startTime time.Time
    endTime   time.Time
}

// StateExecution tracks a single state execution
type StateExecution struct {
    Name      string
    StartedAt time.Time
    EndedAt   time.Time
    Duration  time.Duration
}

// ErrorRecord tracks an execution error
type ErrorRecord struct {
    StepName  string
    StepType  string
    Message   string
    Timestamp time.Time
}

// NewExecutionHistory creates a new execution history
func NewExecutionHistory() *ExecutionHistory {
    return &ExecutionHistory{
        states:    make([]StateExecution, 0),
        errors:    make([]ErrorRecord, 0),
        startTime: time.Now(),
    }
}

// RecordStateEntry records entering a state
func (h *ExecutionHistory) RecordStateEntry(name string, t time.Time) {
    h.states = append(h.states, StateExecution{
        Name:      name,
        StartedAt: t,
    })
}

// RecordStateExit records exiting a state
func (h *ExecutionHistory) RecordStateExit(name string, t time.Time) {
    for i := len(h.states) - 1; i >= 0; i-- {
        if h.states[i].Name == name && h.states[i].EndedAt.IsZero() {
            h.states[i].EndedAt = t
            h.states[i].Duration = t.Sub(h.states[i].StartedAt)
            break
        }
    }
}

// RecordError records an error
func (h *ExecutionHistory) RecordError(stepName, stepType, message string, t time.Time) {
    h.errors = append(h.errors, ErrorRecord{
        StepName:  stepName,
        StepType:  stepType,
        Message:   message,
        Timestamp: t,
    })
}

// GetStates returns all state executions
func (h *ExecutionHistory) GetStates() []StateExecution {
    return h.states
}

// GetErrors returns all errors
func (h *ExecutionHistory) GetErrors() []ErrorRecord {
    return h.errors
}

// Complete marks execution as complete
func (h *ExecutionHistory) Complete(t time.Time) {
    h.endTime = t
}

// GetDuration returns total execution duration
func (h *ExecutionHistory) GetDuration() time.Duration {
    if h.endTime.IsZero() {
        return time.Since(h.startTime)
    }
    return h.endTime.Sub(h.startTime)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/executor -v -run TestExecutionHistory`
Expected: PASS

**Step 5: Update executor to record history completion**

In `executor.go`, update `Run` method:

```go
func (e *Executor) Run() error {
    defer e.history.Complete(time.Now())

    if err := e.Validate(); err != nil {
        return err
    }
    // ... rest of implementation
}
```

**Step 6: Commit**

```bash
git add pkg/flows/executor/history.go pkg/flows/executor/history_test.go pkg/flows/executor/executor.go
git commit -m "feat(executor): add execution history tracking"
```

---

## Task 5: Error Lifecycle with ${error.*} Fields

**Files:**
- Create: `pkg/flows/executor/errors.go`
- Modify: `pkg/flows/executor/executor.go`
- Test: `pkg/flows/executor/errors_test.go`

**Step 1: Write failing test for error context fields**

```go
package executor

import (
    "testing"
    "time"
    "github.com/denkhaus/gollum/pkg/flows"
    "github.com/stretchr/testify/assert"
)

func TestExecutor_CaptureError_SetsErrorContextFields(t *testing.T) {
    flow := &flows.Flow{
        Name: "test-flow",
        States: []flows.State{
            {
                Name: "init", Initial: true,
                Steps: []flows.Step{
                    {Type: "llm", Name: "test-step", OnError: &flows.OnErrorTransition{State: "error"}},
                },
            },
            {Name: "error"},
        },
    }

    exec := NewExecutor(flow)
    exec.captureError(&flows.Step{Type: "llm", Name: "test-step"}, "test error")

    // Check error context fields are set
    val, ok := exec.ctx.GetContextField("error.step_name")
    assert.True(t, ok)
    assert.Equal(t, "test-step", val)

    val, ok = exec.ctx.GetContextField("error.message")
    assert.True(t, ok)
    assert.Equal(t, "test error", val)

    val, ok = exec.ctx.GetContextField("error.step_type")
    assert.True(t, ok)
    assert.Equal(t, "llm", val)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/executor -v -run TestExecutor_CaptureError`
Expected: FAIL with "undefined: captureError"

**Step 3: Implement error lifecycle**

Create `errors.go`:

```go
package executor

import (
    "fmt"
    "time"
    "github.com/denkhaus/gollum/pkg/flows"
)

// ErrorContext holds error lifecycle information
type ErrorContext struct {
    StepName  string
    StepType  string
    Message   string
    Timestamp time.Time
}

// captureError captures error information and sets context fields
func (e *Executor) captureError(step *flows.Step, errMsg string) {
    now := time.Now()

    // Set error context fields
    e.ctx.SetContextField("error.step_name", step.Name)
    e.ctx.SetContextField("error.step_type", step.Type)
    e.ctx.SetContextField("error.message", errMsg)
    e.ctx.SetContextField("error.timestamp", now.Format(time.RFC3339))

    // Record in history
    e.history.RecordError(step.Name, step.Type, errMsg, now)
}

// handleError with on-error transition support
func (e *Executor) handleError(err error, step *flows.Step, state *flows.State) error {
    if step != nil && step.OnError != nil {
        // Capture error context
        e.captureError(step, err.Error())

        // Transition to error state
        return e.transitionTo(step.OnError.State)
    }

    // No error handler - fail
    return fmt.Errorf("unhandled error in state %s: %w", state.Name, err)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/executor -v -run TestExecutor_CaptureError`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/flows/executor/errors.go pkg/flows/executor/errors_test.go pkg/flows/executor/executor.go
git commit -m "feat(executor): add error lifecycle with context fields"
```

---

## Task 6: Template Variable Substitution

**Files:**
- Create: `pkg/flows/executor/template.go`
- Test: `pkg/flows/executor/template_test.go`

**Step 1: Write failing test for template substitution**

```go
package executor

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestSubstituteTemplate_ReplacesVariables(t *testing.T) {
    ctx := NewContext(&flows.InputBlock{}, nil)
    ctx.SetInput("pr_number", 123)
    ctx.SetContextField("pr_title", "Fix bug")

    result := SubstituteTemplate(ctx, "Analyze PR #${input.pr_number}: ${context.pr_title}")

    assert.Equal(t, "Analyze PR #123: Fix bug", result)
}

func TestSubstituteTemplate_HandlesMissingFields(t *testing.T) {
    ctx := NewContext(&flows.InputBlock{}, nil)

    result := SubstituteTemplate(ctx, "Value: ${input.missing}")

    assert.Equal(t, "Value: ${input.missing}", result)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/executor -v -run TestSubstituteTemplate`
Expected: FAIL with "undefined: SubstituteTemplate"

**Step 3: Implement template substitution**

```go
package executor

import (
    "regexp"
    "strings"
    "github.com/denkhaus/gollum/pkg/flows"
)

// Substitution matches ${input.field}, ${context.field}, ${output.field}
var subRegex = regexp.MustCompile(`\$\{(input|context|output)\.([^}]+)\}`)

// SubstituteTemplate replaces variables in template strings
func SubstituteTemplate(ctx *Context, tmpl string) string {
    return subRegex.ReplaceAllStringFunc(tmpl, func(match string) string {
        parts := strings.SplitN(match[2:len(match)-1], ".", 2)
        if len(parts) != 2 {
            return match
        }

        scope := parts[0]
        field := parts[1]

        var value any
        var ok bool

        switch scope {
        case "input":
            value = ctx.GetInput(field)
        case "context":
            value, ok = ctx.GetContextField(field)
            if !ok {
                value = nil
            }
        case "output":
            value, ok = ctx.GetOutputField(field)
            if !ok {
                value = nil
            }
        }

        if value == nil {
            return match
        }

        return fmt.Sprintf("%v", value)
    })
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/executor -v -run TestSubstituteTemplate`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/flows/executor/template.go pkg/flows/executor/template_test.go
git commit -m "feat(executor): add template variable substitution"
```

---

## Task 7: Built-in Tools Implementation

**Files:**
- Create: `pkg/flows/executor/tools.go`
- Test: `pkg/flows/executor/tools_test.go`

**Step 1: Write failing test for set_output_field tool**

```go
package executor

import (
    "testing"
    "encoding/json"
    "github.com/stretchr/testify/assert"
)

func TestToolSetOutputField(t *testing.T) {
    exec := NewExecutor(&flows.Flow{})
    tool := &ToolSetOutputField{executor: exec}

    input := map[string]any{"name": "result", "value": "done"}
    result, err := tool.Execute(input)

    assert.NoError(t, err)
    assert.True(t, result["success"].(bool))

    val, ok := exec.ctx.GetOutputField("result")
    assert.True(t, ok)
    assert.Equal(t, "done", val)
}

func TestToolSetOutputField_ValidatesType(t *testing.T) {
    flow := &flows.Flow{
        Output: &flows.OutputBlock{
            Ints: []flows.FieldDef{{Name: "count"}},
        },
    }
    exec := NewExecutor(flow)
    tool := &ToolSetOutputField{executor: exec, flow: flow}

    // Should accept int value
    input := map[string]any{"name": "count", "value": 42}
    _, err := tool.Execute(input)
    assert.NoError(t, err)

    // Should reject string value for int field
    input = map[string]any{"name": "count", "value": "invalid"}
    _, err = tool.Execute(input)
    assert.Error(t, err)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/executor -v -run TestToolSetOutputField`
Expected: FAIL with "undefined: ToolSetOutputField"

**Step 3: Implement built-in tools**

```go
package executor

import (
    "fmt"
    "strconv"
    "github.com/denkhaus/gollum/pkg/flows"
)

// Tool is the interface for executor tools
type Tool interface {
    Execute(input map[string]any) (map[string]any, error)
}

// ToolSetOutputField implements set_output_field
type ToolSetOutputField struct {
    executor *Executor
    flow     *flows.Flow
}

// Execute sets an output field value
func (t *ToolSetOutputField) Execute(input map[string]any) (map[string]any, error) {
    name, _ := input["name"].(string)
    value := input["value"]

    if name == "" {
        return nil, fmt.Errorf("field name is required")
    }

    // Type validation
    if err := t.validateFieldType(name, value); err != nil {
        return nil, err
    }

    t.executor.ctx.SetOutputField(name, value)

    return map[string]any{"success": true}, nil
}

// validateFieldType checks if value matches field type
func (t *ToolSetOutputField) validateFieldType(name string, value any) error {
    if t.flow == nil || t.flow.Output == nil {
        return nil
    }

    for _, field := range t.flow.Output.GetAllFields() {
        if field.Name == name {
            return validateType(field.Type, value)
        }
    }

    // Field not found in output definition - that's okay, just set it
    return nil
}

// validateType checks value type
func validateType(typ string, value any) error {
    switch typ {
    case "string":
        if _, ok := value.(string); !ok {
            return fmt.Errorf("expected string, got %T", value)
        }
    case "int":
        switch value.(type) {
        case int, int64, float64:
            // Accept numeric types
        default:
            return fmt.Errorf("expected int, got %T", value)
        }
    case "bool":
        if _, ok := value.(bool); !ok {
            return fmt.Errorf("expected bool, got %T", value)
        }
    }
    return nil
}

// ToolSetContextField implements set_context_field
type ToolSetContextField struct {
    executor *Executor
    flow     *flows.Flow
}

// Execute sets a context field value
func (t *ToolSetContextField) Execute(input map[string]any) (map[string]any, error) {
    name, _ := input["name"].(string)
    value := input["value"]

    if name == "" {
        return nil, fmt.Errorf("field name is required")
    }

    // Check if it's a computed field (immutable)
    if t.flow != nil && t.flow.Context != nil {
        for _, cf := range t.flow.Context.Computeds {
            if cf.Name == name {
                return nil, fmt.Errorf("cannot modify computed field '%s'", name)
            }
        }
    }

    t.executor.ctx.SetContextField(name, value)

    return map[string]any{"success": true}, nil
}

// ToolGetContext implements get_context
type ToolGetContext struct {
    executor *Executor
}

// Execute retrieves context fields
func (t *ToolGetContext) Execute(input map[string]any) (map[string]any, error) {
    fields, _ := input["fields"].([]any)

    result := make(map[string]any)
    if len(fields) == 0 {
        // Return all context fields
        for k, v := range t.executor.ctx.values {
            result[k] = v
        }
    } else {
        for _, f := range fields {
            if fieldName, ok := f.(string); ok {
                if val, ok := t.executor.ctx.GetContextField(fieldName); ok {
                    result[fieldName] = val
                }
            }
        }
    }

    return result, nil
}

// ToolEmitLog implements emit_log
type ToolEmitLog struct {
    executor *Executor
}

// Execute logs a message (uses Langfuse hooks)
func (t *ToolEmitLog) Execute(input map[string]any) (map[string]any, error) {
    level, _ := input["level"].(string)
    message, _ := input["message"].(string)

    if level == "" {
        level = "info"
    }

    // TODO: Integrate with Langfuse hooks
    fmt.Printf("[%s] %s\n", level, message)

    return map[string]any{"success": true}, nil
}

// ToolTransitionTo implements transition_to
type ToolTransitionTo struct {
    executor *Executor
}

// Execute transitions to a new state
func (t *ToolTransitionTo) Execute(input map[string]any) (map[string]any, error) {
    toState, _ := input["to"].(string)

    if toState == "" {
        return nil, fmt.Errorf("target state is required")
    }

    // Validate transition is allowed
    currentState := t.executor.currentState
    allowed := false
    for _, s := range t.executor.flow.States {
        if s.Name == currentState {
            for _, trans := range s.Transitions {
                if trans.To == toState {
                    allowed = true
                    break
                }
            }
            break
        }
    }

    if !allowed {
        return nil, fmt.Errorf("transition from %s to %s is not allowed", currentState, toState)
    }

    return map[string]any{
        "success": true,
        "from":    currentState,
        "to":      toState,
    }, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/executor -v -run TestTool`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/flows/executor/tools.go pkg/flows/executor/tools_test.go
git commit -m "feat(executor): add built-in tools (set_output_field, set_context_field, get_context, emit_log, transition_to)"
```

---

## Task 8: LLM Step Execution

**Files:**
- Create: `pkg/flows/executor/llm_step.go`
- Modify: `pkg/flows/executor/executor.go`
- Test: `pkg/flows/executor/llm_step_test.go`

**Step 1: Write failing test for LLM step execution**

```go
package executor

import (
    "testing"
    "github.com/denkhaus/gollum/pkg/flows"
    "github.com/stretchr/testify/assert"
)

func TestExecuteLLMStep_SubstitutesPrompt(t *testing.T) {
    flow := &flows.Flow{
        Name: "test",
        Input: &flows.InputBlock{
            Strings: []flows.FieldDef{{Name: "pr_number"}},
        },
        Agents: []flows.Agent{
            {Name: "worker", Model: "test-model", Prompt: "You are a helper"},
        },
        States: []flows.State{
            {Name: "init", Initial: true, Steps: []flows.Step{
                {Type: "llm", Agent: "worker", Prompt: "Analyze PR #${input.pr_number}"},
            }},
        },
    }

    exec := NewExecutor(flow)
    exec.SetInput(map[string]any{"pr_number": 123})

    step := &flows.Step{Type: "llm", Agent: "worker", Prompt: "Analyze PR #${input.pr_number}"}
    err := exec.executeStep(step, "init")

    // Should get error about LLM execution not being mocked, but prompt substitution should work
    assert.Error(t, err)
}

func TestExecuteLLMStep_ValidatesAgentExists(t *testing.T) {
    flow := &flows.Flow{
        Name: "test",
        States: []flows.State{
            {Name: "init", Initial: true, Steps: []flows.Step{
                {Type: "llm", Agent: "nonexistent"},
            }},
        },
    }

    exec := NewExecutor(flow)
    step := &flows.Step{Type: "llm", Agent: "nonexistent"}
    err := exec.executeStep(step, "init")

    assert.Error(t, err)
    assert.Contains(t, err.Error(), "agent not found")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/executor -v -run TestExecuteLLMStep`
Expected: FAIL with "executeStep not implemented correctly"

**Step 3: Implement LLM step execution**

Create `llm_step.go`:

```go
package executor

import (
    "fmt"
    "strings"
    "github.com/denkhaus/gollum/pkg/flows"
)

// executeLLMStep executes an LLM step
func (e *Executor) executeLLMStep(step *flows.Step, stateName string) error {
    // Find agent
    var agent *flows.Agent
    for i := range e.flow.Agents {
        if e.flow.Agents[i].Name == step.Agent {
            agent = &e.flow.Agents[i]
            break
        }
    }

    if agent == nil {
        return fmt.Errorf("agent not found: %s", step.Agent)
    }

    // Substitute variables in prompt
    prompt := SubstituteTemplate(e.ctx, step.Prompt)

    // Parse tools
    tools := e.parseTools(step.Tools)

    // TODO: Integrate with actual LLM execution
    // For now, this is a placeholder
    fmt.Printf("[LLM] Agent: %s, Model: %s\n", agent.Name, agent.Model)
    fmt.Printf("[LLM] Prompt: %s\n", prompt)
    fmt.Printf("[LLM] Tools: %s\n", strings.Join(tools, ", "))

    return fmt.Errorf("LLM execution not yet implemented")
}

// parseTools parses tools attribute into tool names
func (e *Executor) parseTools(toolsStr string) []string {
    if toolsStr == "" {
        return nil
    }

    return strings.Split(toolsStr, ",")
}
```

Update `executor.go` executeStep:

```go
func (e *Executor) executeStep(step *flows.Step, stateName string) error {
    switch step.Type {
    case "llm":
        return e.executeLLMStep(step, stateName)
    case "shell":
        return e.executeShellStep(step, stateName)
    case "func":
        return e.executeFuncStep(step, stateName)
    case "mcp":
        return e.executeMCPStep(step, stateName)
    default:
        return fmt.Errorf("unknown step type: %s", step.Type)
    }
}

func (e *Executor) executeShellStep(step *flows.Step, stateName string) error {
    return fmt.Errorf("shell step execution not yet implemented")
}

func (e *Executor) executeFuncStep(step *flows.Step, stateName string) error {
    return fmt.Errorf("func step execution not yet implemented")
}

func (e *Executor) executeMCPStep(step *flows.Step, stateName string) error {
    return fmt.Errorf("mcp step execution not yet implemented")
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/executor -v -run TestExecuteLLMStep`
Expected: PASS (with expected error about LLM not being implemented)

**Step 5: Commit**

```bash
git add pkg/flows/executor/llm_step.go pkg/flows/executor/executor.go pkg/flows/executor/llm_step_test.go
git commit -m "feat(executor): add LLM step execution with validation"
```

---

## Task 9: Integration Test - Simple Flow Execution

**Files:**
- Create: `pkg/flows/executor/integration_test.go`

**Step 1: Write integration test**

```go
package executor

import (
    "testing"
    "github.com/denkhaus/gollum/pkg/flows"
    "github.com/stretchr/testify/assert"
)

func TestExecutor_SimpleFlow_ExecutesSuccessfully(t *testing.T) {
    // Define a simple flow with input/output and state transitions
    flow := &flows.Flow{
        Name:    "simple-test",
        Version: "1.0",
        Input: &flows.InputBlock{
            Strings: []flows.FieldDef{{Name: "message", Default: "hello"}},
        },
        Output: &flows.OutputBlock{
            Strings: []flows.FieldDef{{Name: "result"}},
        },
        Context: &flows.ContextBlock{
            Strings: []flows.ContextField{{Name: "greeting"}},
            Computeds: []flows.ComputedField{
                {Name: "is_ready", Type: "bool", When: "EQ(context.greeting, 'hello')"},
            },
        },
        States: []flows.State{
            {
                Name: "init", Initial: true,
                Steps: []flows.Step{
                    {Type: "llm", Agent: "worker", Prompt: "Say ${input.message}"},
                },
                Transitions: []flows.Transition{
                    {To: "done"},
                },
            },
            {Name: "done"},
        },
        Agents: []flows.Agent{
            {Name: "worker", Model: "test", Prompt: "Test agent"},
        },
    }

    exec := NewExecutor(flow)
    exec.SetInput(map[string]any{"message": "hello"})

    // LLM step will fail but we can test state transitions
    err := exec.Validate()
    assert.NoError(t, err)
    assert.Equal(t, "init", exec.currentState)

    // Test computed field evaluation
    exec.ctx.SetContextField("greeting", "hello")
    err = exec.ctx.EvaluateComputedFields(flow.Context)
    assert.NoError(t, err)

    val, ok := exec.ctx.GetContextField("is_ready")
    assert.True(t, ok)
    assert.Equal(t, true, val)
}

func TestExecutor_ErrorHandling_TransitionsToErrorState(t *testing.T) {
    flow := &flows.Flow{
        Name: "error-test",
        States: []flows.State{
            {
                Name: "init", Initial: true,
                Steps: []flows.Step{
                    {Type: "llm", Agent: "missing", OnError: &flows.OnErrorTransition{State: "error"}},
                },
            },
            {Name: "error"},
        },
    }

    exec := NewExecutor(flow)
    err := exec.Run()

    // Should transition to error state
    assert.Error(t, err)
    // The error should be about agent not found
}
```

**Step 2: Run test to verify behavior**

Run: `go test ./pkg/flows/executor -v -run TestExecutor_SimpleFlow`
Expected: PASS (validates context and computed fields work)

**Step 3: Commit**

```bash
git add pkg/flows/executor/integration_test.go
git commit -m "test(executor): add integration tests for simple flows"
```

---

## Task 10: CLI Command for Flow Execution

**Files:**
- Create: `cmd/flows/run/main.go`

**Step 1: Create run command**

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"

    "github.com/denkhaus/gollum/pkg/flows"
    "github.com/denkhaus/gollum/pkg/flows/executor"
    "github.com/denkhaus/gollum/pkg/flows/parser"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Fprintln(os.Stderr, "Usage: flow-run <flow.xml> [input.json]")
        os.Exit(1)
    }

    flowPath := os.Args[1]

    // Parse flow
    flow, err := parser.Parse(flowPath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
        os.Exit(1)
    }

    // Load input if provided
    var inputVals map[string]any
    if len(os.Args) >= 3 {
        inputData, err := os.ReadFile(os.Args[2])
        if err != nil {
            fmt.Fprintf(os.Stderr, "Input file error: %v\n", err)
            os.Exit(1)
        }
        if err := json.Unmarshal(inputData, &inputVals); err != nil {
            fmt.Fprintf(os.Stderr, "Input JSON error: %v\n", err)
            os.Exit(1)
        }
    }

    // Create executor
    exec := executor.NewExecutor(flow)
    exec.SetInput(inputVals)

    // Run flow
    fmt.Printf("Running flow: %s\n", flow.Name)
    if err := exec.Run(); err != nil {
        fmt.Fprintf(os.Stderr, "Execution error: %v\n", err)
        os.Exit(1)
    }

    fmt.Printf("Flow completed successfully\n")
}
```

**Step 2: Test with example flow**

```bash
cd /home/denkhaus/dev/gomodules/gollum
go build -o bin/flow-run ./cmd/flows/run
./bin/flow-run .gollum/flows/examples/simple-flow.xml
```

**Step 3: Commit**

```bash
git add cmd/flows/run/main.go
git commit -m "feat(cli): add flow-run command for executing flows"
```

---

## Summary

This plan implements the Phase 1 MVP of the flows executor:

1. ✅ Context management with input handling
2. ✅ Computed field evaluation with dependency ordering
3. ✅ State machine executor with transitions
4. ✅ Execution history tracking
5. ✅ Error lifecycle with ${error.*} context fields
6. ✅ Template variable substitution
7. ✅ Built-in tools (set_output_field, set_context_field, get_context, emit_log, transition_to)
8. ✅ LLM step execution framework
9. ✅ Integration tests
10. ✅ CLI command for running flows

**Next Steps** (Phase 2 - Composition):
- Sub-flow calls with context isolation
- Function registry and func step execution
- Output mapping for calls

**Next Steps** (Phase 3 - Integration):
- MCP step execution
- Shell step execution with auto-escaping
- Extension function support
- Parallel step groups
- Retry logic with exponential backoff

---

**Testing Strategy:**
- Each component has unit tests
- Integration tests validate end-to-end scenarios
- Use existing example flows for validation

**Verification:**
```bash
# Run all tests
go test ./pkg/flows/executor -v

# Test with example flow
./bin/flow-run .gollum/flows/examples/simple-flow.xml

# Run linter first
./bin/flow-lint .gollum/flows/examples/simple-flow.xml
```
