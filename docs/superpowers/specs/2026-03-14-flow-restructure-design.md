# Flow Spec and Executor Restructure Design

**Date:** 2026-03-14
**Status:** Approved
**Author:** AI Agent

## Overview

Restructure the flow specification and executor to strictly separate the inner representations of "output", "context", "input", and "computed" values with strong types and comprehensive error handling.

## Goals

1. **Separation of Concerns:** Four distinct, independent value containers
2. **Strong Typing:** Fully typed structs with type-safe APIs
3. **Reactive Computed Values:** Automatic re-computation when dependencies change
4. **Better Linting:** Catch errors at parse/validate time
5. **Remove Magic:** Clear, predictable data flow

## Requirements

- **Simple expressions only** for computed fields (no function calls)
- **Fully typed structs** for compile-time safety where possible
- **Reactive evaluation** - computed values recompute when dependencies change
- **Typed error hierarchy** for precise error handling
- **TDD approach** for all implementation

---

## 1. Flow Spec Structure

### XML Schema

```xml
<flow name="example" version="1.0">
    <description>Flow description</description>

    <input>
        <string name="target" required="true" />
        <int name="count" default="0" />
    </input>

    <output>
        <string name="result" />
        <bool name="success" />
    </output>

    <context>
        <int name="complexity" />
        <string name="status" />
    </context>

    <computed>
        <bool name="is_complex" eval="GT(context.complexity, 10)" />
        <string name="summary" eval="CONCAT(input.target, ':', context.status)" />
    </computed>

    <agents>...</agents>
    <states>...</states>
</flow>
```

### Key Changes

- `<computed>` is now a top-level sibling to `<input>`, `<output>`, `<context>`
- Uses `eval` attribute (not `when`)
- Computed fields can reference: `input.*`, `context.*`, `output.*`, `computed.*`

---

## 2. Type System

### New Package: `pkg/flows/variables`

#### Value Types

```go
type ValueType string

const (
    TypeString ValueType = "string"
    TypeInt    ValueType = "int"
    TypeBool   ValueType = "bool"
    TypeFloat  ValueType = "float"
)
```

#### Field Value Container

```go
type FieldValue struct {
    valueType ValueType
    value     any
}

func NewStringValue(v string) FieldValue
func NewIntValue(v int) FieldValue
func NewBoolValue(v bool) FieldValue
func NewFloatValue(v float64) FieldValue

func (fv FieldValue) String() (string, error)
func (fv FieldValue) Int() (int, error)
func (fv FieldValue) Bool() (bool, error)
func (fv FieldValue) Float() (float64, error)
func (fv FieldValue) Type() ValueType
```

#### Scope-Specific Containers

```go
// InputValues - immutable after SetInput
type InputValues struct {
    fields map[string]FieldValue
    defs   map[string]FieldDef
}

func NewInputValues(block *InputBlock) *InputValues
func (iv *InputValues) GetString(name string) (string, error)
func (iv *InputValues) GetInt(name string) (int, error)
func (iv *InputValues) GetBool(name string) (bool, error)
func (iv *InputValues) GetFloat(name string) (float64, error)
func (iv *InputValues) Has(name string) bool

// ContextValues - mutable by tools
type ContextValues struct {
    fields    map[string]FieldValue
    defs      map[string]FieldDef
    evaluator *ComputedEvaluator
}

func NewContextValues(block *ContextBlock) *ContextValues
func (cv *ContextValues) SetString(name string, value string) error
func (cv *ContextValues) SetInt(name string, value int) error
func (cv *ContextValues) SetBool(name string, value bool) error
func (cv *ContextValues) Get...()...

// OutputValues - written by steps
type OutputValues struct {
    fields map[string]FieldValue
    defs   map[string]FieldDef
}

func NewOutputValues(block *OutputBlock) *OutputValues
func (ov *OutputValues) SetString(name string, value string) error
func (ov *OutputValues) Get...()...

// ComputedValues - reactive, read-only
type ComputedValues struct {
    fields map[string]*ComputedField
}

func NewComputedValues(block *ComputedBlock) *ComputedValues
func (cv *ComputedValues) GetBool(name string) (bool, error)
func (cv *ComputedValues) Get...()...
```

