# Flow Spec and Executor Restructure Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restructure flow spec and executor with strict separation of input/context/output/computed, strong typing, reactive computed evaluation, and comprehensive error handling.

**Architecture:** Create new `pkg/flows/variables` package with typed containers (InputValues, ContextValues, OutputValues, ComputedValues), new `pkg/flows/errors` package for error hierarchy, reactive ComputedEvaluator for automatic re-computation, and update executor to use new ExecutionContext.

**Tech Stack:** Go 1.23+, existing codebase patterns (samber/do DI, testify for testing), TDD approach throughout.

---

## Chunk 1: Error Hierarchy Foundation

This chunk establishes the error handling foundation that all other components will depend on.

### Task 1: Create Base Error Types

**Files:**
- Create: `pkg/flows/errors/errors.go`
- Create: `pkg/flows/errors/codes.go`
- Test: `pkg/flows/errors/errors_test.go`

- [ ] **Step 1: Write failing test for base FlowError**

```go
// pkg/flows/errors/errors_test.go
package errors_test

import (
    "errors"
    "testing"

    "github.com/denkhaus/gollum/pkg/flows/errors"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestFlowError_Error(t *testing.T) {
    err := &errors.FlowError{
        Code:    "TEST_CODE",
        Message: "test message",
        Field:   "testField",
    }

    assert.Equal(t, "TEST_CODE: test message (field: testField)", err.Error())
}

func TestFlowError_Unwrap(t *testing.T) {
    cause := errors.New("underlying error")
    err := &errors.FlowError{
        Code:    "ERR_CODE",
        Message: "wrapper message",
        Cause:   cause,
    }

    assert.Equal(t, cause, errors.Unwrap(err))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/errors/...`
Expected: FAIL with "undefined: errors.FlowError"

- [ ] **Step 3: Write minimal implementation**

```go
// pkg/flows/errors/errors.go
package errors

import "fmt"

// FlowError is the base error type for all flow-related errors
type FlowError struct {
    Code    string
    Message string
    Field   string
    Cause   error
}

func (fe *FlowError) Error() string {
    if fe.Field != "" {
        return fmt.Sprintf("%s: %s (field: %s)", fe.Code, fe.Message, fe.Field)
    }
    return fmt.Sprintf("%s: %s", fe.Code, fe.Message)
}

func (fe *FlowError) Unwrap() error {
    return fe.Cause
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/errors/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/errors/errors.go pkg/flows/errors/errors_test.go
git commit -m "feat(errors): add base FlowError type"
```

### Task 2: Add Specific Error Types

**Files:**
- Modify: `pkg/flows/errors/errors.go`
- Modify: `pkg/flows/errors/errors_test.go`

- [ ] **Step 1: Write failing test for UnknownFieldError**

```go
func TestUnknownFieldError(t *testing.T) {
    err := &errors.UnknownFieldError{
        FlowError: errors.FlowError{
            Code:    errors.ErrCodeUnknownField,
            Message: "field not found",
            Field:   "missingField",
        },
        Scope: "context",
    }

    assert.Equal(t, "context", err.Scope)
    assert.Equal(t, errors.ErrCodeUnknownField, err.Code)
}

func TestTypeError(t *testing.T) {
    err := &errors.TypeError{
        FlowError: errors.FlowError{
            Code:    errors.ErrCodeTypeMismatch,
            Message: "type mismatch",
            Field:   "count",
        },
        ExpectedType: "int",
        ActualType:   "string",
    }

    assert.Equal(t, "int", err.ExpectedType)
    assert.Equal(t, "string", err.ActualType)
}

func TestCircularDependencyError(t *testing.T) {
    err := &errors.CircularDependencyError{
        FlowError: errors.FlowError{
            Code:    errors.ErrCodeCircularDep,
            Message: "circular dependency detected",
        },
        Cycle: []string{"a", "b", "a"},
    }

    assert.Equal(t, []string{"a", "b", "a"}, err.Cycle)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/errors/...`
Expected: FAIL with "undefined: errors.UnknownFieldError"

- [ ] **Step 3: Write minimal implementation**

```go
// pkg/flows/errors/errors.go (continued)

// UnknownFieldError indicates a field reference that doesn't exist in the flow
type UnknownFieldError struct {
    FlowError
    Scope string // "input", "context", "output", "computed"
}

// TypeError indicates a type mismatch in value assignment or expression
type TypeError struct {
    FlowError
    ExpectedType string
    ActualType   string
}

// CircularDependencyError indicates computed fields depend on each other
type CircularDependencyError struct {
    FlowError
    Cycle []string
}

// ExpressionError indicates an invalid expression syntax
type ExpressionError struct {
    FlowError
    Expression string
    Position   int
}

// ImmutableFieldError indicates an attempt to modify an immutable field
type ImmutableFieldError struct {
    FlowError
    AttemptedOperation string
}

// ValidationError aggregates multiple validation errors
type ValidationError struct {
    FlowError
    Errors []error
}

func (ve *ValidationError) Error() string {
    return fmt.Sprintf("%s: %d validation errors", ve.Code, len(ve.Errors))
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/errors/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/errors/errors.go pkg/flows/errors/errors_test.go
git commit -m "feat(errors): add specific error types"
```

### Task 3: Define Error Codes

**Files:**
- Create: `pkg/flows/errors/codes.go`

- [ ] **Step 1: Write test for error codes constant**

```go
func TestErrorCodes(t *testing.T) {
    assert.Equal(t, "UNKNOWN_FIELD", errors.ErrCodeUnknownField)
    assert.Equal(t, "TYPE_MISMATCH", errors.ErrCodeTypeMismatch)
    assert.Equal(t, "CIRCULAR_DEPENDENCY", errors.ErrCodeCircularDep)
    assert.Equal(t, "EXPRESSION_ERROR", errors.ErrCodeExpression)
    assert.Equal(t, "IMMUTABLE_FIELD", errors.ErrCodeImmutable)
    assert.Equal(t, "VALIDATION_FAILED", errors.ErrCodeValidation)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/errors/...`
Expected: FAIL with "undefined: errors.ErrCodeUnknownField"

- [ ] **Step 3: Write minimal implementation**

```go
// pkg/flows/errors/codes.go
package errors

const (
    ErrCodeUnknownField     = "UNKNOWN_FIELD"
    ErrCodeTypeMismatch     = "TYPE_MISMATCH"
    ErrCodeCircularDep      = "CIRCULAR_DEPENDENCY"
    ErrCodeExpression       = "EXPRESSION_ERROR"
    ErrCodeImmutable        = "IMMUTABLE_FIELD"
    ErrCodeValidation       = "VALIDATION_FAILED"
)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/errors/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/errors/codes.go pkg/flows/errors/errors_test.go
git commit -m "feat(errors): define error code constants"
```

---

## Chunk 2: Variables Package - Core Types

This chunk creates the core typed value containers.

### Task 4: Create FieldValue Type

**Files:**
- Create: `pkg/flows/variables/typed.go`
- Test: `pkg/flows/variables/typed_test.go`

- [ ] **Step 1: Write failing test for FieldValue**

```go
// pkg/flows/variables/typed_test.go
package variables_test

import (
    "testing"

    "github.com/denkhaus/gollum/pkg/flows/variables"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestNewStringValue(t *testing.T) {
    fv := variables.NewStringValue("test")

    assert.Equal(t, variables.TypeString, fv.Type())

    val, err := fv.String()
    require.NoError(t, err)
    assert.Equal(t, "test", val)
}

func TestNewIntValue(t *testing.T) {
    fv := variables.NewIntValue(42)

    assert.Equal(t, variables.TypeInt, fv.Type())

    val, err := fv.Int()
    require.NoError(t, err)
    assert.Equal(t, 42, val)
}

func TestNewBoolValue(t *testing.T) {
    fv := variables.NewBoolValue(true)

    assert.Equal(t, variables.TypeBool, fv.Type())

    val, err := fv.Bool()
    require.NoError(t, err)
    assert.True(t, val)
}

func TestNewFloatValue(t *testing.T) {
    fv := variables.NewFloatValue(3.14)

    assert.Equal(t, variables.TypeFloat, fv.Type())

    val, err := fv.Float()
    require.NoError(t, err)
    assert.InDelta(t, 3.14, val, 0.001)
}

func TestFieldValue_TypeMismatch(t *testing.T) {
    fv := variables.NewStringValue("test")

    _, err := fv.Int()
    assert.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/variables/...`
Expected: FAIL with "undefined: variables.NewStringValue"

- [ ] **Step 3: Write minimal implementation**

```go
// pkg/flows/variables/typed.go
package variables

import (
    "fmt"
)

// ValueType represents the type of a field value
type ValueType string

const (
    TypeString ValueType = "string"
    TypeInt    ValueType = "int"
    TypeBool   ValueType = "bool"
    TypeFloat  ValueType = "float"
)

// FieldValue is a type-safe wrapper for values
type FieldValue struct {
    valueType ValueType
    value     any
}

// NewStringValue creates a string FieldValue
func NewStringValue(v string) FieldValue {
    return FieldValue{valueType: TypeString, value: v}
}

// NewIntValue creates an int FieldValue
func NewIntValue(v int) FieldValue {
    return FieldValue{valueType: TypeInt, value: v}
}

// NewBoolValue creates a bool FieldValue
func NewBoolValue(v bool) FieldValue {
    return FieldValue{valueType: TypeBool, value: v}
}

// NewFloatValue creates a float FieldValue
func NewFloatValue(v float64) FieldValue {
    return FieldValue{valueType: TypeFloat, value: v}
}

// Type returns the value type
func (fv FieldValue) Type() ValueType {
    return fv.valueType
}

// String returns the string value
func (fv FieldValue) String() (string, error) {
    if fv.valueType != TypeString {
        return "", fmt.Errorf("type mismatch: expected %s, got %s", TypeString, fv.valueType)
    }
    return fv.value.(string), nil
}

// Int returns the int value
func (fv FieldValue) Int() (int, error) {
    if fv.valueType != TypeInt {
        return 0, fmt.Errorf("type mismatch: expected %s, got %s", TypeInt, fv.valueType)
    }
    return fv.value.(int), nil
}

// Bool returns the bool value
func (fv FieldValue) Bool() (bool, error) {
    if fv.valueType != TypeBool {
        return false, fmt.Errorf("type mismatch: expected %s, got %s", TypeBool, fv.valueType)
    }
    return fv.value.(bool), nil
}

// Float returns the float value
func (fv FieldValue) Float() (float64, error) {
    if fv.valueType != TypeFloat {
        return 0, fmt.Errorf("type mismatch: expected %s, got %s", TypeFloat, fv.valueType)
    }
    return fv.value.(float64), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/variables/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/variables/typed.go pkg/flows/variables/typed_test.go
git commit -m "feat(variables): add FieldValue type"
```

### Task 5: Create InputValues Container

**Files:**
- Modify: `pkg/flows/variables/typed.go`
- Modify: `pkg/flows/variables/typed_test.go`

- [ ] **Step 1: Write failing test for InputValues**

