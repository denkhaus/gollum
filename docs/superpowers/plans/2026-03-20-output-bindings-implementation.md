# Output Bindings Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add declarative output field bindings using `from` attribute to eliminate verbose `assign` function.

**Architecture:** Output fields can optionally declare `from="scope.field"` for automatic value binding. Declarative outputs (with `from`) are readonly at runtime. Imperative outputs (without `from`) remain mutable.

**Tech Stack:** Go 1.23+, XML parsing, TDD with standard testing

---

## File Structure

### Files to Modify

| File | Changes |
|------|---------|
| `pkg/flows/types.go` | Add `From` field to `FieldDef`, add helpers to `OutputBlock` |
| `pkg/flows/executor/context.go` | Add output binding initialization, readonly enforcement |
| `pkg/flows/linter/linter.go` | Add Phase 8 for output binding validation |
| `pkg/flows/variables/field_values.go` | Add `IsDeclarative` check for output field readonly |

### Files to Create

| File | Purpose |
|------|---------|
| `pkg/flows/errors/output_binding.go` | New error types: `OutputFieldReadOnlyError`, `OutputBindingError` |
| `pkg/flows/linter/output_bindings.go` | Phase 8 linter for output binding validation |
| `pkg/flows/linter/output_bindings_test.go` | Tests for output binding linter |
| `pkg/flows/variables/output_bindings.go` | `OutputBinding` struct and initialization logic |
| `pkg/flows/variables/output_bindings_test.go` | Tests for output bindings |
| `pkg/flows/executor/output_binder.go` | Output binding initialization logic |
| `pkg/flows/executor/output_binder_test.go` | Tests for output binder |

### Files to Refactor (Flow Migration)

| File | Changes |
|------|---------|
| `.gollum/flows/examples/arithmetic-computed.xml` | Remove assign steps, add `from` attributes |
| `.gollum/flows/examples/assign-input-to-output.xml` | Remove assign step, add `from` attribute |
| `.gollum/flows/examples/step-types-example.xml` | Update assign usage |
| `.gollum/flows/examples/nested-context-example.xml` | Update assign usage |
| `.gollum/flows/modules/code-analysis/complexity-check.xml` | Update assign usage |
| `.gollum/flows/modules/code-analysis/security-scan.xml` | Update assign usage |
| `.gollum/flows/forgejo-workflow/fetch-pr.xml` | Update assign usage |
| `.gollum/flows/forgejo-workflow/review-phase.xml` | Update assign usage |
| `test/fixtures/flows/func_step_test.xml` | Update assign usage |

---

## Chunk 1: Schema Changes

### Task 1: Add `From` attribute to `FieldDef`

**Files:**
- Modify: `pkg/flows/types.go:240-247`

- [ ] **Step 1: Write failing test**

```go
// pkg/flows/types_test.go

func TestFieldDef_FromAttribute(t *testing.T) {
    xml := `<string name="result" from="computed.sum" />`
    var field FieldDef
    err := xml.Unmarshal([]byte(xml), &field)
    assert.NoError(t, err)
    assert.Equal(t, "computed.sum", field.From)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows -run TestFieldDef_FromAttribute -v`
Expected: FAIL with "unknown field"

- [ ] **Step 3: Add From field to FieldDef**

```go
// pkg/flows/types.go:240-247

// FieldDef is a base type for field definitions
type FieldDef struct {
    XMLName  xml.Name
    Name     string    `xml:"name,attr"`
    Type     ValueType `xml:"type,attr"`
    Required bool      `xml:"required,attr"`
    Default  string    `xml:"default,attr"`
    From     string    `xml:"from,attr,omitempty"`  // NEW: source reference for bindings
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows -run TestFieldDef_FromAttribute -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/types.go pkg/flows/types_test.go
git commit -m "feat(types): add From attribute to FieldDef for output bindings"
```

---

### Task 2: Add helpers to OutputBlock

**Files:**
- Modify: `pkg/flows/types.go:161-198`

- [ ] **Step 1: Write failing test**

```go
// pkg/flows/types_test.go

func TestOutputBlock_GetDeclarative(t *testing.T) {
    block := &OutputBlock{
        Ints: []FieldDef{
            {Name: "sum", From: "computed.sum"},      // declarative
            {Name: "count"},                          // imperative
        },
    }

    declarative := block.GetDeclarative()
    assert.Len(t, declarative, 1)
    assert.Equal(t, "sum", declarative[0].Name)

    imperative := block.GetImperative()
    assert.Len(t, imperative, 1)
    assert.Equal(t, "count", imperative[0].Name)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows -run TestOutputBlock_GetDeclarative -v`
Expected: FAIL with "method GetDeclarative does not exist"

- [ ] **Step 3: Add helper methods to OutputBlock**