#### Computed Field Definition

```go
type ComputedField struct {
    Name         string
    Type         ValueType
    Expression   string
    Dependencies []FieldReference
    lastValue    *FieldValue
    dirty        bool
}

type FieldReference struct {
    Scope string // "input", "context", "output", "computed"
    Name  string
}
```

---

## 3. Reactive Computed Evaluation

### ComputedEvaluator

```go
type ComputedEvaluator struct {
    computeds  *ComputedValues
    input      *InputValues
    context    *ContextValues
    output     *OutputValues
    exprEval   *ExpressionEvaluator
    dependents map[string][]string // field -> computed fields that depend on it
}

func NewComputedEvaluator(computed *ComputedValues, input *InputValues,
    context *ContextValues, output *OutputValues) *ComputedEvaluator

// ComputeDirty re-evaluates only dirty computed fields
func (ce *ComputedEvaluator) ComputeDirty() error

// MarkDirty marks a computed field and its dependents as needing recompute
func (ce *ComputedEvaluator) MarkDirty(changedField string)

// GetValue returns a computed value, evaluating if dirty
func (ce *ComputedEvaluator) GetValue(name string) (any, error)
```

### Dependency Graph

```
input.x ──┬──> computed.a (GT(input.x, 10)) ──> computed.c (AND(computed.a, computed.b))
          │
          └──> computed.b (LT(input.x, 20))

When input.x changes:
  1. MarkDirty("input.x")
  2. Mark computed.a and computed.b as dirty
  3. ComputeDirty() evaluates a and b
  4. If a or b changed, mark computed.c as dirty
```

---

## 4. Expression Evaluation

### Supported Operators

| Category | Operators |
|----------|-----------|
| Comparison | GT, LT, GTE, LTE, EQ, NEQ |
| Logical | AND, OR, NOT |
| Arithmetic | ADD, SUB, MUL, DIV |

### Expression Syntax

```
GT(context.complexity, 10)
AND(context.is_ready, input.has_permission)
OR(context.failed, output.is_error)
NOT(context.is_valid)
ADD(input.x, input.y)
```

### ExpressionEvaluator

```go
type ExpressionEvaluator struct {
    parser *ExpressionParser
}

func NewExpressionEvaluator() *ExpressionEvaluator

// Evaluate evaluates an expression against the current scope
func (ee *ExpressionEvaluator) Evaluate(expr string, scope *EvaluationScope) (any, error)

// ExtractDependencies returns all field references in an expression
func (ee *ExpressionEvaluator) ExtractDependencies(expr string) []FieldReference

// GetReturnType infers the return type of an expression
func (ee *ExpressionEvaluator) GetReturnType(expr string) (ValueType, error)
```

### Evaluation Scope

```go
type EvaluationScope struct {
    Input    map[string]any
    Context  map[string]any
    Output   map[string]any
    Computed map[string]any
}
```

---

## 5. Error Hierarchy

### New Package: `pkg/flows/errors`

```go
// Base error type
type FlowError struct {
    Code    string
    Message string
    Field   string
    Cause   error
}

func (fe *FlowError) Error() string
func (fe *FlowError) Unwrap() error

// Specific error types
type UnknownFieldError struct {
    FlowError
    Scope string // "input", "context", "output", "computed"
}

type TypeError struct {
    FlowError
    ExpectedType ValueType
    ActualType   ValueType
}

type CircularDependencyError struct {
    FlowError
    Cycle []string
}

type ExpressionError struct {
    FlowError
    Expression string
    Position   int
}

type ImmutableFieldError struct {
    FlowError
    AttemptedOperation string
}

type ValidationError struct {
    FlowError
    Errors []error
}
```

### Error Codes