```go
func TestInputValues_GetSet(t *testing.T) {
    block := &flows.InputBlock{
        Strings: []flows.FieldDef{{Name: "name", Type: "string"}},
        Ints:    []flows.FieldDef{{Name: "count", Type: "int"}},
    }

    input := variables.NewInputValues(block)

    // Set values
    err := input.SetString("name", "test")
    require.NoError(t, err)

    err = input.SetInt("count", 42)
    require.NoError(t, err)

    // Get values
    val, err := input.GetString("name")
    require.NoError(t, err)
    assert.Equal(t, "test", val)

    count, err := input.GetInt("count")
    require.NoError(t, err)
    assert.Equal(t, 42, count)
}

func TestInputValues_UnknownField(t *testing.T) {
    block := &flows.InputBlock{}
    input := variables.NewInputValues(block)

    err := input.SetString("unknown", "test")
    assert.Error(t, err)
}

func TestInputValues_TypeMismatch(t *testing.T) {
    block := &flows.InputBlock{
        Strings: []flows.FieldDef{{Name: "name", Type: "string"}},
    }
    input := variables.NewInputValues(block)

    err := input.SetInt("name", 42)
    assert.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/variables/...`
Expected: FAIL with "undefined: variables.NewInputValues"

- [ ] **Step 3: Write minimal implementation**

```go
// pkg/flows/variables/typed.go (add after FieldValue)

import (
    "github.com/denkhaus/gollum/pkg/flows"
    "github.com/denkhaus/gollum/pkg/flows/errors"
)

// InputValues stores input field values (immutable after SetInput)
type InputValues struct {
    fields map[string]FieldValue
    defs   map[string]flows.FieldDef
}

// NewInputValues creates a new InputValues from InputBlock
func NewInputValues(block *flows.InputBlock) *InputValues {
    iv := &InputValues{
        fields: make(map[string]FieldValue),
        defs:   make(map[string]flows.FieldDef),
    }

    if block == nil {
        return iv
    }

    for _, field := range block.GetAllFields() {
        iv.defs[field.Name] = field
    }

    return iv
}

// SetString sets a string input value
func (iv *InputValues) SetString(name string, value string) error {
    if _, ok := iv.defs[name]; !ok {
        return &errors.UnknownFieldError{
            FlowError: errors.FlowError{
                Code:    errors.ErrCodeUnknownField,
                Message: "input field not defined",
                Field:   name,
            },
            Scope: "input",
        }
    }

    if iv.defs[name].Type != string(TypeString) {
        return &errors.TypeError{
            FlowError: errors.FlowError{
                Code:    errors.ErrCodeTypeMismatch,
                Message: "field type mismatch",
                Field:   name,
            },
            ExpectedType: iv.defs[name].Type,
            ActualType:   string(TypeString),
        }
    }

    iv.fields[name] = NewStringValue(value)
    return nil
}

// SetInt sets an int input value
func (iv *InputValues) SetInt(name string, value int) error {
    if _, ok := iv.defs[name]; !ok {
        return &errors.UnknownFieldError{
            FlowError: errors.FlowError{
                Code:    errors.ErrCodeUnknownField,
                Message: "input field not defined",
                Field:   name,
            },
            Scope: "input",
        }
    }

    if iv.defs[name].Type != string(TypeInt) {
        return &errors.TypeError{
            FlowError: errors.FlowError{
                Code:    errors.ErrCodeTypeMismatch,
                Message: "field type mismatch",
                Field:   name,
            },
            ExpectedType: iv.defs[name].Type,
            ActualType:   string(TypeInt),
        }
    }

    iv.fields[name] = NewIntValue(value)
    return nil
}

// SetBool sets a bool input value
func (iv *InputValues) SetBool(name string, value bool) error {
    if _, ok := iv.defs[name]; !ok {
        return &errors.UnknownFieldError{...}
    }
    if iv.defs[name].Type != string(TypeBool) {
        return &errors.TypeError{...}
    }
    iv.fields[name] = NewBoolValue(value)
    return nil
}

// SetFloat sets a float input value
func (iv *InputValues) SetFloat(name string, value float64) error {
    // Similar to SetInt/SetBool
    if _, ok := iv.defs[name]; !ok {
        return &errors.UnknownFieldError{...}
    }
    if iv.defs[name].Type != string(TypeFloat) {
        return &errors.TypeError{...}
    }
    iv.fields[name] = NewFloatValue(value)
    return nil
}

// GetString returns a string input value
func (iv *InputValues) GetString(name string) (string, error) {
    fv, ok := iv.fields[name]
    if !ok {
        return "", &errors.UnknownFieldError{...}
    }
    return fv.String()
}

// GetInt returns an int input value
func (iv *InputValues) GetInt(name string) (int, error) {
    fv, ok := iv.fields[name]
    if !ok {
        return 0, &errors.UnknownFieldError{...}
    }
    return fv.Int()
}

// GetBool returns a bool input value
func (iv *InputValues) GetBool(name string) (bool, error) {
    fv, ok := iv.fields[name]
    if !ok {
        return false, &errors.UnknownFieldError{...}
    }
    return fv.Bool()
}

// GetFloat returns a float input value
func (iv *InputValues) GetFloat(name string) (float64, error) {
    fv, ok := iv.fields[name]
    if !ok {
        return 0, &errors.UnknownFieldError{...}
    }
    return fv.Float()
}

// Has returns true if field has a value
func (iv *InputValues) Has(name string) bool {
    _, ok := iv.fields[name]
    return ok
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/variables/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/variables/typed.go pkg/flows/variables/typed_test.go
git commit -m "feat(variables): add InputValues container"
```

### Task 6: Create ContextValues Container

**Files:**
- Modify: `pkg/flows/variables/typed.go`
- Modify: `pkg/flows/variables/typed_test.go`

- [ ] **Step 1: Write failing test for ContextValues**

```go
func TestContextValues_GetSet(t *testing.T) {
    block := &flows.ContextBlock{
        Strings: []flows.ContextField{{Name: "status", Type: "string"}},
        Ints:    []flows.ContextField{{Name: "count", Type: "int"}},
    }

    context := variables.NewContextValues(block)

    // Set values
    err := context.SetString("status", "ready")
    require.NoError(t, err)

    err = context.SetInt("count", 10)
    require.NoError(t, err)

    // Get values
    val, err := context.GetString("status")
    require.NoError(t, err)
    assert.Equal(t, "ready", val)

    count, err := context.GetInt("count")
    require.NoError(t, err)
    assert.Equal(t, 10, count)
}

func TestContextValues_DefaultValues(t *testing.T) {
    block := &flows.ContextBlock{
        Ints: []flows.ContextField{
            {Name: "timeout", Type: "int", Default: "30"},
        },
    }

    context := variables.NewContextValues(block)

    // Get default value before set
    val, err := context.GetInt("timeout")
    require.NoError(t, err)
    assert.Equal(t, 30, val)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/variables/...`
Expected: FAIL with "undefined: variables.NewContextValues"

- [ ] **Step 3: Write minimal implementation**

```go
// pkg/flows/variables/typed.go (add after InputValues)

import (
    "strconv"
)

// ContextValues stores context field values (mutable by tools)
type ContextValues struct {
    fields    map[string]FieldValue
    defs      map[string]flows.ContextField
    evaluator *ComputedEvaluator // Will be set later
}

// NewContextValues creates a new ContextValues from ContextBlock
func NewContextValues(block *flows.ContextBlock) *ContextValues {
    cv := &ContextValues{
        fields: make(map[string]FieldValue),
        defs:   make(map[string]flows.ContextField),
    }

    if block == nil {
        return cv
    }

    // Register string fields
    for _, field := range block.Strings {
        cv.defs[field.Name] = field
        if field.Default != "" {
            cv.fields[field.Name] = NewStringValue(field.Default)
        }
    }

    // Register int fields
    for _, field := range block.Ints {
        cv.defs[field.Name] = field
        if field.Default != "" {
            if i, err := strconv.Atoi(field.Default); err == nil {
                cv.fields[field.Name] = NewIntValue(i)
            }
        }
    }

    // Register bool fields
    for _, field := range block.Bools {
        cv.defs[field.Name] = field
        if field.Default != "" {
            if b, err := strconv.ParseBool(field.Default); err == nil {
                cv.fields[field.Name] = NewBoolValue(b)
            }
        }
    }

    // Register float fields
    for _, field := range block.Floats {
        cv.defs[field.Name] = field
        if field.Default != "" {
            if f, err := strconv.ParseFloat(field.Default, 64); err == nil {
                cv.fields[field.Name] = NewFloatValue(f)
            }
        }
    }

    return cv
}

// SetString sets a string context value
func (cv *ContextValues) SetString(name string, value string) error {
    if _, ok := cv.defs[name]; !ok {
        return &errors.UnknownFieldError{
            FlowError: errors.FlowError{
                Code:    errors.ErrCodeUnknownField,
                Message: "context field not defined",
                Field:   name,
            },
            Scope: "context",
        }
    }

    def, ok := cv.defs[name]
    if !ok || def.Type != string(TypeString) {
        return &errors.TypeError{...}
    }

    // Check if value changed
    oldValue, hadOld := cv.fields[name]
    cv.fields[name] = NewStringValue(value)

    // Notify evaluator of change
    if cv.evaluator != nil {
        if !hadOld || oldValue.value != value {
            cv.evaluator.MarkDirty(name)
        }
    }

    return nil
}

// SetInt sets an int context value
func (cv *ContextValues) SetInt(name string, value int) error {
    if _, ok := cv.defs[name]; !ok {
        return &errors.UnknownFieldError{...}
    }

    def := cv.defs[name]
    if def.Type != string(TypeInt) {
        return &errors.TypeError{...}
    }

    oldValue, hadOld := cv.fields[name]
    cv.fields[name] = NewIntValue(value)

    if cv.evaluator != nil {
        if !hadOld || oldValue.value != value {
            cv.evaluator.MarkDirty(name)
        }
    }

    return nil
}

// SetBool sets a bool context value
func (cv *ContextValues) SetBool(name string, value bool) error {
    if _, ok := cv.defs[name]; !ok {
        return &errors.UnknownFieldError{...}
    }
    if cv.defs[name].Type != string(TypeBool) {
        return &errors.TypeError{...}
    }

    oldValue, hadOld := cv.fields[name]
    cv.fields[name] = NewBoolValue(value)

    if cv.evaluator != nil {
        if !hadOld || oldValue.value != value {
            cv.evaluator.MarkDirty(name)
        }
    }

    return nil
}

// SetFloat sets a float context value
func (cv *ContextValues) SetFloat(name string, value float64) error {
    // Similar to SetInt
    if _, ok := cv.defs[name]; !ok {
        return &errors.UnknownFieldError{...}
    }
    if cv.defs[name].Type != string(TypeFloat) {
        return &errors.TypeError{...}
    }

    oldValue, hadOld := cv.fields[name]
    cv.fields[name] = NewFloatValue(value)

    if cv.evaluator != nil {
        if !hadOld || oldValue.value != value {
            cv.evaluator.MarkDirty(name)
        }
    }

    return nil
}

// GetString returns a string context value
func (cv *ContextValues) GetString(name string) (string, error) {
    fv, ok := cv.fields[name]
    if !ok {
        return "", &errors.UnknownFieldError{...}
    }
    return fv.String()
}

// GetInt returns an int context value
func (cv *ContextValues) GetInt(name string) (int, error) {
    fv, ok := cv.fields[name]
    if !ok {
        return 0, &errors.UnknownFieldError{...}
    }
    return fv.Int()
}

// GetBool returns a bool context value
func (cv *ContextValues) GetBool(name string) (bool, error) {
    fv, ok := cv.fields[name]
    if !ok {
        return false, &errors.UnknownFieldError{...}
    }
    return fv.Bool()
}

// GetFloat returns a float context value
func (cv *ContextValues) GetFloat(name string) (float64, error) {
    fv, ok := cv.fields[name]
    if !ok {
        return 0, &errors.UnknownFieldError{...}
    }
    return fv.Float()
}

// SetEvaluator sets the computed evaluator for reactive updates
func (cv *ContextValues) SetEvaluator(eval *ComputedEvaluator) {
    cv.evaluator = eval
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/variables/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/variables/typed.go pkg/flows/variables/typed_test.go
git commit -m "feat(variables): add ContextValues container with reactive support"
```