```go
// pkg/flows/types.go - add after GetAllFields method (around line 198)

// GetDeclarative returns all output fields with a 'from' attribute
func (o *OutputBlock) GetDeclarative() []FieldDef {
    var fields []FieldDef
    for _, f := range o.GetAllFields() {
        if f.From != "" {
            fields = append(fields, f)
        }
    }
    return fields
}

// GetImperative returns all output fields without a 'from' attribute
func (o *OutputBlock) GetImperative() []FieldDef {
    var fields []FieldDef
    for _, f := range o.GetAllFields() {
        if f.From == "" {
            fields = append(fields, f)
        }
    }
    return fields
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows -run TestOutputBlock_GetDeclarative -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/types.go pkg/flows/types_test.go
git commit -m "feat(types): add GetDeclarative/GetImperative helpers to OutputBlock"
```

---

## Chunk 2: Error Types

### Task 3: Create output binding error types

**Files:**
- Create: `pkg/flows/errors/output_binding.go`
- Modify: `pkg/flows/errors/codes.go`

- [ ] **Step 1: Write failing test**

```go
// pkg/flows/errors/output_binding_test.go

package errors

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestOutputFieldReadOnlyError(t *testing.T) {
    err := &OutputFieldReadOnlyError{
        FieldName: "sum",
        Source:    "computed.sum",
    }

    assert.Contains(t, err.Error(), "sum")
    assert.Contains(t, err.Error(), "computed.sum")
    assert.Contains(t, err.Error(), "readonly")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/errors -run TestOutputFieldReadOnlyError -v`
Expected: FAIL with "undefined type"

- [ ] **Step 3: Create error types**

```go
// pkg/flows/errors/output_binding.go

package errors

// OutputFieldReadOnlyError is returned when attempting to set a declarative output field at runtime
type OutputFieldReadOnlyError struct {
    FlowError
    FieldName string
    Source    string // the 'from' source reference
}

func (e *OutputFieldReadOnlyError) Error() string {
    return e.FlowError.Error()
}

// OutputBindingError is returned when output binding initialization fails
type OutputBindingError struct {
    FlowError
    OutputName string
    Source     string // the source that failed
    Cause      error  // underlying error
}

func (e *OutputBindingError) Error() string {
    if e.Cause != nil {
        return e.FlowError.Error()
    }
    return e.FlowError.Error()
}

func (e *OutputBindingError) Unwrap() error {
    return e.Cause
}
```

- [ ] **Step 4: Add error codes**

```go
// pkg/flows/errors/codes.go - add to existing constants

const (
    // ... existing codes ...

    // Output binding errors
    ErrCodeOutputFieldReadOnly = "OUTPUT_FIELD_READONLY"
    ErrCodeOutputBinding       = "OUTPUT_BINDING_ERROR"
)
```

- [ ] **Step 5: Initialize error messages**

```go
// pkg/flows/errors/output_binding.go - update with proper messages

func NewOutputFieldReadOnlyError(fieldName, source string) *OutputFieldReadOnlyError {
    return &OutputFieldReadOnlyError{
        FlowError: FlowError{
            Code:    ErrCodeOutputFieldReadOnly,
            Message: sprintf("output field '%s' has a 'from' attribute and cannot be set at runtime (value flows from: %s)", fieldName, source),
            Field:   fieldName,
        },
        FieldName: fieldName,
        Source:    source,
    }
}

func NewOutputBindingError(outputName, source string, cause error) *OutputBindingError {
    return &OutputBindingError{
        FlowError: FlowError{
            Code:    ErrCodeOutputBinding,
            Message: sprintf("failed to bind output '%s' from source '%s': %v", outputName, source, cause),
            Field:   outputName,
            Cause:   cause,
        },
        OutputName: outputName,
        Source:     source,
        Cause:      cause,
    }
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./pkg/flows/errors -run TestOutputFieldReadOnlyError -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add pkg/flows/errors/output_binding.go pkg/flows/errors/output_binding_test.go pkg/flows/errors/codes.go
git commit -m "feat(errors): add output binding error types"
```

---

## Chunk 3: Output Bindings in Variables Package

### Task 4: Create OutputBinding struct

**Files:**
- Create: `pkg/flows/variables/output_bindings.go`
- Create: `pkg/flows/variables/output_bindings_test.go`

- [ ] **Step 1: Write failing test**

```go
// pkg/flows/variables/output_bindings_test.go

package variables

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/denkhaus/gollum/pkg/flows"
)

func TestOutputBinding(t *testing.T) {
    binding := OutputBinding{
        TargetName:  "result",
        SourceScope: "computed",
        SourceName:  "sum",
    }

    assert.Equal(t, "result", binding.TargetName)
    assert.Equal(t, "computed", binding.SourceScope)
    assert.Equal(t, "sum", binding.SourceName)
}

func TestParseFieldReference(t *testing.T) {
    scope, name, err := ParseFieldReference("computed.sum")
    assert.NoError(t, err)
    assert.Equal(t, "computed", scope)
    assert.Equal(t, "sum", name)

    _, _, err = ParseFieldReference("invalid")
    assert.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/variables -run TestOutputBinding -v`
Expected: FAIL with "undefined type"

- [ ] **Step 3: Implement OutputBinding and parser**