```go
const (
    ErrCodeUnknownField     = "UNKNOWN_FIELD"
    ErrCodeTypeMismatch     = "TYPE_MISMATCH"
    ErrCodeCircularDep      = "CIRCULAR_DEPENDENCY"
    ErrCodeExpression       = "EXPRESSION_ERROR"
    ErrCodeImmutable        = "IMMUTABLE_FIELD"
    ErrCodeValidation       = "VALIDATION_FAILED"
)
```

---

## 6. Updated Executor Integration

### ExecutionContext

```go
// executor/execution_context.go
type ExecutionContext struct {
    input     *variables.InputValues
    context   *variables.ContextValues
    output    *variables.OutputValues
    computed  *variables.ComputedValues
    evaluator *variables.ComputedEvaluator
}

func NewExecutionContext(flow *flows.Flow) (*ExecutionContext, error)

// SetInput initializes input values (converts from string map)
func (ec *ExecutionContext) SetInput(vals map[string]string) error

// SetContextField uses typed API based on field definition
func (ec *ExecutionContext) SetContextField(name string, value string) error {
    fieldDef := ec.getContextFieldDef(name)
    switch fieldDef.Type {
    case variables.TypeString:
        return ec.context.SetString(name, value)
    case variables.TypeInt:
        intValue, err := strconv.Atoi(value)
        if err != nil {
            return &errors.TypeError{...}
        }
        return ec.context.SetInt(name, intValue)
    // ... etc
}

// GetContextField uses typed API
func (ec *ExecutionContext) GetContextField(name string) (any, error)

// GetComputedField returns computed value (triggers eval if dirty)
func (ec *ExecutionContext) GetComputedField(name string) (any, error) {
    return ec.evaluator.GetValue(name)
}
```

### Executor Changes

```go
// executor/executor.go
type flowExecutorImpl struct {
    flow              *flows.Flow
    ctx               *ExecutionContext  // Changed from *Context
    // ... other fields
}

// substituteTemplate now uses typed getters
func (p *flowExecutorImpl) substituteTemplate(cmd string) string {
    scope := map[string]any{
        "input":   p.ctx.BuildInputScope(),
        "context": p.ctx.BuildContextScope(),
        "output":  p.ctx.BuildOutputScope(),
        "computed": p.ctx.BuildComputedScope(),
    }
    // ... template substitution
}
```

---

## 7. Updated Flow Types

### pkg/flows/types.go

```go
type Flow struct {
    XMLName     xml.Name          `xml:"flow"`
    Name        string            `xml:"name,attr"`
    Version     string            `xml:"version,attr"`
    Description string            `xml:"description"`
    Input       *InputBlock       `xml:"input"`
    Output      *OutputBlock      `xml:"output"`
    Context     *ContextBlock     `xml:"context"`
    Computed    *ComputedBlock    `xml:"computed"`  // NEW
    Agents      []Agent           `xml:"agents>agent"`
    States      []State           `xml:"states>state"`
}

// ComputedBlock defines computed fields (top-level)
type ComputedBlock struct {
    Bools   []ComputedFieldDef `xml:"bool"`
    Ints    []ComputedFieldDef `xml:"int"`
    Strings []ComputedFieldDef `xml:"string"`
    Floats  []ComputedFieldDef `xml:"float"`
}

func (cb *ComputedBlock) GetAll() []ComputedFieldDef

type ComputedFieldDef struct {
    XMLName xml.Name `xml:"bool,int,string,float"`
    Name    string   `xml:"name,attr"`
    Type    string   `xml:"type,attr"`
    Eval    string   `xml:"eval,attr"`  // Changed from "when"
}

// ContextBlock no longer has Computeds
type ContextBlock struct {
    Strings []ContextField `xml:"string"`
    Ints    []ContextField `xml:"int"`
    Bools   []ContextField `xml:"bool"`
    Floats  []ContextField `xml:"float"`
    Objects []ObjectDef    `xml:"object"`
    // Computeds removed
}
```

---

## 8. File Structure