### Task 7: Create OutputValues Container

**Files:**
- Modify: `pkg/flows/variables/typed.go`
- Modify: `pkg/flows/variables/typed_test.go`

- [ ] **Step 1: Write failing test for OutputValues**

```go
func TestOutputValues_GetSet(t *testing.T) {
    block := &flows.OutputBlock{
        Strings: []flows.FieldDef{{Name: "result", Type: "string"}},
        Bools:   []flows.FieldDef{{Name: "success", Type: "bool"}},
    }

    output := variables.NewOutputValues(block)

    // Set values
    err := output.SetString("result", "done")
    require.NoError(t, err)

    err = output.SetBool("success", true)
    require.NoError(t, err)

    // Get values
    val, err := output.GetString("result")
    require.NoError(t, err)
    assert.Equal(t, "done", val)

    success, err := output.GetBool("success")
    require.NoError(t, err)
    assert.True(t, success)
}

func TestOutputValues_WriteOnce(t *testing.T) {
    block := &flows.OutputBlock{
        Strings: []flows.FieldDef{{Name: "result", Type: "string"}},
    }

    output := variables.NewOutputValues(block)

    err := output.SetString("result", "first")
    require.NoError(t, err)

    // Second write should fail
    err = output.SetString("result", "second")
    assert.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/variables/...`
Expected: FAIL with "undefined: variables.NewOutputValues"

- [ ] **Step 3: Write minimal implementation**

```go
// pkg/flows/variables/typed.go (add after ContextValues)

// OutputValues stores output field values (write-once per field)
type OutputValues struct {
    fields map[string]FieldValue
    defs   map[string]flows.FieldDef
    written map[string]bool
}

// NewOutputValues creates a new OutputValues from OutputBlock
func NewOutputValues(block *flows.OutputBlock) *OutputValues {
    ov := &OutputValues{
        fields:  make(map[string]FieldValue),
        defs:    make(map[string]flows.FieldDef),
        written: make(map[string]bool),
    }

    if block == nil {
        return ov
    }

    for _, field := range block.GetAllFields() {
        ov.defs[field.Name] = field
    }

    return ov
}

// SetString sets a string output value (write-once)
func (ov *OutputValues) SetString(name string, value string) error {
    if ov.written[name] {
        return &errors.ImmutableFieldError{
            FlowError: errors.FlowError{
                Code:    errors.ErrCodeImmutable,
                Message: "output field already written",
                Field:   name,
            },
            AttemptedOperation: "SetString",
        }
    }

    if _, ok := ov.defs[name]; !ok {
        return &errors.UnknownFieldError{...}
    }

    if ov.defs[name].Type != string(TypeString) {
        return &errors.TypeError{...}
    }

    ov.fields[name] = NewStringValue(value)
    ov.written[name] = true
    return nil
}

// SetInt sets an int output value
func (ov *OutputValues) SetInt(name string, value int) error {
    if ov.written[name] {
        return &errors.ImmutableFieldError{...}
    }
    if _, ok := ov.defs[name]; !ok {
        return &errors.UnknownFieldError{...}
    }
    if ov.defs[name].Type != string(TypeInt) {
        return &errors.TypeError{...}
    }
    ov.fields[name] = NewIntValue(value)
    ov.written[name] = true
    return nil
}

// SetBool sets a bool output value
func (ov *OutputValues) SetBool(name string, value bool) error {
    if ov.written[name] {
        return &errors.ImmutableFieldError{...}
    }
    if _, ok := ov.defs[name]; !ok {
        return &errors.UnknownFieldError{...}
    }
    if ov.defs[name].Type != string(TypeBool) {
        return &errors.TypeError{...}
    }
    ov.fields[name] = NewBoolValue(value)
    ov.written[name] = true
    return nil
}

// SetFloat sets a float output value
func (ov *OutputValues) SetFloat(name string, value float64) error {
    if ov.written[name] {
        return &errors.ImmutableFieldError{...}
    }
    if _, ok := ov.defs[name]; !ok {
        return &errors.UnknownFieldError{...}
    }
    if ov.defs[name].Type != string(TypeFloat) {
        return &errors.TypeError{...}
    }
    ov.fields[name] = NewFloatValue(value)
    ov.written[name] = true
    return nil
}

// GetString returns a string output value
func (ov *OutputValues) GetString(name string) (string, error) {
    fv, ok := ov.fields[name]
    if !ok {
        return "", &errors.UnknownFieldError{...}
    }
    return fv.String()
}

// GetInt returns an int output value
func (ov *OutputValues) GetInt(name string) (int, error) {
    fv, ok := ov.fields[name]
    if !ok {
        return 0, &errors.UnknownFieldError{...}
    }
    return fv.Int()
}

// GetBool returns a bool output value
func (ov *OutputValues) GetBool(name string) (bool, error) {
    fv, ok := ov.fields[name]
    if !ok {
        return false, &errors.UnknownFieldError{...}
    }
    return fv.Bool()
}

// GetFloat returns a float output value
func (ov *OutputValues) GetFloat(name string) (float64, error) {
    fv, ok := ov.fields[name]
    if !ok {
        return 0, &errors.UnknownFieldError{...}
    }
    return fv.Float()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/variables/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/variables/typed.go pkg/flows/variables/typed_test.go
git commit -m "feat(variables): add OutputValues container with write-once semantics"
```

---

## Chunk 3: Expression Evaluation

This chunk implements the expression parser and evaluator.

### Task 8: Create Expression Types

**Files:**
- Create: `pkg/flows/variables/expression.go`
- Test: `pkg/flows/variables/expression_test.go`

- [ ] **Step 1: Write failing test for expression parsing**

```go
// pkg/flows/variables/expression_test.go
package variables_test

import (
    "testing"

    "github.com/denkhaus/gollum/pkg/flows/variables"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestExpressionParser_Parse(t *testing.T) {
    parser := variables.NewExpressionParser()

    expr, err := parser.Parse("GT(context.x, 10)")
    require.NoError(t, err)
    assert.Equal(t, "GT", expr.Operator)
    assert.Len(t, expr.Args, 2)
}

func TestExpressionParser_ParseInvalid(t *testing.T) {
    parser := variables.NewExpressionParser()

    _, err := parser.Parse("INVALID(context.x, 10)")
    assert.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/variables/...`
Expected: FAIL with "undefined: variables.NewExpressionParser"

- [ ] **Step 3: Write minimal implementation**

```go
// pkg/flows/variables/expression.go
package variables

import (
    "fmt"
    "regexp"
    "strings"
)

// Expression represents a parsed expression
type Expression struct {
    Operator string
    Args     []string
}

// ExpressionParser parses expression strings
type ExpressionParser struct {
    // Could add regex cache here
}

// NewExpressionParser creates a new parser
func NewExpressionParser() *ExpressionParser {
    return &ExpressionParser{}
}

// Parse parses an expression string
// Format: OPERATOR(arg1, arg2, ...)
func (ep *ExpressionParser) Parse(expr string) (*Expression, error) {
    expr = strings.TrimSpace(expr)

    // Match: OPERATOR(...)
    re := regexp.MustCompile(`^([A-Z]+)\((.*)\)$`)
    matches := re.FindStringSubmatch(expr)
    if matches == nil {
        return nil, fmt.Errorf("invalid expression format: %s", expr)
    }

    operator := matches[1]
    argsStr := matches[2]

    // Validate operator
    validOperators := map[string]bool{
        "GT": true, "LT": true, "GTE": true, "LTE": true,
        "EQ": true, "NEQ": true,
        "AND": true, "OR": true, "NOT": true,
        "ADD": true, "SUB": true, "MUL": true, "DIV": true,
    }

    if !validOperators[operator] {
        return nil, fmt.Errorf("unknown operator: %s", operator)
    }

    // Parse arguments (simple split by comma for now)
    // TODO: Handle nested expressions
    var args []string
    if argsStr != "" {
        for _, arg := range strings.Split(argsStr, ",") {
            args = append(args, strings.TrimSpace(arg))
        }
    }

    return &Expression{
        Operator: operator,
        Args:     args,
    }, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/variables/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/variables/expression.go pkg/flows/variables/expression_test.go
git commit -m "feat(variables): add expression parser"
```

### Task 9: Add Dependency Extraction

**Files:**
- Modify: `pkg/flows/variables/expression.go`
- Modify: `pkg/flows/variables/expression_test.go`

- [ ] **Step 1: Write failing test for dependency extraction**

```go
func TestExpressionParser_ExtractDependencies(t *testing.T) {
    parser := variables.NewExpressionParser()

    deps := parser.ExtractDependencies("GT(context.x, 10)")

    assert.Len(t, deps, 1)
    assert.Equal(t, "context", deps[0].Scope)
    assert.Equal(t, "x", deps[0].Name)
}

func TestExpressionParser_ExtractDependenciesMultiple(t *testing.T) {
    parser := variables.NewExpressionParser()

    deps := parser.ExtractDependencies("AND(input.a, context.b, output.c)")

    assert.Len(t, deps, 3)

    scopes := []string{}
    names := []string{}
    for _, dep := range deps {
        scopes = append(scopes, dep.Scope)
        names = append(names, dep.Name)
    }

    assert.Contains(t, scopes, "input")
    assert.Contains(t, scopes, "context")
    assert.Contains(t, scopes, "output")
    assert.Contains(t, names, "a")
    assert.Contains(t, names, "b")
    assert.Contains(t, names, "c")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/variables/...`
Expected: FAIL with "undefined: variables.FieldReference" or method not found

- [ ] **Step 3: Write minimal implementation**

```go
// pkg/flows/variables/expression.go (add before ExpressionParser)

// FieldReference represents a reference to a field
type FieldReference struct {
    Scope string // "input", "context", "output", "computed"
    Name  string
}

// ExtractDependencies extracts field references from an expression
func (ep *ExpressionParser) ExtractDependencies(expr string) []FieldReference {
    var deps []FieldReference

    // Match patterns like: scope.fieldName
    re := regexp.MustCompile(`(input|context|output|computed)\.([a-zA-Z_][a-zA-Z0-9_]*)`)
    matches := re.FindAllStringSubmatch(expr, -1)

    for _, match := range matches {
        deps = append(deps, FieldReference{
            Scope: match[1],
            Name:  match[2],
        })
    }

    return deps
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/variables/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/variables/expression.go pkg/flows/variables/expression_test.go
git commit -m "feat(variables): add dependency extraction from expressions"
```

### Task 10: Add Expression Evaluation

**Files:**
- Modify: `pkg/flows/variables/expression.go`
- Modify: `pkg/flows/variables/expression_test.go`

- [ ] **Step 1: Write failing test for expression evaluation**

```go
func TestExpressionEvaluator_EvaluateGT(t *testing.T) {
    evaluator := variables.NewExpressionEvaluator()
    scope := &variables.EvaluationScope{
        Context: map[string]any{"x": 15},
    }

    result, err := evaluator.Evaluate("GT(context.x, 10)", scope)
    require.NoError(t, err)
    assert.True(t, result.(bool))
}

func TestExpressionEvaluator_EvaluateAND(t *testing.T) {
    evaluator := variables.NewExpressionEvaluator()
    scope := &variables.EvaluationScope{
        Context: map[string]any{
            "a": true,
            "b": true,
        },
    }

    result, err := evaluator.Evaluate("AND(context.a, context.b)", scope)
    require.NoError(t, err)
    assert.True(t, result.(bool))
}

func TestExpressionEvaluator_EvaluateADD(t *testing.T) {
    evaluator := variables.NewExpressionEvaluator()
    scope := &variables.EvaluationScope{
        Input: map[string]any{"x": 5, "y": 3},
    }

    result, err := evaluator.Evaluate("ADD(input.x, input.y)", scope)
    require.NoError(t, err)
    assert.Equal(t, 8, result)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/variables/...`