```go
// pkg/flows/variables/output_bindings.go

package variables

import (
    "fmt"
    "strings"

    "github.com/denkhaus/gollum/pkg/flows"
)

// OutputBinding represents a declarative output field binding
type OutputBinding struct {
    TargetName  string         // output field name
    SourceScope string         // "input", "context", "computed", "output"
    SourceName  string         // source field name
    Value       flows.ValueType // type of the value
}

// ParseFieldReference parses a field reference like "computed.sum" into scope and name
func ParseFieldReference(ref string) (scope string, name string, err error) {
    parts := strings.SplitN(ref, ".", 2)
    if len(parts) != 2 {
        return "", "", fmt.Errorf("invalid field reference format: '%s' (expected 'scope.fieldname')", ref)
    }

    scope = parts[0]
    name = parts[1]

    if scope == "" || name == "" {
        return "", "", fmt.Errorf("invalid field reference: '%s' (empty scope or name)", ref)
    }

    return scope, name, nil
}

// IsValidSourceScope checks if the scope is valid for output bindings
func IsValidSourceScope(scope string) bool {
    switch scope {
    case "input", "context", "computed", "output":
        return true
    default:
        return false
    }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/flows/variables -run TestOutputBinding -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/variables/output_bindings.go pkg/flows/variables/output_bindings_test.go
git commit -m "feat(variables): add OutputBinding struct and field reference parser"
```

---

### Task 5: Add IsDeclarative check to FieldValues

**Files:**
- Modify: `pkg/flows/variables/field_values.go`

- [ ] **Step 1: Write failing test**

```go
// pkg/flows/variables/field_values_test.go

func TestFieldValues_IsDeclarative(t *testing.T) {
    defs := []flows.FieldDef{
        {Name: "sum", From: "computed.sum", Type: flows.TypeInt},
        {Name: "count", Type: flows.TypeInt},
    }

    fv := NewFieldValues(defs)

    // Note: This will fail initially - we need to track declarative fields
    // The actual implementation will be in context.go which has access to OutputBlock
}
```

- [ ] **Step 2: Skip this test for now**

The `IsDeclarative` check needs to be in `context.go` which has access to the `OutputBlock`. We'll implement it in Task 7.

---

## Chunk 4: Executor Changes

### Task 6: Create output binder

**Files:**
- Create: `pkg/flows/executor/output_binder.go`
- Create: `pkg/flows/executor/output_binder_test.go`

- [ ] **Step 1: Write failing test**

```go
// pkg/flows/executor/output_binder_test.go

package executor

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/denkhaus/gollum/pkg/flows"
    "github.com/denkhaus/gollum/pkg/flows/variables"
)

func TestOutputBinder_InitializeBindings(t *testing.T) {
    flow := &flows.Flow{
        Input: &flows.InputBlock{
            Ints: []flows.FieldDef{{Name: "a", Type: flows.TypeInt}},
        },
        Computed: &flows.ComputedBlock{
            Ints: []flows.ComputedFieldDef{
                {Name: "doubled", Type: flows.TypeInt, Eval: "MUL(input.a, 2)"},
            },
        },
        Output: &flows.OutputBlock{
            Ints: []flows.FieldDef{
                {Name: "result", From: "computed.doubled", Type: flows.TypeInt},
            },
        },
    }

    ctx := newContext(flow.Input, flow.Output, flow.Context, map[string]string{"a": "10"})
    ctx.SetComputedBlock(flow.Computed)

    // Evaluate computed fields
    err := ctx.EvaluateComputed()
    assert.NoError(t, err)

    // Initialize output bindings
    binder := NewOutputBinder()
    err = binder.InitializeBindings(ctx, flow.Output)
    assert.NoError(t, err)

    // Verify output has the value
    result, err := ctx.GetOutputField("result")
    assert.NoError(t, err)
    assert.Equal(t, 20, result)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/executor -run TestOutputBinder_InitializeBindings -v`
Expected: FAIL with "undefined type"

- [ ] **Step 3: Implement OutputBinder**

```go
// pkg/flows/executor/output_binder.go

package executor

import (
    "fmt"

    "github.com/denkhaus/gollum/pkg/flows"
    "github.com/denkhaus/gollum/pkg/flows/errors"
    "github.com/denkhaus/gollum/pkg/flows/variables"
)

// OutputBinder handles declarative output field binding initialization
type OutputBinder struct{}

// NewOutputBinder creates a new output binder
func NewOutputBinder() *OutputBinder {
    return &OutputBinder{}
}

// InitializeBindings populates declarative output fields from their sources
func (b *OutputBinder) InitializeBindings(ctx ExecutionContext, outputBlock *flows.OutputBlock) error {
    if outputBlock == nil {
        return nil
    }

    // Get declarative output fields (those with 'from' attribute)
    declarative := outputBlock.GetDeclarative()

    for _, field := range declarative {
        if err := b.initializeBinding(ctx, &field); err != nil {
            return err
        }
    }

    return nil
}

// initializeBinding populates a single declarative output field
func (b *OutputBinder) initializeBinding(ctx ExecutionContext, field *flows.FieldDef) error {
    // Parse the 'from' reference
    sourceScope, sourceName, err := variables.ParseFieldReference(field.From)
    if err != nil {
        return errors.NewOutputBindingError(field.Name, field.From, err)
    }

    // Validate scope
    if !variables.IsValidSourceScope(sourceScope) {
        return errors.NewOutputBindingError(field.Name, field.From,
            fmt.Errorf("invalid scope '%s' (allowed: input, context, computed, output)", sourceScope))
    }

    // Get the value from the source
    var value any
    switch sourceScope {
    case "computed":
        value, err = ctx.GetComputedField(sourceName)
    case "input":
        value = ctx.GetInput(sourceName)
        if value == nil {
            return errors.NewOutputBindingError(field.Name, field.From,
                fmt.Errorf("input field not set: %s", sourceName))
        }
    case "context":
        value, err = ctx.GetContextField(sourceName)
    case "output":
        value, err = ctx.GetOutputField(sourceName)
    default:
        return errors.NewOutputBindingError(field.Name, field.From,
            fmt.Errorf("unsupported scope: %s", sourceScope))
    }

    if err != nil {
        return errors.NewOutputBindingError(field.Name, field.From, err)
    }

    // Set the output value
    return ctx.SetOutputField(field.Name, value)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/flows/executor -run TestOutputBinder_InitializeBindings -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/executor/output_binder.go pkg/flows/executor/output_binder_test.go
git commit -m "feat(executor): add OutputBinder for declarative output initialization"
```