```
pkg/flows/
├── types.go              # XML structs (add ComputedBlock)
├── registry/
├── executor/
│   ├── executor.go       # Update to use ExecutionContext
│   ├── execution_context.go  # NEW (replace context.go)
│   └── template.go       # Keep
├── variables/            # NEW PACKAGE
│   ├── typed.go          # FieldValue, *Values structs
│   ├── computed.go       # ComputedEvaluator, reactive logic
│   ├── expression.go     # ExpressionEvaluator
│   ├── scope.go          # EvaluationScope
│   └── reference.go      # FieldReference
├── errors/               # NEW PACKAGE
│   ├── errors.go         # FlowError base, specific types
│   └── codes.go          # Error code constants
└── validation.go         # Flow validation helpers
```

---

## 9. Migration Path

### Phase 1: Add New Types (Non-Breaking)
1. Create `pkg/flows/variables` package
2. Create `pkg/flows/errors` package
3. Add `ComputedBlock` to types.go
4. Keep `ContextBlock.Computeds` for backward compatibility
5. Add deprecation warning

### Phase 2: Update Executor
1. Create `ExecutionContext` alongside `Context`
2. Integrate reactive computed evaluation
3. Update all `SetContext/SetOutput` calls
4. Add tests for new implementation

### Phase 3: Migrate Flows
1. Linter detects flows using old-style `<context><computed>`
2. Provide migration tool to move computed to top-level
3. Update example flows

### Phase 4: Remove Deprecated
1. Remove `when` attribute support
2. Remove `ContextBlock.Computeds`
3. Remove old `Context` implementation

---

## 10. Linting Enhancements

### New Linter Checks

```go
// pkg/lint/computed.go
func CheckComputedFields(flow *Flow) []LintError {
    var errors []LintError

    for _, computed := range flow.Computed.GetAll() {
        // 1. Parse expression
        // 2. Verify referenced fields exist
        // 3. Check for circular dependencies
        // 4. Type validation
    }

    return errors
}

// Detect undefined fields
func CheckFieldReferences(flow *Flow) []LintError

// Detect circular dependencies
func DetectCycles(computed *ComputedBlock) [][]string

// Validate expression syntax
func ValidateExpressions(computed *ComputedBlock) []error
```

### CLI Output

```bash
$ gollum flow lint myflow.xml
❌ 2 errors, 1 warning found

❌ [E001] Undefined field: context.undef in computed.is_valid
   └─ myflow.xml:15:13

❌ [E002] Circular dependency detected
   └─ cycle: is_a → is_b → is_a

⚠️  [W001] Expression returns string but field declared as int
   └─ computed.result
```

---

## 11. Testing Strategy

### Unit Tests (TDD Approach)

**variables/typed_test.go:**
- Test type-safe getters/setters
- Test type mismatch errors
- Test unknown field errors
- Test immutability of InputValues

**variables/computed_test.go:**
- Test reactive updates
- Test dependency tracking
- Test circular dependency detection
- Test topological evaluation order

**variables/expression_test.go:**
- Test all operators
- Test type coercion
- Test error cases
- Test dependency extraction

**errors/errors_test.go:**
- Test error wrapping
- Test error messages
- Test error type assertions

### Integration Tests

**executor/executor_test.go:**
- Test complete flow execution with computed
- Test computed reactivity during execution
- Test error propagation

### Linter Tests

**lint/computed_test.go:**
- Test undefined field detection
- Test circular dependency detection
- Test type mismatch warnings

---

## 12. Implementation Notes

### TDD Workflow
1. Write failing test
2. Implement minimum code to pass
3. Refactor
4. Repeat

### Key Invariants
- InputValues is immutable after SetInput
- ComputedValues is read-only (computed from expressions)
- ContextValues is mutable (written by tools)
- OutputValues is write-once (per field)

### Performance Considerations
- Lazy evaluation of computed fields
- Dependency graph cached
- Only dirty fields are re-evaluated

---

## 13. Success Criteria

- [ ] All value containers use strongly-typed APIs
- [ ] Computed fields are reactive to dependency changes
- [ ] Linter catches all common errors at parse time
- [ ] Error hierarchy provides precise error types
- [ ] All code has comprehensive test coverage (TDD)
- [ ] No breaking changes to existing flows (migration path)
- [ ] Documentation updated