Expected: FAIL with "undefined: variables.NewExpressionEvaluator"

- [ ] **Step 3: Write minimal implementation**

```go
// pkg/flows/variables/expression.go (add after ExpressionParser)

// EvaluationScope provides values for expression evaluation
type EvaluationScope struct {
    Input    map[string]any
    Context  map[string]any
    Output   map[string]any
    Computed map[string]any
}

// ExpressionEvaluator evaluates expressions
type ExpressionEvaluator struct {
    parser *ExpressionParser
}

// NewExpressionEvaluator creates a new evaluator
func NewExpressionEvaluator() *ExpressionEvaluator {
    return &ExpressionEvaluator{
        parser: NewExpressionParser(),
    }
}

// Evaluate evaluates an expression against the given scope
func (ee *ExpressionEvaluator) Evaluate(expr string, scope *EvaluationScope) (any, error) {
    parsed, err := ee.parser.Parse(expr)
    if err != nil {
        return nil, err
    }

    // Resolve arguments to values
    args, err := ee.resolveArgs(parsed.Args, scope)
    if err != nil {
        return nil, err
    }

    // Apply operator
    return ee.applyOperator(parsed.Operator, args)
}

// resolveArgs resolves argument references to values
func (ee *ExpressionEvaluator) resolveArgs(args []string, scope *EvaluationScope) ([]any, error) {
    resolved := make([]any, len(args))

    for i, arg := range args {
        // Check if it's a field reference
        re := regexp.MustCompile(`^(input|context|output|computed)\.([a-zA-Z_][a-zA-Z0-9_]*)$`)
        matches := re.FindStringSubmatch(arg)

        if matches != nil {
            // It's a field reference
            scopeName := matches[1]
            fieldName := matches[2]

            var scopeMap map[string]any
            switch scopeName {
            case "input":
                scopeMap = scope.Input
            case "context":
                scopeMap = scope.Context
            case "output":
                scopeMap = scope.Output
            case "computed":
                scopeMap = scope.Computed
            }

            val, ok := scopeMap[fieldName]
            if !ok {
                return nil, fmt.Errorf("field not found: %s.%s", scopeName, fieldName)
            }
            resolved[i] = val
        } else {
            // It's a literal value (number or bool)
            // Try parsing as int
            if intVal, err := strconv.Atoi(arg); err == nil {
                resolved[i] = intVal
                continue
            }
            // Try parsing as bool
            if boolVal, err := strconv.ParseBool(arg); err == nil {
                resolved[i] = boolVal
                continue
            }
            // It's a string
            resolved[i] = arg
        }
    }

    return resolved, nil
}

// applyOperator applies the operator to the arguments
func (ee *ExpressionEvaluator) applyOperator(op string, args []any) (any, error) {
    switch op {
    case "GT":
        return compare(args, ">")
    case "LT":
        return compare(args, "<")
    case "GTE":
        return compare(args, ">=")
    case "LTE":
        return compare(args, "<=")
    case "EQ":
        return compare(args, "==")
    case "NEQ":
        return compare(args, "!=")
    case "AND":
        return logicalAnd(args)
    case "OR":
        return logicalOr(args)
    case "NOT":
        return logicalNot(args)
    case "ADD":
        return arithmetic(args, "+")
    case "SUB":
        return arithmetic(args, "-")
    case "MUL":
        return arithmetic(args, "*")
    case "DIV":
        return arithmetic(args, "/")
    default:
        return nil, fmt.Errorf("unknown operator: %s", op)
    }
}

// compare performs comparison operations
func compare(args []any, op string) (bool, error) {
    if len(args) != 2 {
        return false, fmt.Errorf("comparison requires 2 arguments, got %d", len(args))
    }

    // Convert to comparable types
    a, b := args[0], args[1]

    // Handle int comparisons
    aInt, aOk := toInt64(a)
    bInt, bOk := toInt64(b)
    if aOk && bOk {
        switch op {
        case ">":
            return aInt > bInt, nil
        case "<":
            return aInt < bInt, nil
        case ">=":
            return aInt >= bInt, nil
        case "<=":
            return aInt <= bInt, nil
        case "==":
            return aInt == bInt, nil
        case "!=":
            return aInt != bInt, nil
        }
    }

    // Handle string comparisons
    aStr, aOk := a.(string)
    bStr, bOk := b.(string)
    if aOk && bOk {
        switch op {
        case "==":
            return aStr == bStr, nil
        case "!=":
            return aStr != bStr, nil
        }
    }

    return false, fmt.Errorf("cannot compare %T and %T", a, b)
}

// logicalAnd performs logical AND
func logicalAnd(args []any) (bool, error) {
    for _, arg := range args {
        b, ok := toBool(arg)
        if !ok {
            return false, fmt.Errorf("AND requires bool arguments, got %T", arg)
        }
        if !b {
            return false, nil
        }
    }
    return true, nil
}

// logicalOr performs logical OR
func logicalOr(args []any) (bool, error) {
    for _, arg := range args {
        b, ok := toBool(arg)
        if !ok {
            return false, fmt.Errorf("OR requires bool arguments, got %T", arg)
        }
        if b {
            return true, nil
        }
    }
    return false, nil
}

// logicalNot performs logical NOT
func logicalNot(args []any) (bool, error) {
    if len(args) != 1 {
        return false, fmt.Errorf("NOT requires 1 argument, got %d", len(args))
    }
    b, ok := toBool(args[0])
    if !ok {
        return false, fmt.Errorf("NOT requires bool argument, got %T", args[0])
    }
    return !b, nil
}

// arithmetic performs arithmetic operations
func arithmetic(args []any, op string) (int, error) {
    if len(args) != 2 {
        return 0, fmt.Errorf("%s requires 2 arguments, got %d", op, len(args))
    }

    a, ok := toInt64(args[0])
    if !ok {
        return 0, fmt.Errorf("arithmetic requires int arguments, got %T", args[0])
    }
    b, ok := toInt64(args[1])
    if !ok {
        return 0, fmt.Errorf("arithmetic requires int arguments, got %T", args[1])
    }

    switch op {
    case "+":
        return int(a + b), nil
    case "-":
        return int(a - b), nil
    case "*":
        return int(a * b), nil
    case "/":
        if b == 0 {
            return 0, fmt.Errorf("division by zero")
        }
        return int(a / b), nil
    default:
        return 0, fmt.Errorf("unknown arithmetic operator: %s", op)
    }
}

// toInt64 converts a value to int64
func toInt64(v any) (int64, bool) {
    switch val := v.(type) {
    case int:
        return int64(val), true
    case int32:
        return int64(val), true
    case int64:
        return val, true
    case float64:
        return int64(val), true
    default:
        return 0, false
    }
}

// toBool converts a value to bool
func toBool(v any) (bool, bool) {
    b, ok := v.(bool)
    return b, ok
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/variables/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/variables/expression.go pkg/flows/variables/expression_test.go
git commit -m "feat(variables): add expression evaluation"
```

---

## Chunk 4: Computed Values and Reactivity

This chunk implements the reactive computed field system.

### Task 11: Create ComputedValues Container

**Files:**
- Create: `pkg/flows/variables/computed.go`
- Test: `pkg/flows/variables/computed_test.go`

- [ ] **Step 1: Write failing test for ComputedValues**

```go
// pkg/flows/variables/computed_test.go
package variables_test

import (
    "testing"

    "github.com/denkhaus/gollum/pkg/flows"
    "github.com/denkhaus/gollum/pkg/flows/variables"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestComputedValues_Register(t *testing.T) {
    block := &flows.ComputedBlock{
        Bools: []flows.ComputedFieldDef{
            {Name: "is_large", Type: "bool", Eval: "GT(context.x, 10)"},
        },
    }

    computed := variables.NewComputedValues(block)

    assert.True(t, computed.Has("is_large"))
    assert.False(t, computed.Has("unknown"))
}

func TestComputedValues_GetBeforeEvaluate(t *testing.T) {
    block := &flows.ComputedBlock{
        Bools: []flows.ComputedDef{
            {Name: "is_large", Type: "bool", Eval: "GT(context.x, 10)"},
        },
    }

    computed := variables.NewComputedValues(block)

    // Getting before evaluating should return error or dirty flag
    _, err := computed.GetBool("is_large")
    assert.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/variables/...`
Expected: FAIL with "undefined: variables.NewComputedValues"

- [ ] **Step 3: Write minimal implementation**

```go
// pkg/flows/variables/computed.go
package variables

import (
    "fmt"

    "github.com/denkhaus/gollum/pkg/flows"
    "github.com/denkhaus/gollum/pkg/flows/errors"
)

// ComputedField represents a single computed field
type ComputedField struct {
    Name         string
    Type         ValueType
    Expression   string
    Dependencies []FieldReference
    lastValue    *FieldValue
    dirty        bool
}

// ComputedValues stores computed field definitions
type ComputedValues struct {
    fields map[string]*ComputedField
}

// NewComputedValues creates a new ComputedValues from ComputedBlock
func NewComputedValues(block *flows.ComputedBlock) *ComputedValues {
    cv := &ComputedValues{
        fields: make(map[string]*ComputedField),
    }

    if block == nil {
        return cv
    }

    parser := NewExpressionParser()

    // Register bool computed fields
    for _, field := range block.Bools {
        deps := parser.ExtractDependencies(field.Eval)
        cv.fields[field.Name] = &ComputedField{
            Name:         field.Name,
            Type:         TypeBool,
            Expression:   field.Eval,
            Dependencies: deps,
            dirty:        true, // Needs initial evaluation
        }
    }

    // Register int computed fields
    for _, field := range block.Ints {
        deps := parser.ExtractDependencies(field.Eval)
        cv.fields[field.Name] = &ComputedField{
            Name:         field.Name,
            Type:         TypeInt,
            Expression:   field.Eval,
            Dependencies: deps,
            dirty:        true,
        }
    }

    // Register string computed fields
    for _, field := range block.Strings {
        deps := parser.ExtractDependencies(field.Eval)
        cv.fields[field.Name] = &ComputedField{
            Name:         field.Name,
            Type:         TypeString,
            Expression:   field.Eval,
            Dependencies: deps,
            dirty:        true,
        }
    }

    // Register float computed fields
    for _, field := range block.Floats {
        deps := parser.ExtractDependencies(field.Eval)
        cv.fields[field.Name] = &ComputedField{
            Name:         field.Name,
            Type:         TypeFloat,
            Expression:   field.Eval,
            Dependencies: deps,
            dirty:        true,
        }
    }

    return cv
}

// Has returns true if the computed field exists
func (cv *ComputedValues) Has(name string) bool {
    _, ok := cv.fields[name]
    return ok
}

// GetField returns the computed field definition
func (cv *ComputedValues) GetField(name string) (*ComputedField, error) {
    field, ok := cv.fields[name]
    if !ok {
        return nil, &errors.UnknownFieldError{
            FlowError: errors.FlowError{
                Code:    errors.ErrCodeUnknownField,
                Message: "computed field not defined",
                Field:   name,
            },
            Scope: "computed",
        }
    }
    return field, nil
}

// MarkDirty marks a computed field as needing re-evaluation
func (cv *ComputedValues) MarkDirty(name string) {
    if field, ok := cv.fields[name]; ok {
        field.dirty = true
    }
}

// IsDirty returns true if the field needs evaluation
func (cv *ComputedValues) IsDirty(name string) bool {
    if field, ok := cv.fields[name]; ok {
        return field.dirty
    }
    return false
}

// SetValue sets the cached value for a computed field
func (cv *ComputedValues) SetValue(name string, value FieldValue) {
    if field, ok := cv.fields[name]; ok {
        field.lastValue = &value
        field.dirty = false
    }
}

// GetBool returns a bool computed value
func (cv *ComputedValues) GetBool(name string) (bool, error) {
    field, err := cv.GetField(name)
    if err != nil {
        return false, err
    }
    if field.lastValue == nil || field.dirty {
        return false, fmt.Errorf("computed field not evaluated: %s", name)
    }
    return field.lastValue.Bool()
}

// GetInt returns an int computed value
func (cv *ComputedValues) GetInt(name string) (int, error) {
    field, err := cv.GetField(name)
    if err != nil {
        return 0, err
    }
    if field.lastValue == nil || field.dirty {
        return 0, fmt.Errorf("computed field not evaluated: %s", name)
    }
    return field.lastValue.Int()
}

// GetString returns a string computed value
func (cv *ComputedValues) GetString(name string) (string, error) {
    field, err := cv.GetField(name)
    if err != nil {
        return "", err
    }
    if field.lastValue == nil || field.dirty {
        return "", fmt.Errorf("computed field not evaluated: %s", name)
    }
    return field.lastValue.String()
}

// GetFloat returns a float computed value
func (cv *ComputedValues) GetFloat(name string) (float64, error) {
    field, err := cv.GetField(name)
    if err != nil {
        return 0, err
    }
    if field.lastValue == nil || field.dirty {
        return 0, fmt.Errorf("computed field not evaluated: %s", name)
    }
    return field.lastValue.Float()
}

// GetAll returns all computed field definitions
func (cv *ComputedValues) GetAll() map[string]*ComputedField {
    return cv.fields
}

// GetDependents returns all computed fields that depend on the given field
func (cv *ComputedValues) GetDependents(fieldRef FieldReference) []string {
    var dependents []string

    for _, field := range cv.fields {
        for _, dep := range field.Dependencies {
            if dep.Scope == fieldRef.Scope && dep.Name == fieldRef.Name {
                dependents = append(dependents, field.Name)
                break
            }
        }
    }

    return dependents
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/variables/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/variables/computed.go pkg/flows/variables/computed_test.go
git commit -m "feat(variables): add ComputedValues container"
```