---

### Task 7: Add readonly enforcement to SetOutputField

**Files:**
- Modify: `pkg/flows/executor/context.go:321-360`

- [ ] **Step 1: Write failing test**

```go
// pkg/flows/executor/context_test.go

func TestContextImpl_SetOutputField_DeclarativeReadonly(t *testing.T) {
    flow := &flows.Flow{
        Output: &flows.OutputBlock{
            Ints: []flows.FieldDef{
                {Name: "sum", From: "computed.sum", Type: flows.TypeInt},
                {Name: "count", Type: flows.TypeInt},
            },
        },
    }

    ctx := newContext(flow.Input, flow.Output, flow.Context, nil)

    // Attempting to set a declarative field should fail
    err := ctx.SetOutputField("sum", 42)
    assert.Error(t, err)

    var readOnlyErr *errors.OutputFieldReadOnlyError
    assert.ErrorAs(t, err, &readOnlyErr)
    assert.Equal(t, "sum", readOnlyErr.FieldName)
    assert.Equal(t, "computed.sum", readOnlyErr.Source)

    // Setting an imperative field should succeed
    err = ctx.SetOutputField("count", 100)
    assert.NoError(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/executor -run TestContextImpl_SetOutputField_DeclarativeReadonly -v`
Expected: FAIL with "no error returned" (readonly check not implemented)

- [ ] **Step 3: Add readonly check to SetOutputField**

```go
// pkg/flows/executor/context.go - modify SetOutputField method (around line 321)

// SetOutputField sets the output variable by name
// Returns error if the field is not defined or is a declarative (readonly) field
func (c *contextImpl) SetOutputField(name string, value any) error {
    if c.outputValues == nil {
        return &errors.NoSchemaError{
            FlowError: errors.FlowError{
                Code:    errors.ErrCodeUnknownField,
                Message: "output schema not defined",
            },
            Scope: "output",
        }
    }

    // Check if field exists
    if !c.outputValues.Has(name) {
        return &errors.UnknownFieldError{
            FlowError: errors.FlowError{
                Code:    errors.ErrCodeUnknownField,
                Message: "field not defined",
                Field:   name,
            },
            Scope: "output",
        }
    }

    // NEW: Check if field is declarative (readonly)
    if c.outputBlock != nil {
        for _, field := range c.outputBlock.GetAllFields() {
            if field.Name == name && field.From != "" {
                return errors.NewOutputFieldReadOnlyError(name, field.From)
            }
        }
    }

    // Convert value based on its actual type
    switch v := value.(type) {
    case string:
        return c.outputValues.SetFromString(name, v)
    case int:
        return c.outputValues.SetInt(name, v)
    case bool:
        return c.outputValues.SetBool(name, v)
    case float64:
        return c.outputValues.SetFloat(name, v)
    default:
        // Try to convert to string as fallback
        return c.outputValues.SetFromString(name, fmt.Sprintf("%v", v))
    }
}
```

- [ ] **Step 4: Update imports**

```go
// pkg/flows/executor/context.go - add to imports

import (
    "context"
    "fmt"
    "strconv"
    "strings"
    "time"

    "github.com/denkhaus/gollum/pkg/extensions"
    "github.com/denkhaus/gollum/pkg/flows"
    flowregistry "github.com/denkhaus/gollum/pkg/flows/registry"
    "github.com/denkhaus/gollum/pkg/flows/errors"  // NEW
    "github.com/denkhaus/gollum/pkg/flows/variables"
    // ... rest of imports
)
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./pkg/flows/executor -run TestContextImpl_SetOutputField_DeclarativeReadonly -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add pkg/flows/executor/context.go pkg/flows/executor/context_test.go
git commit -m "feat(executor): add readonly enforcement for declarative output fields"
```

---

### Task 8: Integrate output binder into executor lifecycle

**Files:**
- Modify: `pkg/flows/executor/executor.go`

- [ ] **Step 1: Write integration test**

```go
// pkg/flows/executor/integration_test.go

func TestFlowExecutor_DeclarativeOutput(t *testing.T) {
    flowXML := `
    <flow name="test" version="1.0">
        <input><int name="a" default="5" /></input>
        <computed><int name="doubled" eval="MUL(input.a, 2)" /></computed>
        <output><int name="result" from="computed.doubled" /></output>
        <states><state name="done" initial="true" /></states>
    </flow>`

    flow, _ := testParseFlow(flowXML)
    service := newTestExecutorService(t)
    executor := service.New(flow)

    err := executor.SetInput(map[string]string{"a": "10"})
    assert.NoError(t, err)

    err = executor.Validate()
    assert.NoError(t, err)

    err = executor.Run()
    assert.NoError(t, err)

    // Verify declarative output was populated
    result, _ := executor.GetContext().GetOutputField("result")
    assert.Equal(t, 20, result)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/executor -run TestFlowExecutor_DeclarativeOutput -v`
Expected: FAIL (output bindings not initialized)

- [ ] **Step 3: Add output binder to executor lifecycle**

```go
// pkg/flows/executor/executor.go - modify flowExecutorImpl struct

type flowExecutorImpl struct {
    flow              *flows.Flow
    ctx               *contextImpl
    currentState      string
    history           *ExecutionHistory
    startTime         time.Time
    logService        logger.LoggerService
    bashToolProvider  tools.BashToolProvider
    extService        extensions.ExtensionService
    flowRegistry      flowregistry.FlowRegistry
    hookManager       hooks.HookManager
    flowToolsProvider tools.FlowToolsProvider
    mcpRegistry       mcpregistry.MCPRegistry
    agentFactory      shared.AgentFactory
    pendingTransition string
    outputBinder      *OutputBinder  // NEW
}
```

```go
// pkg/flows/executor/executor.go - modify New method

func (p *flowExecutorServiceImpl) New(flow *flows.Flow) FlowExecutorInstance {
    ctx := newContext(flow.Input, flow.Output, flow.Context, nil)

    // Initialize computed fields from ComputedBlock
    if flow.Computed != nil {
        ctx.SetComputedBlock(flow.Computed)
    }

    return &flowExecutorImpl{
        flow:              flow,
        ctx:               ctx,
        history:           NewExecutionHistory(),
        startTime:         time.Now(),
        logService:        p.logService,
        bashToolProvider:  p.bashToolProvider,
        extService:        p.extService,
        flowRegistry:      p.flowRegistry,
        hookManager:       p.hookManager,
        flowToolsProvider: p.flowToolsProvider,
        mcpRegistry:       p.mcpRegistry,
        agentFactory:      p.agentFactory,
        outputBinder:      NewOutputBinder(),  // NEW
    }
}
```

```go
// pkg/flows/executor/executor.go - find where EvaluateComputed is called and add output binding after

// In the Run method or state execution, after evaluating computed:
if p.flow.Computed != nil {
    if err := p.ctx.EvaluateComputed(); err != nil {
        return err
    }

    // NEW: Initialize output bindings after computed evaluation
    if err := p.outputBinder.InitializeBindings(p.ctx, p.flow.Output); err != nil {
        return err
    }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/flows/executor -run TestFlowExecutor_DeclarativeOutput -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/executor/executor.go pkg/flows/executor/integration_test.go
git commit -m "feat(executor): integrate output binder into execution lifecycle"
```

---

## Chunk 5: Linter Validation

### Task 9: Create output binding linter

**Files:**
- Create: `pkg/flows/linter/output_bindings.go`
- Create: `pkg/flows/linter/output_bindings_test.go`
- Modify: `pkg/flows/linter/linter.go`

- [ ] **Step 1: Write failing test**

```go
// pkg/flows/linter/output_bindings_test.go

package linter

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/denkhaus/gollum/pkg/flows"
)

func TestOutputBindingsChecker_Valid(t *testing.T) {
    flow := &flows.Flow{
        Input: &flows.InputBlock{
            Strings: []flows.FieldDef{{Name: "msg", Type: flows.TypeString}},
        },
        Output: &flows.OutputBlock{
            Strings: []flows.FieldDef{
                {Name: "result", From: "input.msg", Type: flows.TypeString},
            },
        },
    }

    checker := NewOutputBindingsChecker()
    result := &flows.LinterResult{}
    checker.Check(flow, result)

    assert.Empty(t, result.Errors)
    assert.Empty(t, result.Warnings)
}

func TestOutputBindingsChecker_InvalidField(t *testing.T) {
    flow := &flows.Flow{
        Output: &flows.OutputBlock{
            Strings: []flows.FieldDef{
                {Name: "result", From: "computed.nonexistent", Type: flows.TypeString},
            },
        },
    }

    checker := NewOutputBindingsChecker()
    result := &flows.LinterResult{}
    checker.Check(flow, result)

    assert.NotEmpty(t, result.Errors)
    assert.Contains(t, result.Errors[0].Code, "E007")
}

func TestOutputBindingsChecker_CircularDependency(t *testing.T) {
    flow := &flows.Flow{
        Output: &flows.OutputBlock{
            Ints: []flows.FieldDef{
                {Name: "a", From: "output.b", Type: flows.TypeInt},
                {Name: "b", From: "output.a", Type: flows.TypeInt},
            },
        },
    }

    checker := NewOutputBindingsChecker()
    result := &flows.LinterResult{}
    checker.Check(flow, result)

    assert.NotEmpty(t, result.Errors)
    assert.Contains(t, result.Errors[0].Code, "E008")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/linter -run TestOutputBindingsChecker -v`