### Task 12: Implement ComputedEvaluator with Reactivity

**Files:**
- Modify: `pkg/flows/variables/computed.go`
- Modify: `pkg/flows/variables/computed_test.go`

- [ ] **Step 1: Write failing test for reactive evaluation**

```go
func TestComputedEvaluator_ReactiveUpdate(t *testing.T) {
    // Setup
    computedBlock := &flows.ComputedBlock{
        Bools: []flows.ComputedFieldDef{
            {Name: "is_large", Type: "bool", Eval: "GT(context.x, 10)"},
        },
    }
    computed := variables.NewComputedValues(computedBlock)

    contextBlock := &flows.ContextBlock{
        Ints: []flows.ContextField{{Name: "x", Type: "int"}},
    }
    context := variables.NewContextValues(contextBlock)

    evaluator := variables.NewComputedEvaluator(computed, nil, context, nil)
    context.SetEvaluator(evaluator)

    // Initial evaluation
    context.SetInt("x", 15)
    err := evaluator.ComputeDirty()
    require.NoError(t, err)

    val, err := computed.GetBool("is_large")
    require.NoError(t, err)
    assert.True(t, val) // 15 > 10

    // Change dependency
    context.SetInt("x", 5)
    err = evaluator.ComputeDirty()
    require.NoError(t, err)

    val, err = computed.GetBool("is_large")
    require.NoError(t, err)
    assert.False(t, val) // 5 < 10
}

func TestComputedEvaluator_CircularDependency(t *testing.T) {
    computedBlock := &flows.ComputedBlock{
        Bools: []flows.ComputedFieldDef{
            {Name: "a", Type: "bool", Eval: "computed.b"},
            {Name: "b", Type: "bool", Eval: "computed.a"},
        },
    }
    computed := variables.NewComputedValues(computedBlock)

    evaluator := variables.NewComputedEvaluator(computed, nil, nil, nil)

    err := evaluator.ComputeDirty()
    assert.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/variables/...`
Expected: FAIL with "undefined: variables.NewComputedEvaluator"

- [ ] **Step 3: Write minimal implementation**

```go
// pkg/flows/variables/computed.go (add after ComputedValues)

// ComputedEvaluator manages reactive computed field evaluation
type ComputedEvaluator struct {
    computed   *ComputedValues
    input      *InputValues
    context    *ContextValues
    output     *OutputValues
    exprEval   *ExpressionEvaluator
    dependents map[string][]string // fieldRef -> computed fields that depend on it
    evaluating map[string]bool     // for cycle detection
}

// NewComputedEvaluator creates a new computed evaluator
func NewComputedEvaluator(computed *ComputedValues, input *InputValues,
    context *ContextValues, output *OutputValues) *ComputedEvaluator {

    evaluator := &ComputedEvaluator{
        computed:   computed,
        input:      input,
        context:    context,
        output:     output,
        exprEval:   NewExpressionEvaluator(),
        dependents: make(map[string][]string),
        evaluating: make(map[string]bool),
    }

    // Build dependency graph
    evaluator.buildDependentsMap()

    return evaluator
}

// buildDependentsMap builds the reverse dependency graph
func (ce *ComputedEvaluator) buildDependentsMap() {
    for _, field := range ce.computed.GetAll() {
        for _, dep := range field.Dependencies {
            key := dep.Scope + "." + dep.Name
            ce.dependents[key] = append(ce.dependents[key], field.Name)
        }
    }
}

// MarkDirty marks a computed field and its dependents as dirty
func (ce *ComputedEvaluator) MarkDirty(changedField string) {
    // This is called when a context/input/output field changes
    // We need to mark all computed fields that depend on it

    // The changedField is just the name, we need to figure out the scope
    // For now, let's assume context scope (most common case)
    key := "context." + changedField

    // Mark direct dependents
    for _, depName := range ce.dependents[key] {
        ce.computed.MarkDirty(depName)
        // Recursively mark dependents of dependents
        ce.markDependentsDirty(depName)
    }
}

// markDependentsDirty recursively marks dependents dirty
func (ce *ComputedEvaluator) markDependentsDirty(computedName string) {
    fieldRef := FieldReference{Scope: "computed", Name: computedName}
    key := "computed." + computedName

    for _, depName := range ce.dependents[key] {
        ce.computed.MarkDirty(depName)
        ce.markDependentsDirty(depName)
    }
}

// ComputeDirty evaluates all dirty computed fields
func (ce *ComputedEvaluator) ComputeDirty() error {
    evaluated := make(map[string]bool)

    for name, field := range ce.computed.GetAll() {
        if field.dirty && !evaluated[name] {
            if err := ce.evaluateField(name, evaluated); err != nil {
                return err
            }
        }
    }

    return nil
}

// evaluateField evaluates a single computed field (with dependency evaluation)
func (ce *ComputedEvaluator) evaluateField(name string, evaluated map[string]bool) error {
    // Check for cycles
    if ce.evaluating[name] {
        return &errors.CircularDependencyError{
            FlowError: errors.FlowError{
                Code:    errors.ErrCodeCircularDep,
                Message: "circular dependency detected",
            },
            Cycle: ce.extractCycle(name),
        }
    }

    ce.evaluating[name] = true
    defer delete(ce.evaluating, name)

    field, err := ce.computed.GetField(name)
    if err != nil {
        return err
    }

    // First, evaluate all dependencies
    for _, dep := range field.Dependencies {
        if dep.Scope == "computed" && !evaluated[dep.Name] {
            if err := ce.evaluateField(dep.Name, evaluated); err != nil {
                return err
            }
        }
    }

    // Build evaluation scope
    scope := ce.buildScope()

    // Evaluate the expression
    result, err := ce.exprEval.Evaluate(field.Expression, scope)
    if err != nil {
        return &errors.ExpressionError{
            FlowError: errors.FlowError{
                Code:    errors.ErrCodeExpression,
                Message: "expression evaluation failed",
                Field:   name,
                Cause:   err,
            },
            Expression: field.Expression,
        }
    }

    // Store the result
    var fv FieldValue
    switch field.Type {
    case TypeBool:
        boolVal, ok := result.(bool)
        if !ok {
            return &errors.TypeError{...}
        }
        fv = NewBoolValue(boolVal)
    case TypeInt:
        intVal, ok := result.(int)
        if !ok {
            // Try converting float64
            floatVal, ok := result.(float64)
            if !ok {
                return &errors.TypeError{...}
            }
            intVal = int(floatVal)
        }
        fv = NewIntValue(intVal)
    case TypeString:
        strVal, ok := result.(string)
        if !ok {
            return &errors.TypeError{...}
        }
        fv = NewStringValue(strVal)
    case TypeFloat:
        floatVal, ok := result.(float64)
        if !ok {
            return &errors.TypeError{...}
        }
        fv = NewFloatValue(floatVal)
    }

    ce.computed.SetValue(name, fv)
    evaluated[name] = true

    return nil
}

// buildScope builds the evaluation scope
func (ce *ComputedEvaluator) buildScope() *EvaluationScope {
    scope := &EvaluationScope{
        Input:    make(map[string]any),
        Context:  make(map[string]any),
        Output:   make(map[string]any),
        Computed: make(map[string]any),
    }

    // Populate input scope
    if ce.input != nil {
        for name := range ce.input.defs {
            if val, err := ce.input.GetString(name); err == nil {
                scope.Input[name] = val
            } else if val, err := ce.input.GetInt(name); err == nil {
                scope.Input[name] = val
            } else if val, err := ce.input.GetBool(name); err == nil {
                scope.Input[name] = val
            } else if val, err := ce.input.GetFloat(name); err == nil {
                scope.Input[name] = val
            }
        }
    }

    // Populate context scope
    if ce.context != nil {
        for name := range ce.context.defs {
            if val, err := ce.context.GetString(name); err == nil {
                scope.Context[name] = val
            } else if val, err := ce.context.GetInt(name); err == nil {
                scope.Context[name] = val
            } else if val, err := ce.context.GetBool(name); err == nil {
                scope.Context[name] = val
            } else if val, err := ce.context.GetFloat(name); err == nil {
                scope.Context[name] = val
            }
        }
    }

    // Populate output scope
    if ce.output != nil {
        for name := range ce.output.defs {
            if val, err := ce.output.GetString(name); err == nil {
                scope.Output[name] = val
            } else if val, err := ce.output.GetInt(name); err == nil {
                scope.Output[name] = val
            } else if val, err := ce.output.GetBool(name); err == nil {
                scope.Output[name] = val
            } else if val, err := ce.output.GetFloat(name); err == nil {
                scope.Output[name] = val
            }
        }
    }

    // Populate computed scope (already evaluated values)
    for name, field := range ce.computed.GetAll() {
        if field.lastValue != nil && !field.dirty {
            if val, err := field.lastValue.Bool(); err == nil {
                scope.Computed[name] = val
            } else if val, err := field.lastValue.Int(); err == nil {
                scope.Computed[name] = val
            } else if val, err := field.lastValue.String(); err == nil {
                scope.Computed[name] = val
            } else if val, err := field.lastValue.Float(); err == nil {
                scope.Computed[name] = val
            }
        }
    }

    return scope
}

// extractCycle extracts the cycle from the current evaluation state
func (ce *ComputedEvaluator) extractCycle(startNode string) []string {
    cycle := []string{startNode}
    current := startNode

    // Simple cycle extraction - follow the evaluating chain
    for name := range ce.evaluating {
        if name != startNode {
            cycle = append([]string{name}, cycle...)
        }
    }

    return cycle
}

// GetValue returns a computed value (evaluates if dirty)
func (ce *ComputedEvaluator) GetValue(name string) (any, error) {
    field, err := ce.computed.GetField(name)
    if err != nil {
        return nil, err
    }

    if field.dirty {
        if err := ce.ComputeDirty(); err != nil {
            return nil, err
        }
    }

    switch field.Type {
    case TypeBool:
        return ce.computed.GetBool(name)
    case TypeInt:
        return ce.computed.GetInt(name)
    case TypeString:
        return ce.computed.GetString(name)
    case TypeFloat:
        return ce.computed.GetFloat(name)
    default:
        return nil, fmt.Errorf("unknown type: %s", field.Type)
    }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/variables/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/variables/computed.go pkg/flows/variables/computed_test.go
git commit -m "feat(variables): add ComputedEvaluator with reactivity"
```