Expected: FAIL with "undefined type"

- [ ] **Step 3: Implement OutputBindingsChecker**

```go
// pkg/flows/linter/output_bindings.go

package linter

import (
    "fmt"

    "github.com/denkhaus/gollum/pkg/flows"
    "github.com/denkhaus/gollum/pkg/flows/variables"
)

// OutputBindingsChecker validates output field bindings
type OutputBindingsChecker struct {
    PosTracker *PositionTracker
}

// NewOutputBindingsChecker creates a new output bindings checker
func NewOutputBindingsChecker() *OutputBindingsChecker {
    return &OutputBindingsChecker{}
}

// Check validates all output field bindings in the flow
func (c *OutputBindingsChecker) Check(flow *flows.Flow, result *flows.LinterResult) {
    if flow.Output == nil {
        return
    }

    for _, field := range flow.Output.GetDeclarative() {
        c.checkBinding(flow, &field, result)
    }
}

// checkBinding validates a single output binding
func (c *OutputBindingsChecker) checkBinding(flow *flows.Flow, field *flows.FieldDef, result *flows.LinterResult) {
    // Parse the 'from' reference
    sourceScope, sourceName, err := variables.ParseFieldReference(field.From)
    if err != nil {
        result.AddError(flows.LintError{
            Code:    "E005",
            Message: fmt.Sprintf("invalid 'from' reference: %s", field.From),
            Position: c.getPosition(field),
        })
        return
    }

    // Validate scope
    if !variables.IsValidSourceScope(sourceScope) {
        result.AddError(flows.LintError{
            Code:    "E006",
            Message: fmt.Sprintf("invalid scope '%s' in 'from' attribute (allowed: input, context, computed, output)", sourceScope),
            Position: c.getPosition(field),
        })
        return
    }

    // Check for circular dependencies in output->output references
    if sourceScope == "output" {
        if c.hasCycle(flow, field.Name, sourceName, make(map[string]bool)) {
            result.AddError(flows.LintError{
                Code:    "E008",
                Message: fmt.Sprintf("circular dependency: output.%s -> output.%s", field.Name, sourceName),
                Position: c.getPosition(field),
            })
            return
        }
    }

    // Check referenced field exists
    if !c.fieldExists(flow, sourceScope, sourceName) {
        result.AddError(flows.LintError{
            Code:    "E007",
            Message: fmt.Sprintf("field '%s.%s' does not exist", sourceScope, sourceName),
            Position: c.getPosition(field),
        })
        return
    }

    // Check for duplicate sources (warning)
    if c.hasDuplicateSource(flow.Output, field.From, field.Name) {
        result.AddWarning(flows.LintError{
            Code:    "W003",
            Message: fmt.Sprintf("multiple output fields map from '%s'", field.From),
            Position: c.getPosition(field),
        })
    }
}

// fieldExists checks if a field exists in the specified scope
func (c *OutputBindingsChecker) fieldExists(flow *flows.Flow, scope, name string) bool {
    switch scope {
    case "input":
        return flow.Input != nil && flow.Input.HasField(name)
    case "context":
        return flow.Context != nil && flow.Context.HasField(name)
    case "computed":
        return flow.Computed != nil && flow.Computed.HasField(name)
    case "output":
        return flow.Output != nil && flow.Output.HasField(name)
    }
    return false
}

// hasCycle detects cycles in output->output references using DFS
func (c *OutputBindingsChecker) hasCycle(flow *flows.Flow, current, target string, visited map[string]bool) bool {
    if current == target {
        return true
    }
    if visited[current] {
        return false
    }
    visited[current] = true

    // Find current field and check if it has 'from'
    field := flow.Output.GetField(current)
    if field == nil || field.From == "" {
        return false
    }

    scope, name, _ := variables.ParseFieldReference(field.From)
    if scope == "output" {
        return c.hasCycle(flow, name, target, visited)
    }

    return false
}

// hasDuplicateSource checks if multiple outputs map from the same source
func (c *OutputBindingsChecker) hasDuplicateSource(output *flows.OutputBlock, source, excludeField string) bool {
    count := 0
    for _, f := range output.GetDeclarative() {
        if f.From == source && f.Name != excludeField {
            count++
        }
        if count > 0 {
            return true
        }
    }
    return false
}

// getPosition returns the position for a field (simplified)
func (c *OutputBindingsChecker) getPosition(field *flows.FieldDef) flows.Position {
    if c.PosTracker != nil {
        // Use position tracker if available
        return flows.Position{Line: 1, Column: 1} // Simplified
    }
    return flows.Position{}
}
```

- [ ] **Step 4: Add HasField helpers**

```go
// pkg/flows/types.go - add helpers to blocks

func (i *InputBlock) HasField(name string) bool {
    for _, f := range i.GetAllFields() {
        if f.Name == name {
            return true
        }
    }
    return false
}

func (c *ContextBlock) HasField(name string) bool {
    for _, f := range c.GetAllFields() {
        if f.Name == name {
            return true
        }
    }
    return false
}

func (c *ComputedBlock) HasField(name string) bool {
    for _, f := range c.GetAllFields() {
        if f.Name == name {
            return true
        }
    }
    return false
}

func (o *OutputBlock) HasField(name string) bool {
    for _, f := range o.GetAllFields() {
        if f.Name == name {
            return true
        }
    }
    return false
}

func (o *OutputBlock) GetField(name string) *flows.FieldDef {
    for _, f := range o.GetAllFields() {
        if f.Name == name {
            return &f
        }
    }
    return nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./pkg/flows/linter -run TestOutputBindingsChecker -v`
Expected: PASS

- [ ] **Step 6: Add to linter phases**

```go
// pkg/flows/linter/linter.go - add Phase 8

func LintWithContent(flowPath, xmlContent string, flow *flows.Flow) *flows.LinterResult {
    result := &flows.LinterResult{}

    var posTracker *PositionTracker
    if xmlContent != "" {
        posTracker = NewPositionTracker(xmlContent)
    }

    // Phase 1-7...

    // Phase 8: Output binding validation
    phase8 := NewOutputBindingsChecker()
    phase8.PosTracker = posTracker
    phase8.Check(flow, result)

    result.Valid = len(result.Errors) == 0
    return result
}
```

- [ ] **Step 7: Commit**

```bash
git add pkg/flows/linter/output_bindings.go pkg/flows/linter/output_bindings_test.go pkg/flows/linter/linter.go pkg/flows/types.go
git commit -m "feat(linter): add Phase 8 output binding validation"
```

---

## Chunk 6: Flow Migration

### Task 10: Refactor arithmetic-computed.xml

**Files:**
- Modify: `.gollum/flows/examples/arithmetic-computed.xml`

- [ ] **Step 1: Read current file**

Run: `cat .gollum/flows/examples/arithmetic-computed.xml`

- [ ] **Step 2: Replace with new version**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!--
Integer Arithmetic with Computed Fields
Tests ADD, SUB, MUL, DIV using computed block and output bindings
-->
<flow name="arithmetic-computed" version="1.0">
    <description>
        Integer arithmetic using computed block and output bindings.
        Test with: gollum flow run arithmetic-computed.xml -i "a=20,b=4"
        Expected: sum=24, diff=16, product=80, quotient=5
    </description>

    <input>
        <int name="a" default="10" />
        <int name="b" default="5" />
    </input>

    <computed>
        <int name="sum" eval="ADD(input.a, input.b)" />
        <int name="difference" eval="SUB(input.a, input.b)" />
        <int name="product" eval="MUL(input.a, input.b)" />
        <int name="quotient" eval="DIV(input.a, input.b)" />
    </computed>

    <output>
        <int name="sum" from="computed.sum" />
        <int name="difference" from="computed.difference" />
        <int name="product" from="computed.product" />
        <int name="quotient" from="computed.quotient" />
    </output>

    <states>
        <state name="done" initial="true" />
    </states>
</flow>
```

- [ ] **Step 3: Verify with linter**

Run: `gollum flow lint .gollum/flows/examples/arithmetic-computed.xml`
Expected: No errors

- [ ] **Step 4: Test flow execution**

Run: `gollum flow run .gollum/flows/examples/arithmetic-computed.xml -i "a=20,b=4"`
Expected: sum=24, diff=16, product=80, quotient=5

- [ ] **Step 5: Commit**

```bash
git add .gollum/flows/examples/arithmetic-computed.xml
git commit -m "refactor(flows): update arithmetic-computed.xml to use output bindings"
```

---

### Task 11: Refactor assign-input-to-output.xml

**Files:**
- Modify: `.gollum/flows/examples/assign-input-to-output.xml`

- [ ] **Step 1: Replace with new version**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!--
Assign Input to Output
Simplest test - pass input values directly to output using bindings
-->
<flow name="assign-input-to-output" version="1.0">
    <description>
        Simplest flow - copies input to output using bindings.
        Test with: gollum flow run assign-input-to-output.xml -i "message=Hello World"
    </description>

    <input>
        <string name="message" default="Hello" />
    </input>

    <output>
        <string name="result" from="input.message" />
    </output>

    <states>
        <state name="done" initial="true" />
    </states>
</flow>
```

- [ ] **Step 2: Verify with linter**

Run: `gollum flow lint .gollum/flows/examples/assign-input-to-output.xml`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add .gollum/flows/examples/assign-input-to-output.xml
git commit -m "refactor(flows): update assign-input-to-output.xml to use output bindings"
```

---

### Task 12: Refactor remaining flows

**Files:**
- Modify: Remaining 7 flow files

For each file:
1. Read the current content
2. Identify `assign` function usage
3. Replace with appropriate `from` attributes
4. Remove unnecessary states
5. Verify with linter
6. Commit

Pattern for commit messages:
```
refactor(flows): update <filename> to use output bindings
```

---

## Chunk 7: Documentation & Deprecation

### Task 13: Update syntax guide

**Files:**
- Modify: `docs/flows/syntax-guide.md`

- [ ] **Step 1: Add new section after "Field References"**

```markdown
## Output Field Bindings