---

## Chunk 5: Update Flow Types

This chunk updates the flow types to support the new computed section.

### Task 13: Add ComputedBlock to Flow Types

**Files:**
- Modify: `pkg/flows/types.go`
- Modify: `pkg/flows/types_test.go`

- [ ] **Step 1: Write failing test for ComputedBlock**

```go
// pkg/flows/types_test.go
func TestFlow_ComputedBlock(t *testing.T) {
    xml := `
    <flow name="test" version="1.0">
        <computed>
            <bool name="is_large" eval="GT(context.x, 10)" />
            <int name="doubled" eval="MUL(input.y, 2)" />
        </computed>
    </flow>
    `

    var flow flows.Flow
    err := xml.Unmarshal([]byte(xml), &flow)
    require.NoError(t, err)

    require.NotNil(t, flow.Computed)
    assert.Len(t, flow.Computed.Bools, 1)
    assert.Equal(t, "is_large", flow.Computed.Bools[0].Name)
    assert.Equal(t, "bool", flow.Computed.Bools[0].Type)
    assert.Equal(t, "GT(context.x, 10)", flow.Computed.Bools[0].Eval)

    assert.Len(t, flow.Computed.Ints, 1)
    assert.Equal(t, "doubled", flow.Computed.Ints[0].Name)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/...`
Expected: FAIL with "undefined: flows.ComputedBlock"

- [ ] **Step 3: Write minimal implementation**

```go
// pkg/flows/types.go (add after OutputBlock, before Flow struct)

// ComputedBlock defines computed fields (top-level section)
type ComputedBlock struct {
    Bools   []ComputedFieldDef `xml:"bool"`
    Ints    []ComputedFieldDef `xml:"int"`
    Strings []ComputedFieldDef `xml:"string"`
    Floats  []ComputedFieldDef `xml:"float"`
}

// GetAll returns all computed field definitions
func (cb *ComputedBlock) GetAll() []ComputedFieldDef {
    var fields []ComputedFieldDef
    fields = append(fields, cb.Bools...)
    fields = append(fields, cb.Ints...)
    fields = append(fields, cb.Strings...)
    fields = append(fields, cb.Floats...)
    return fields
}

// ComputedFieldDef defines a computed field
type ComputedFieldDef struct {
    XMLName xml.Name `xml:"bool,int,string,float"`
    Name    string   `xml:"name,attr"`
    Type    string   `xml:"type,attr"`
    Eval    string   `xml:"eval,attr"` // Changed from "when"
}

// Then add to Flow struct:
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

// ContextBlock - remove Computeds field (deprecated but keep for backward compat)
type ContextBlock struct {
    Strings   []ContextField  `xml:"string"`
    Ints      []ContextField  `xml:"int"`
    Bools     []ContextField  `xml:"bool"`
    Floats    []ContextField  `xml:"float"`
    Objects   []ObjectDef     `xml:"object"`
    Computeds []ComputedField `xml:"computed"` // DEPRECATED: Use top-level <computed>
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/types.go pkg/flows/types_test.go
git commit -m "feat(flows): add ComputedBlock to flow types"
```

---

## Chunk 6: ExecutionContext Integration

This chunk creates the new ExecutionContext that replaces the old Context.

### Task 14: Create ExecutionContext

**Files:**
- Create: `pkg/flows/executor/execution_context.go`
- Test: `pkg/flows/executor/execution_context_test.go`

- [ ] **Step 1: Write failing test for ExecutionContext**

```go
// pkg/flows/executor/execution_context_test.go
package executor_test

import (
    "testing"

    "github.com/denkhaus/gollum/pkg/flows"
    "github.com/denkhaus/gollum/pkg/flows/executor"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestExecutionContext_SetInput(t *testing.T) {
    flow := &flows.Flow{
        Input: &flows.InputBlock{
            Strings: []flows.FieldDef{{Name: "name", Type: "string"}},
            Ints:    []flows.FieldDef{{Name: "count", Type: "int"}},
        },
    }

    ctx, err := executor.NewExecutionContext(flow)
    require.NoError(t, err)

    err = ctx.SetInput(map[string]string{
        "name":  "test",
        "count": "42",
    })
    require.NoError(t, err)

    val, err := ctx.GetInputField("name")
    require.NoError(t, err)
    assert.Equal(t, "test", val)

    count, err := ctx.GetInputField("count")
    require.NoError(t, err)
    assert.Equal(t, 42, count)
}

func TestExecutionContext_SetContextField(t *testing.T) {
    flow := &flows.Flow{
        Context: &flows.ContextBlock{
            Strings: []flows.ContextField{{Name: "status", Type: "string"}},
            Ints:    []flows.ContextField{{Name: "value", Type: "int"}},
        },
        Computed: &flows.ComputedBlock{
            Bools: []flows.ComputedFieldDef{
                {Name: "is_ready", Type: "bool", Eval: "EQ(context.status, \"ready\")"},
            },
        },
    }

    ctx, err := executor.NewExecutionContext(flow)
    require.NoError(t, err)

    err = ctx.SetContextField("status", "ready")
    require.NoError(t, err)

    val, err := ctx.GetContextField("status")
    require.NoError(t, err)
    assert.Equal(t, "ready", val)

    // Check computed was evaluated
    isReady, err := ctx.GetComputedField("is_ready")
    require.NoError(t, err)
    assert.True(t, isReady.(bool))
}

func TestExecutionContext_SetOutputField(t *testing.T) {
    flow := &flows.Flow{
        Output: &flows.OutputBlock{
            Strings: []flows.FieldDef{{Name: "result", Type: "string"}},
        },
    }

    ctx, err := executor.NewExecutionContext(flow)
    require.NoError(t, err)

    err = ctx.SetOutputField("result", "done")
    require.NoError(t, err)

    val, err := ctx.GetOutputField("result")
    require.NoError(t, err)
    assert.Equal(t, "done", val)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/executor/...`
Expected: FAIL with "undefined: executor.NewExecutionContext"

- [ ] **Step 3: Write minimal implementation**

```go
// pkg/flows/executor/execution_context.go
package executor

import (
    "fmt"
    "strconv"

    "github.com/denkhaus/gollum/pkg/flows"
    "github.com/denkhaus/gollum/pkg/flows/variables"
)

// ExecutionContext manages all value containers for a flow execution
type ExecutionContext struct {
    input     *variables.InputValues
    context   *variables.ContextValues
    output    *variables.OutputValues
    computed  *variables.ComputedValues
    evaluator *variables.ComputedEvaluator
}

// NewExecutionContext creates a new execution context from a flow
func NewExecutionContext(flow *flows.Flow) (*ExecutionContext, error) {
    ctx := &ExecutionContext{
        input:    variables.NewInputValues(flow.Input),
        context:  variables.NewContextValues(flow.Context),
        output:   variables.NewOutputValues(flow.Output),
        computed: variables.NewComputedValues(flow.Computed),
    }

    // Create evaluator
    ctx.evaluator = variables.NewComputedEvaluator(
        ctx.computed,
        ctx.input,
        ctx.context,
        ctx.output,
    )

    // Wire up reactivity
    ctx.context.SetEvaluator(ctx.evaluator)

    return ctx, nil
}

// SetInput sets input values from a string map
func (ec *ExecutionContext) SetInput(vals map[string]string) error {
    for name, val := range vals {
        // Determine the field type and call appropriate setter
        if ec.input.defs[name].Type == string(variables.TypeString) {
            if err := ec.input.SetString(name, val); err != nil {
                return err
            }
        } else if ec.input.defs[name].Type == string(variables.TypeInt) {
            intVal, err := strconv.Atoi(val)
            if err != nil {
                return fmt.Errorf("invalid int value for %s: %w", name, err)
            }
            if err := ec.input.SetInt(name, intVal); err != nil {
                return err
            }
        } else if ec.input.defs[name].Type == string(variables.TypeBool) {
            boolVal, err := strconv.ParseBool(val)
            if err != nil {
                return fmt.Errorf("invalid bool value for %s: %w", name, err)
            }
            if err := ec.input.SetBool(name, boolVal); err != nil {
                return err
            }
        } else if ec.input.defs[name].Type == string(variables.TypeFloat) {
            floatVal, err := strconv.ParseFloat(val, 64)
            if err != nil {
                return fmt.Errorf("invalid float value for %s: %w", name, err)
            }
            if err := ec.input.SetFloat(name, floatVal); err != nil {
                return err
            }
        }
    }

    // Trigger computed evaluation after input is set
    if ec.evaluator != nil {
        return ec.evaluator.ComputeDirty()
    }

    return nil
}

// GetInputField returns an input field value
func (ec *ExecutionContext) GetInputField(name string) (any, error) {
    def, ok := ec.input.defs[name]
    if !ok {
        return nil, fmt.Errorf("input field not defined: %s", name)
    }

    switch def.Type {
    case string(variables.TypeString):
        return ec.input.GetString(name)
    case string(variables.TypeInt):
        return ec.input.GetInt(name)
    case string(variables.TypeBool):
        return ec.input.GetBool(name)
    case string(variables.TypeFloat):
        return ec.input.GetFloat(name)
    }

    return nil, fmt.Errorf("unknown type: %s", def.Type)
}

// SetContextField sets a context field value (string)
func (ec *ExecutionContext) SetContextField(name string, value string) error {
    def, ok := ec.context.defs[name]
    if !ok {
        return fmt.Errorf("context field not defined: %s", name)
    }

    switch def.Type {
    case string(variables.TypeString):
        return ec.context.SetString(name, value)
    case string(variables.TypeInt):
        intVal, err := strconv.Atoi(value)
        if err != nil {
            return fmt.Errorf("invalid int value for %s: %w", name, err)
        }
        return ec.context.SetInt(name, intVal)
    case string(variables.TypeBool):
        boolVal, err := strconv.ParseBool(value)
        if err != nil {
            return fmt.Errorf("invalid bool value for %s: %w", name, err)
        }
        return ec.context.SetBool(name, boolVal)
    case string(variables.TypeFloat):
        floatVal, err := strconv.ParseFloat(value, 64)
        if err != nil {
            return fmt.Errorf("invalid float value for %s: %w", name, err)
        }
        return ec.context.SetFloat(name, floatVal)
    }

    return fmt.Errorf("unknown type: %s", def.Type)
}

// GetContextField returns a context field value
func (ec *ExecutionContext) GetContextField(name string) (any, error) {
    def, ok := ec.context.defs[name]
    if !ok {
        return nil, fmt.Errorf("context field not defined: %s", name)
    }

    switch def.Type {
    case string(variables.TypeString):
        return ec.context.GetString(name)
    case string(variables.TypeInt):
        return ec.context.GetInt(name)
    case string(variables.TypeBool):
        return ec.context.GetBool(name)
    case string(variables.TypeFloat):
        return ec.context.GetFloat(name)
    }

    return nil, fmt.Errorf("unknown type: %s", def.Type)
}

// SetOutputField sets an output field value (string)
func (ec *ExecutionContext) SetOutputField(name string, value string) error {
    def, ok := ec.output.defs[name]
    if !ok {
        return fmt.Errorf("output field not defined: %s", name)
    }

    switch def.Type {
    case string(variables.TypeString):
        return ec.output.SetString(name, value)
    case string(variables.TypeInt):
        intVal, err := strconv.Atoi(value)
        if err != nil {
            return fmt.Errorf("invalid int value for %s: %w", name, err)
        }
        return ec.output.SetInt(name, intVal)
    case string(variables.TypeBool):
        boolVal, err := strconv.ParseBool(value)
        if err != nil {
            return fmt.Errorf("invalid bool value for %s: %w", name, err)
        }
        return ec.output.SetBool(name, boolVal)
    case string(variables.TypeFloat):
        floatVal, err := strconv.ParseFloat(value, 64)
        if err != nil {
            return fmt.Errorf("invalid float value for %s: %w", name, err)
        }
        return ec.output.SetFloat(name, floatVal)
    }

    return fmt.Errorf("unknown type: %s", def.Type)
}

// GetOutputField returns an output field value
func (ec *ExecutionContext) GetOutputField(name string) (any, error) {
    def, ok := ec.output.defs[name]
    if !ok {
        return nil, fmt.Errorf("output field not defined: %s", name)
    }

    switch def.Type {
    case string(variables.TypeString):
        return ec.output.GetString(name)
    case string(variables.TypeInt):
        return ec.output.GetInt(name)
    case string(variables.TypeBool):
        return ec.output.GetBool(name)
    case string(variables.TypeFloat):
        return ec.output.GetFloat(name)
    }

    return nil, fmt.Errorf("unknown type: %s", def.Type)
}

// GetComputedField returns a computed field value
func (ec *ExecutionContext) GetComputedField(name string) (any, error) {
    return ec.evaluator.GetValue(name)
}

// BuildInputScope builds a scope map for input (for template substitution)
func (ec *ExecutionContext) BuildInputScope() map[string]any {
    scope := make(map[string]any)
    for name := range ec.input.defs {
        if val, err := ec.GetInputField(name); err == nil {
            scope[name] = val
        }
    }
    return scope
}

// BuildContextScope builds a scope map for context
func (ec *ExecutionContext) BuildContextScope() map[string]any {
    scope := make(map[string]any)
    for name := range ec.context.defs {
        if val, err := ec.GetContextField(name); err == nil {
            scope[name] = val
        }
    }
    return scope
}

// BuildOutputScope builds a scope map for output
func (ec *ExecutionContext) BuildOutputScope() map[string]any {
    scope := make(map[string]any)
    for name := range ec.output.defs {
        if val, err := ec.GetOutputField(name); err == nil {
            scope[name] = val
        }
    }
    return scope
}

// BuildComputedScope builds a scope map for computed
func (ec *ExecutionContext) BuildComputedScope() map[string]any {
    scope := make(map[string]any)
    for name := range ec.computed.GetAll() {
        if val, err := ec.GetComputedField(name); err == nil {
            scope[name] = val
        }
    }
    return scope
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/executor/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/executor/execution_context.go pkg/flows/executor/execution_context_test.go
git commit -m "feat(executor): add ExecutionContext with typed containers"
```