Output fields can optionally declare their source using the `from` attribute:

### Declarative Output (with `from`)

Output fields with `from` are automatically populated from the specified source:

```xml
<output>
    <int name="sum" from="computed.sum" />
    <string name="message" from="input.text" />
    <bool name="is_valid" from="computed.is_valid" />
</output>
```

**Key properties:**
- **Readonly at runtime** - Cannot be set by tools/steps via `set_output_value`
- **Auto-populated** - Value flows automatically after source evaluation
- **Type-safe** - Linter validates source exists and type matches

### Imperative Output (without `from`)

Output fields without `from` are set by tools/steps at runtime:

```xml
<output>
    <string name="result" />
    <int name="count" />
</output>
```

### Allowed Sources

| Source | Example | When to Use |
|--------|---------|-------------|
| `input.*` | `from="input.value"` | Pass input through to output |
| `context.*` | `from="context.status"` | Expose context value as output |
| `computed.*` | `from="computed.sum"` | Expose computed value as output |
| `output.*` | `from="output.other"` | Alias another output field |

### Migration from `assign` Function

**Before (verbose):**
```xml
<step type="func" function="assign">
    <params>
        <param name="from" value="computed.sum" />
        <param name="to" value="output.sum" />
    </params>
</step>
```

**After (declarative):**
```xml
<output>
    <int name="sum" from="computed.sum" />
</output>
```
```

- [ ] **Step 2: Commit**

```bash
git add docs/flows/syntax-guide.md
git commit -m "docs: add output field bindings documentation to syntax guide"
```

---

### Task 14: Create migration guide

**Files:**
- Create: `docs/flows/migration-guide.md`

- [ ] **Step 1: Create migration guide**

```markdown
# Flow Migration Guide

## Migrating from `assign` Function to Output Bindings

The `assign` function is deprecated. Use output field bindings instead.

### Step 1: Identify `assign` Usage

Find all flows using `assign`:

```bash
grep -r 'function="assign"' .gollum/flows/
```

### Step 2: Replace with Output Bindings

**Before:**
```xml
<state name="assign" initial="true">
    <steps>
        <step type="func" function="assign">
            <params>
                <param name="from" value="computed.sum" />
                <param name="to" value="output.sum" />
            </params>
        </step>
    </steps>
</state>
```

**After:**
```xml
<output>
    <int name="sum" from="computed.sum" />
</output>
```

### Step 3: Remove Empty States

If the state only contained assign steps, remove it.

### Step 4: Verify with Linter

```bash
gollum flow lint your-flow.xml
```
```

- [ ] **Step 2: Commit**

```bash
git add docs/flows/migration-guide.md
git commit -m "docs: add migration guide for assign to output bindings"
```

---

## Chunk 8: Final Integration & Testing

### Task 15: Run all tests

- [ ] **Step 1: Run package tests**

Run: `go test ./pkg/flows/... -v`
Expected: All PASS

- [ ] **Step 2: Run linter tests**

Run: `go test ./pkg/flows/linter/... -v`
Expected: All PASS

- [ ] **Step 3: Run executor tests**

Run: `go test ./pkg/flows/executor/... -v`
Expected: All PASS

- [ ] **Step 4: Run integration tests**

Run: `go test ./pkg/flows/... -run Integration -v`
Expected: All PASS

- [ ] **Step 5: Commit any fixes**

```bash
git add .
git commit -m "test: fix failing tests after output bindings implementation"
```

---

### Task 16: Verify all refactored flows

- [ ] **Step 1: Lint all example flows**

Run: `gollum flow lint .gollum/flows/examples/*.xml`
Expected: No errors

- [ ] **Step 2: Lint all module flows**

Run: `gollum flow lint .gollum/flows/modules/**/*.xml`
Expected: No errors

- [ ] **Step 3: Test arithmetic-computed flow**

Run: `gollum flow run .gollum/flows/examples/arithmetic-computed.xml -i "a=20,b=4"`
Expected: sum=24, diff=16, product=80, quotient=5

- [ ] **Step 4: Commit any fixes**

```bash
git add .
git commit -m "fix: correct issues found in refactored flows"
```

---

## Summary

This implementation plan adds declarative output field bindings using the `from` attribute, eliminating the verbose `assign` function while maintaining backward compatibility.

### Files Created: 8
- `pkg/flows/errors/output_binding.go`
- `pkg/flows/linter/output_bindings.go`
- `pkg/flows/variables/output_bindings.go`
- `pkg/flows/executor/output_binder.go`
- Associated test files

### Files Modified: 6
- `pkg/flows/types.go`
- `pkg/flows/executor/context.go`
- `pkg/flows/executor/executor.go`
- `pkg/flows/linter/linter.go`
- `docs/flows/syntax-guide.md`
- 9 flow XML files

### Success Criteria
- [ ] Schema supports `from` attribute
- [ ] Declarative outputs auto-populated
- [ ] Readonly enforcement at runtime
- [ ] Linter validates all rules
- [ ] All tests pass (TDD)
- [ ] All 9 flows migrated
- [ ] Documentation updated