---

## Chunk 7: Update Executor to Use ExecutionContext

This chunk updates the main executor to use the new ExecutionContext.

### Task 15: Update Executor to Use ExecutionContext

**Files:**
- Modify: `pkg/flows/executor/executor.go`
- Modify: `pkg/flows/executor/executor_test.go`

- [ ] **Step 1: Update existing executor tests to pass with new context**

First, let's update the tests to ensure compatibility:
```go
// The existing tests should work with the new ExecutionContext
// We need to ensure the executor.go uses the new context
```

- [ ] **Step 2: Modify executor.go to use ExecutionContext**

```go
// pkg/flows/executor/executor.go
// Change: ctx *Context -> ctx *ExecutionContext
// Update all context field accesses to use the typed API

type flowExecutorImpl struct {
    flow              *flows.Flow
    ctx               *ExecutionContext  // CHANGED from *Context
    currentState      string
    history           *ExecutionHistory
    startTime         time.Time
    // ... rest of fields
}

func (p *flowExecutorServiceImpl) New(flow *flows.Flow) FlowExecutorInstance {
    return &flowExecutorImpl{
        flow:              flow,
        ctx:               NewExecutionContext(flow), // CHANGED
        history:           NewExecutionHistory(),
        startTime:         time.Now(),
        // ... rest of initialization
    }
}

func (p *flowExecutorImpl) SetInput(vals map[string]string) {
    // CHANGED: Use ExecutionContext
    newCtx, _ := NewExecutionContext(p.flow)
    newCtx.SetInput(vals)
    p.ctx = newCtx
}

func (p *flowExecutorImpl) GetContext() *Context {
    // TEMPORARY: Keep old method for backward compatibility
    // Return a wrapper that adapts ExecutionContext to old Context interface
    return adaptContext(p.ctx)
}

// Update substituteTemplate to use typed getters
func (p *flowExecutorImpl) substituteTemplate(cmd string) string {
    // CHANGED: Use ExecutionContext scope builders
    scope := map[string]any{
        "input":    p.ctx.BuildInputScope(),
        "context":  p.ctx.BuildContextScope(),
        "output":   p.ctx.BuildOutputScope(),
        "computed": p.ctx.BuildComputedScope(),
    }
    // ... rest of template substitution
}

// Update all SetContextField/SetOutputField calls to use ExecutionContext
// These should already work through the FlowContext interface
```

- [ ] **Step 3: Run tests to verify compatibility**

Run: `go test ./pkg/flows/executor/...`
Expected: All existing tests pass

- [ ] **Step 4: Commit**

```bash
git add pkg/flows/executor/executor.go pkg/flows/executor/executor_test.go
git commit -m "refactor(executor): use ExecutionContext instead of Context"
```

---

## Chunk 8: Linting Enhancements

This chunk adds linting support for the new computed fields.

### Task 16: Add Computed Field Linting

**Files:**
- Create: `pkg/flows/linter/computed.go`
- Test: `pkg/flows/linter/computed_test.go`

- [ ] **Step 1: Write failing test for computed field linting**

```go
// pkg/flows/linter/computed_test.go
package linter_test

import (
    "testing"

    "github.com/denkhaus/gollum/pkg/flows"
    "github.com/denkhaus/gollum/pkg/flows/linter"
    "github.com/stretchr/testify/assert"
)

func TestLintComputedFields_UndefinedField(t *testing.T) {
    flow := &flows.Flow{
        Computed: &flows.ComputedBlock{
            Bools: []flows.ComputedFieldDef{
                {Name: "test", Type: "bool", Eval: "GT(context.undef, 10)"},
            },
        },
        Context: &flows.ContextBlock{
            Ints: []flows.ContextField{{Name: "x", Type: "int"}},
        },
    }

    errors := linter.CheckComputedFields(flow)

    assert.Len(t, errors, 1)
    assert.Contains(t, errors[0].Message, "Undefined field")
    assert.Contains(t, errors[0].Message, "context.undef")
}

func TestLintComputedFields_CircularDependency(t *testing.T) {
    flow := &flows.Flow{
        Computed: &flows.ComputedBlock{
            Bools: []flows.ComputedFieldDef{
                {Name: "a", Type: "bool", Eval: "computed.b"},
                {Name: "b", Type: "bool", Eval: "computed.a"},
            },
        },
    }

    errors := linter.CheckComputedFields(flow)

    assert.Len(t, errors, 1)
    assert.Contains(t, errors[0].Message, "circular")
}

func TestLintComputedFields_Valid(t *testing.T) {
    flow := &flows.Flow{
        Computed: &flows.ComputedBlock{
            Bools: []flows.ComputedFieldDef{
                {Name: "is_large", Type: "bool", Eval: "GT(context.x, 10)"},
            },
        },
        Context: &flows.ContextBlock{
            Ints: []flows.ContextField{{Name: "x", Type: "int"}},
        },
    }

    errors := linter.CheckComputedFields(flow)

    assert.Empty(t, errors)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/linter/...`
Expected: FAIL with "undefined: linter.CheckComputedFields"

- [ ] **Step 3: Write minimal implementation**

```go
// pkg/flows/linter/computed.go
package linter

import (
    "fmt"

    "github.com/denkhaus/gollum/pkg/flows"
    "github.com/denkhaus/gollum/pkg/flows/variables"
)

// LintError represents a linting error
type LintError struct {
    Level   string // "error" or "warning"
    Message string
    Field   string
}

// CheckComputedFields validates all computed fields in a flow
func CheckComputedFields(flow *flows.Flow) []LintError {
    var errors []LintError

    if flow.Computed == nil {
        return errors
    }

    parser := variables.NewExpressionParser()

    for _, computed := range flow.Computed.GetAll() {
        // 1. Parse expression
        _, err := parser.Parse(computed.Eval)
        if err != nil {
            errors = append(errors, LintError{
                Level:   "error",
                Message: fmt.Sprintf("Invalid expression: %v", err),
                Field:   computed.Name,
            })
            continue
        }

        // 2. Verify referenced fields exist
        deps := parser.ExtractDependencies(computed.Eval)
        for _, dep := range deps {
            if !fieldExists(flow, dep) {
                errors = append(errors, LintError{
                    Level:   "error",
                    Message: fmt.Sprintf("Undefined field: %s.%s", dep.Scope, dep.Name),
                    Field:   computed.Name,
                })
            }
        }
    }

    // 3. Check for circular dependencies
    if cycle := detectCycle(flow.Computed); cycle != nil {
        errors = append(errors, LintError{
            Level:   "error",
            Message: fmt.Sprintf("Circular dependency: %v", cycle),
            Field:   cycle[0],
        })
    }

    return errors
}

// fieldExists checks if a field reference exists in the flow
func fieldExists(flow *flows.Flow, dep variables.FieldReference) bool {
    switch dep.Scope {
    case "input":
        if flow.Input == nil {
            return false
        }
        for _, field := range flow.Input.GetAllFields() {
            if field.Name == dep.Name {
                return true
            }
        }
    case "context":
        if flow.Context == nil {
            return false
        }
        for _, field := range flow.Context.Strings {
            if field.Name == dep.Name {
                return true
            }
        }
        for _, field := range flow.Context.Ints {
            if field.Name == dep.Name {
                return true
            }
        }
        for _, field := range flow.Context.Bools {
            if field.Name == dep.Name {
                return true
            }
        }
        for _, field := range flow.Context.Floats {
            if field.Name == dep.Name {
                return true
            }
        }
    case "output":
        if flow.Output == nil {
            return false
        }
        for _, field := range flow.Output.GetAllFields() {
            if field.Name == dep.Name {
                return true
            }
        }
    case "computed":
        if flow.Computed == nil {
            return false
        }
        for _, field := range flow.Computed.GetAll() {
            if field.Name == dep.Name {
                return true
            }
        }
    }
    return false
}

// detectCycle detects circular dependencies in computed fields
func detectCycle(block *flows.ComputedBlock) []string {
    if block == nil {
        return nil
    }

    parser := variables.NewExpressionParser()

    // Build dependency graph
    graph := make(map[string][]string)
    for _, field := range block.GetAll() {
        deps := parser.ExtractDependencies(field.Eval)
        for _, dep := range deps {
            if dep.Scope == "computed" {
                graph[field.Name] = append(graph[field.Name], dep.Name)
            }
        }
    }

    // Detect cycle using DFS
    visited := make(map[string]bool)
    recStack := make(map[string]bool)

    for node := range graph {
        if !visited[node] {
            if cycle := dfsCycle(node, graph, visited, recStack); cycle != nil {
                return cycle
            }
        }
    }

    return nil
}

// dfsCycle performs DFS to detect cycles
func dfsCycle(node string, graph map[string][]string, visited, recStack map[string]bool) []string {
    visited[node] = true
    recStack[node] = true

    for _, neighbor := range graph[node] {
        if !visited[neighbor] {
            if cycle := dfsCycle(neighbor, graph, visited, recStack); cycle != nil {
                return append(cycle, node)
            }
        } else if recStack[neighbor] {
            return []string{node, neighbor}
        }
    }

    recStack[node] = false
    return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/linter/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/linter/computed.go pkg/flows/linter/computed_test.go
git commit -m "feat(linter): add computed field validation"
```

---

## Chunk 9: CLI Integration and Documentation

### Task 17: Update Example Flow

**Files:**
- Modify: `.gollum/flows/examples/simple-flow.xml`

- [ ] **Step 1: Update example to use new computed section**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<flow name="simple-flow" version="1.0">
    <description>
        Minimal flow demonstrating core concepts with top-level computed section.
    </description>

    <input>
        <string name="target" required="true" />
    </input>

    <output>
        <int name="complexity" />
        <string name="recommendation" />
        <bool name="needs_refactor" />
    </output>

    <context>
        <int name="complexity" />
        <string name="recommendation" />
        <bool name="needs_refactor" />
    </context>

    <!-- NEW: Top-level computed section -->
    <computed>
        <bool name="is_complex" eval="GT(context.complexity, 10)" />
    </computed>

    <agents>
        <agent name="analyzer" model="anthropic/claude-opus-4-6">
            <prompt>Code analyzer - concise and actionable</prompt>
            <temperature>0.0</temperature>
        </agent>
    </agents>

    <states>
        <state name="analyze" initial="true">
            <steps>
                <step type="llm" agent="analyzer">
                    <prompt><![CDATA[
                        Analyze ${target} and provide:
                        1. Complexity score (1-20) - use set_context_field("complexity", <score>)
                        2. Brief recommendation - use set_context_field("recommendation", "<text>")
                        3. needs_refactor flag - use set_context_field("needs_refactor", true/false)

                        Computed value is_complex will be automatically updated.
                    ]]></prompt>
                    <tools>set_context_field</tools>
                    <timeout>60s</timeout>
                </step>
            </steps>

            <transitions>
                <transition to="done" />
            </transitions>
        </state>

        <state name="done" />
    </states>
</flow>
```

- [ ] **Step 2: Commit**

```bash
git add .gollum/flows/examples/simple-flow.xml
git commit -m "docs(example): update simple-flow to use top-level computed section"
```

### Task 18: Update Documentation

**Files:**
- Create: `docs/flows/computed-fields.md`

- [ ] **Step 1: Create computed fields documentation**

```markdown
# Computed Fields

## Overview

Computed fields allow you to derive values from other fields using expressions. They are automatically re-evaluated when their dependencies change.

## Syntax

```xml
<flow name="example" version="1.0">
    <input>
        <int name="x" />
    </input>

    <context>
        <int name="y" />
    </context>

    <computed>
        <bool name="is_large" eval="GT(context.y, 10)" />
        <int name="doubled" eval="MUL(input.x, 2)" />
        <bool name="both" eval="AND(computed.is_large, computed.doubled)" />
    </computed>

    ...
</flow>
```

## Supported Operators

### Comparison
- `GT(a, b)` - greater than
- `LT(a, b)` - less than
- `GTE(a, b)` - greater than or equal
- `LTE(a, b)` - less than or equal
- `EQ(a, b)` - equal
- `NEQ(a, b)` - not equal

### Logical
- `AND(a, b, ...)` - logical AND
- `OR(a, b, ...)` - logical OR
- `NOT(a)` - logical NOT

### Arithmetic
- `ADD(a, b)` - addition
- `SUB(a, b)` - subtraction
- `MUL(a, b)` - multiplication
- `DIV(a, b)` - division

## Field References

- `input.fieldName` - reference input fields
- `context.fieldName` - reference context fields
- `output.fieldName` - reference output fields
- `computed.fieldName` - reference other computed fields

## Reactivity

Computed fields are automatically re-evaluated when:
- Any input field changes (at flow start)
- Any context field is modified by a tool
- Any output field is written
- Any depended-upon computed field changes

## Validation

The linter will detect:
- Undefined field references
- Circular dependencies
- Invalid expression syntax
- Type mismatches

Run: `gollum flow lint myflow.xml`
```

- [ ] **Step 2: Commit**

```bash
git add docs/flows/computed-fields.md
git commit -m "docs: add computed fields documentation"
```

---

## Chunk 10: Migration and Cleanup

### Task 19: Add Migration Helper

**Files:**
- Create: `cmd/gollum/cli/migrate.go`

- [ ] **Step 1: Create migration command**

```go
package cli

import (
    "encoding/xml"
    "fmt"
    "os"

    "github.com/denkhaus/gollum/pkg/flows"
    "github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
    Use:   "migrate <flow-file>",
    Short: "Migrate a flow file to the new schema",
    Args:  cobra.ExactArgs(1),
    RunE func(cmd *cobra.Command, args []string) error {
        filePath := args[0]

        // Read the flow file
        data, err := os.ReadFile(filePath)
        if err != nil {
            return fmt.Errorf("reading file: %w", err)
        }

        // Parse the flow
        var flow flows.Flow
        if err := xml.Unmarshal(data, &flow); err != nil {
            return fmt.Errorf("parsing flow: %w", err)
        }

        // Check if migration is needed
        needsMigration := false
        if flow.Context != nil && len(flow.Context.Computeds) > 0 {
            needsMigration = true
        }

        if !needsMigration {
            fmt.Println("Flow is already using the new schema")
            return nil
        }

        // Migrate: Move computed fields from context to top-level
        if flow.Computed == nil {
            flow.Computed = &flows.ComputedBlock{}
        }

        for _, oldComputed := range flow.Context.Computeds {
            newField := flows.ComputedFieldDef{
                Name: oldComputed.Name,
                Type: oldComputed.Type,
                Eval: oldComputed.When, // Note: When -> Eval migration
            }

            switch oldComputed.Type {
            case "bool":
                flow.Computed.Bools = append(flow.Computed.Bools, newField)
            case "int":
                flow.Computed.Ints = append(flow.Computed.Ints, newField)
            case "string":
                flow.Computed.Strings = append(flow.Computed.Strings, newField)
            case "float":
                flow.Computed.Floats = append(flow.Computed.Floats, newField)
            }
        }

        // Clear old computed fields
        flow.Context.Computeds = nil

        // Write the migrated flow
        newData, err := xml.MarshalIndent(flow, "", "  ")
        if err != nil {
            return fmt.Errorf("marshaling flow: %w", err)
        }

        // Backup original
        backupPath := filePath + ".bak"
        if err := os.WriteFile(backupPath, data, 0644); err != nil {
            return fmt.Errorf("writing backup: %w", err)
        }

        // Write migrated
        if err := os.WriteFile(filePath, newData, 0644); err != nil {
            return fmt.Errorf("writing migrated flow: %w", err)
        }

        fmt.Printf("Migrated %s (backup: %s)\n", filePath, backupPath)
        return nil
    },
}

func init() {
    rootCmd.AddCommand(migrateCmd)
}
```

- [ ] **Step 2: Commit**

```bash
git add cmd/gollum/cli/migrate.go
git commit -m "feat(cli): add flow migration command"
```

### Task 20: Final Integration Tests

**Files:**
- Modify: `pkg/flows/integration_test.go`

- [ ] **Step 1: Add end-to-end test with computed fields**

```go
func TestFlowExecution_WithComputedFields(t *testing.T) {
    xml := `
    <flow name="test-computed" version="1.0">
        <input><int name="x" /></input>
        <context><int name="y" /></context>
        <computed>
            <bool name="is_large" eval="GT(ADD(input.x, context.y), 10)" />
        </computed>
        <states>
            <state name="start" initial="true">
                <steps>
                    <step type="shell" cmd="echo 'setting context'">
                        <output><string name="stdout" assign="ignore" /></output>
                    </step>
                </steps>
                <transitions>
                    <transition to="end" />
                </transition>
            </state>
            <state name="end" />
        </states>
    </flow>
    `

    flow := parseFlow(t, xml)
    exec := newTestExecutor(t, flow)

    exec.SetInput(map[string]string{"x": "5"})

    // Initially, is_large depends on context.y which is 0 (default)
    // 5 + 0 = 5, not > 10, so is_large = false

    // Set context.y
    exec.SetContextField("y", "7")

    // Now: 5 + 7 = 12 > 10, so is_large = true (reactive update)
    isLarge, err := exec.GetComputedField("is_large")
    require.NoError(t, err)
    assert.True(t, isLarge.(bool))
}
```

- [ ] **Step 2: Run full test suite**

Run: `go test ./pkg/flows/... ./pkg/flows/executor/...`
Expected: All tests pass

- [ ] **Step 3: Commit**

```bash
git add pkg/flows/integration_test.go
git commit -m "test(integration): add computed fields end-to-end test"
```

---

## Chunk 11: Final Review and Polish

### Task 21: Review and Finalize

- [ ] **Step 1: Run full test suite**

```bash
go test ./... -v
```

Expected: All tests pass

- [ ] **Step 2: Run linter on example flows**

```bash
go run cmd/gollum/main.go flow lint .gollum/flows/examples/simple-flow.xml
```

Expected: No errors

- [ ] **Step 3: Check code coverage**

```bash
go test ./pkg/flows/variables/... -cover
go test ./pkg/flows/errors/... -cover
```

Expected: High coverage (>80%)

- [ ] **Step 4: Update CLAUDE.md if needed**

```bash
# Add notes about the new flow structure to project docs
```

- [ ] **Step 5: Final commit**

```bash
git add docs/
git commit -m "docs: update project documentation for flow restructure"
```

---

## Summary

This implementation plan restructures the flow spec and executor with:

1. **Separate typed containers** for input, context, output, and computed values
2. **Reactive computed evaluation** that automatically updates when dependencies change
3. **Strong error hierarchy** for precise error handling
4. **Enhanced linting** to catch errors at parse time
5. **TDD approach** throughout with granular test-commit steps

**Total commits: ~21** (one per task for easy rollback and review)

**Key files created:**
- `pkg/flows/errors/errors.go`, `codes.go`
- `pkg/flows/variables/typed.go`, `computed.go`, `expression.go`
- `pkg/flows/executor/execution_context.go`
- `pkg/flows/linter/computed.go`
- `docs/flows/computed-fields.md`
- `cmd/gollum/cli/migrate.go`

**Key files modified:**
- `pkg/flows/types.go` - added ComputedBlock
- `pkg/flows/executor/executor.go` - use ExecutionContext
- `.gollum/flows/examples/simple-flow.xml` - updated example
