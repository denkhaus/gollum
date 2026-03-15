# Typed Values Generic Refactoring Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate code duplication in InputValues, ContextValues, and OutputValues by introducing a generic FieldValues[T] type.

**Architecture:** Create a FieldDefinition interface implemented by existing types (FieldDef, ContextField), then create a generic FieldValues[T] that encapsulates all common storage and accessor logic. Replace the three duplicated types with the generic.

**Tech Stack:** Go 1.23+ (generics), existing flows/errors package, existing flows/types

---

## File Structure

### New Files
- `pkg/flows/variables/field_definition.go` - FieldDefinition interface and adapters
- `pkg/flows/variables/field_values.go` - Generic FieldValues[T] type
- `pkg/flows/variables/computed.go` - Moved from typed.go (unchanged logic)

### Modified Files
- `pkg/flows/variables/typed.go` - Remove InputValues/ContextValues/OutputValues, keep FieldValue
- `pkg/flows/variables/typed_test.go` - Update tests for new types
- `pkg/flows/executor/context.go` - Update to use FieldValues[T]
- `pkg/flows/executor/context_test.go` - Update tests

---

## Chunk 1: FieldDefinition Interface

### Task 1: Create field_definition.go with Interface and Adapters

**Files:**
- Create: `pkg/flows/variables/field_definition.go`
- Test: Create: `pkg/flows/variables/field_definition_test.go`

- [ ] **Step 1: Write tests for FieldDefinition interface**

First, let's define what we need. The interface should abstract the common properties of field definitions.

```go
package variables

import (
    "testing"

    "github.com/denkhaus/gollum/pkg/flows"
)

func TestFieldDefinitionInterface(t *testing.T) {
    // Test that FieldDef implements FieldDefinition
    var fd FieldDefinition = flows.FieldDef{
        Name:  "test_field",
        Type:  "string",
        Default: "default_value",
    }
    if fd.GetName() != "test_field" {
        t.Errorf("expected GetName() to return 'test_field', got '%s'", fd.GetName())
    }
    if fd.GetType() != "string" {
        t.Errorf("expected GetType() to return 'string', got '%s'", fd.GetType())
    }
    if fd.GetDefault() != "default_value" {
        t.Errorf("expected GetDefault() to return 'default_value', got '%s'", fd.GetDefault())
    }
}

func TestContextFieldImplementsFieldDefinition(t *testing.T) {
    var fd FieldDefinition = flows.ContextField{
        Name:  "context_field",
        Type:  "int",
        Default: "42",
    }
    if fd.GetName() != "context_field" {
        t.Errorf("expected GetName() to return 'context_field', got '%s'", fd.GetName())
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./pkg/flows/variables/ -run TestFieldDefinition -v
```
Expected: FAIL - "FieldDefinition not defined" and methods don't exist

- [ ] **Step 3: Implement FieldDefinition interface and adapters**

Create `pkg/flows/variables/field_definition.go`:

```go
package variables

// FieldDefinition defines the interface for field metadata used by FieldValues.
// Both flows.FieldDef and flows.ContextField implement this interface.
type FieldDefinition interface {
    GetName() string
    GetType() string
    GetDefault() string
}
```

- [ ] **Step 4: Add adapter methods for flows.FieldDef**

Add to `pkg/flows/variables/field_definition.go`:

```go
import "github.com/denkhaus/gollum/pkg/flows"

// GetName returns the field name
func (f flows.FieldDef) GetName() string {
    return f.Name
}

// GetType returns the field type
func (f flows.FieldDef) GetType() string {
    return f.Type
}

// GetDefault returns the default value
func (f flows.FieldDef) GetDefault() string {
    return f.Default
}
```

- [ ] **Step 5: Add adapter methods for flows.ContextField**

Add to `pkg/flows/variables/field_definition.go`:

```go
// GetName returns the field name
func (f flows.ContextField) GetName() string {
    return f.Name
}

// GetType returns the field type
func (f flows.ContextField) GetType() string {
    return f.Type
}

// GetDefault returns the default value
func (f flows.ContextField) GetDefault() string {
    return f.Default
}
```

- [ ] **Step 6: Run tests to verify they pass**

```bash
go test ./pkg/flows/variables/ -run TestFieldDefinition -v
```
Expected: PASS

- [ ] **Step 7: Run all tests in variables package**

```bash
go test ./pkg/flows/variables/ -v
```
Expected: PASS (no existing tests should break)

- [ ] **Step 8: Commit**

```bash
git add pkg/flows/variables/field_definition.go pkg/flows/variables/field_definition_test.go
git commit -m "feat: add FieldDefinition interface for generic field values"
```

---

## Chunk 2: Generic FieldValues Type

### Task 2: Create field_values.go with Generic Type

**Files:**
- Create: `pkg/flows/variables/field_values.go`
- Test: Create: `pkg/flows/variables/field_values_test.go`

- [ ] **Step 1: Write failing test for FieldValues constructor**

```go
package variables

import (
    "testing"

    "github.com/denkhaus/gollum/pkg/flows"
)

func TestNewFieldValues(t *testing.T) {
    defs := []flows.FieldDef{
        {Name: "name", Type: "string", Default: ""},
        {Name: "age", Type: "int", Default: "0"},
    }

    fv := NewFieldValues(defs)
    if fv == nil {
        t.Fatal("NewFieldValues returned nil")
    }
    if !fv.Has("name") {
        t.Error("expected Has('name') to return true")
    }
    if !fv.Has("age") {
        t.Error("expected Has('age') to return true")
    }
    if fv.Has("unknown") {
        t.Error("expected Has('unknown') to return false")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./pkg/flows/variables/ -run TestNewFieldValues -v
```
Expected: FAIL - "NewFieldValues not defined"

- [ ] **Step 3: Implement FieldValues struct and constructor**

Create `pkg/flows/variables/field_values.go`:

```go
package variables

import (
    "fmt"
    "strconv"

    "github.com/denkhaus/gollum/pkg/flows"
    "github.com/denkhaus/gollum/pkg/flows/errors"
)

// FieldValues is a generic container for typed field values.
// T must be a type that implements FieldDefinition (e.g., flows.FieldDef or flows.ContextField).
type FieldValues[T FieldDefinition] struct {
    fields map[string]FieldValue
    defs   map[string]T
}

// NewFieldValues creates a new FieldValues from a slice of field definitions.
// Initializes all fields with their default values or unset placeholders.
func NewFieldValues[T FieldDefinition](defs []T) *FieldValues[T] {
    fv := &FieldValues[T]{
        fields: make(map[string]FieldValue),
        defs:   make(map[string]T),
    }

    for _, def := range defs {
        name := def.GetName()
        fv.defs[name] = def

        // Set default value if available
        if def.GetDefault() != "" {
            switch ValueType(def.GetType()) {
            case TypeString:
                fv.fields[name] = NewStringValue(def.GetDefault())
            case TypeInt:
                if i, err := strconv.Atoi(def.GetDefault()); err == nil {
                    fv.fields[name] = NewIntValue(i)
                } else {
                    fv.fields[name] = NewUnsetIntValue()
                }
            case TypeBool:
                if b, err := strconv.ParseBool(def.GetDefault()); err == nil {
                    fv.fields[name] = NewBoolValue(b)
                } else {
                    fv.fields[name] = NewUnsetBoolValue()
                }
            case TypeFloat:
                if f, err := strconv.ParseFloat(def.GetDefault(), 64); err == nil {
                    fv.fields[name] = NewFloatValue(f)
                } else {
                    fv.fields[name] = NewUnsetFloatValue()
                }
            default:
                // Unknown type - store as unset
                fv.fields[name] = NewUnsetStringValue()
            }
        } else {
            // No default - initialize with unset placeholder based on type
            switch ValueType(def.GetType()) {
            case TypeString:
                fv.fields[name] = NewUnsetStringValue()
            case TypeInt:
                fv.fields[name] = NewUnsetIntValue()
            case TypeBool:
                fv.fields[name] = NewUnsetBoolValue()
            case TypeFloat:
                fv.fields[name] = NewUnsetFloatValue()
            default:
                fv.fields[name] = NewUnsetStringValue()
            }
        }
    }

    return fv
}

// Has returns true if the field is defined in the schema
func (fv *FieldValues[T]) Has(name string) bool {
    _, exists := fv.defs[name]
    return exists
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./pkg/flows/variables/ -run TestNewFieldValues -v
```
Expected: PASS

- [ ] **Step 5: Write failing test for SetString/GetString**

```go
func TestFieldValuesSetString(t *testing.T) {
    defs := []flows.FieldDef{
        {Name: "name", Type: "string", Default: ""},
    }
    fv := NewFieldValues(defs)

    // Test setting valid string field
    err := fv.SetString("name", "Alice")
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }

    val, err := fv.GetString("name")
    if err != nil {
        t.Errorf("unexpected error getting value: %v", err)
    }
    if val != "Alice" {
        t.Errorf("expected 'Alice', got '%s'", val)
    }

    // Test setting undefined field
    err = fv.SetString("unknown", "value")
    if err == nil {
        t.Error("expected error for undefined field")
    }
}
```

- [ ] **Step 6: Run test to verify it fails**

```bash
go test ./pkg/flows/variables/ -run TestFieldValuesSetString -v
```
Expected: FAIL - methods not defined

- [ ] **Step 7: Implement SetString and GetString**

Add to `pkg/flows/variables/field_values.go`:

```go
// SetString sets a string value
func (fv *FieldValues[T]) SetString(name string, value string) error {
    if _, ok := fv.defs[name]; !ok {
        return &errors.UnknownFieldError{
            FlowError: errors.FlowError{
                Code:    errors.ErrCodeUnknownField,
                Message: "field not defined",
                Field:   name,
            },
            Scope: "values",
        }
    }

    def, ok := fv.defs[name]
    if !ok || def.GetType() != string(TypeString) {
        return &errors.TypeError{
            FlowError: errors.FlowError{
                Code:    errors.ErrCodeTypeMismatch,
                Message: fmt.Sprintf("field '%s' is not of type string", name),
                Field:   name,
            },
            ExpectedType: "string",
            ActualType:   def.GetType(),
        }
    }

    fv.fields[name] = NewStringValue(value)
    return nil
}

// GetString retrieves a string value
func (fv *FieldValues[T]) GetString(name string) (string, error) {
    fieldVal, ok := fv.fields[name]
    if !ok {
        return "", &errors.UnknownFieldError{
            FlowError: errors.FlowError{
                Code:    errors.ErrCodeUnknownField,
                Message: "field not defined",
                Field:   name,
            },
            Scope: "values",
        }
    }
    if !fieldVal.IsSet() {
        return "", &errors.UnknownFieldError{
            FlowError: errors.FlowError{
                Code:    errors.ErrCodeUnknownField,
                Message: "field not set",
                Field:   name,
            },
            Scope: "values",
        }
    }
    return fieldVal.String()
}
```

- [ ] **Step 8: Run tests to verify they pass**

```bash
go test ./pkg/flows/variables/ -run TestFieldValuesSetString -v
```
Expected: PASS

- [ ] **Step 9: Write tests for remaining type setters/getters**

Add tests for SetInt/GetInt, SetBool/GetBool, SetFloat/GetFloat following the same pattern.

- [ ] **Step 10: Implement remaining type setters/getters**

Add SetInt, GetInt, SetBool, GetBool, SetFloat, GetFloat following the SetString/GetString pattern.

- [ ] **Step 11: Write test for SetFromString**

```go
func TestFieldValuesSetFromString(t *testing.T) {
    defs := []flows.FieldDef{
        {Name: "name", Type: "string", Default: ""},
        {Name: "age", Type: "int", Default: ""},
        {Name: "active", Type: "bool", Default: ""},
        {Name: "score", Type: "float", Default: ""},
    }
    fv := NewFieldValues(defs)

    // Test string to string
    if err := fv.SetFromString("name", "Bob"); err != nil {
        t.Errorf("unexpected error: %v", err)
    }

    // Test string to int
    if err := fv.SetFromString("age", "42"); err != nil {
        t.Errorf("unexpected error: %v", err)
    }
    val, _ := fv.GetInt("age")
    if val != 42 {
        t.Errorf("expected 42, got %d", val)
    }

    // Test string to bool
    if err := fv.SetFromString("active", "true"); err != nil {
        t.Errorf("unexpected error: %v", err)
    }
    active, _ := fv.GetBool("active")
    if !active {
        t.Error("expected true")
    }

    // Test invalid int conversion
    if err := fv.SetFromString("age", "not_a_number"); err == nil {
        t.Error("expected error for invalid int")
    }
}
```

- [ ] **Step 12: Implement SetFromString**

Add to `pkg/flows/variables/field_values.go`:

```go
// SetFromString sets a field value from a string, converting to the target type
func (fv *FieldValues[T]) SetFromString(name string, value string) error {
    def, exists := fv.defs[name]
    if !exists {
        return &errors.UnknownFieldError{
            FlowError: errors.FlowError{
                Code:    errors.ErrCodeUnknownField,
                Message: "field not defined",
                Field:   name,
            },
            Scope: "values",
        }
    }

    switch ValueType(def.GetType()) {
    case TypeString:
        return fv.SetString(name, value)
    case TypeInt:
        i, err := strconv.Atoi(value)
        if err != nil {
            return &errors.TypeError{
                FlowError: errors.FlowError{
                    Code:    errors.ErrCodeTypeMismatch,
                    Message: fmt.Sprintf("cannot convert '%s' to int: %v", value, err),
                    Field:   name,
                },
                ExpectedType: "int",
                ActualType:   "string",
            }
        }
        return fv.SetInt(name, i)
    case TypeBool:
        b, err := strconv.ParseBool(value)
        if err != nil {
            return &errors.TypeError{
                FlowError: errors.FlowError{
                    Code:    errors.ErrCodeTypeMismatch,
                    Message: fmt.Sprintf("cannot convert '%s' to bool: %v", value, err),
                    Field:   name,
                },
                ExpectedType: "bool",
                ActualType:   "string",
            }
        }
        return fv.SetBool(name, b)
    case TypeFloat:
        f, err := strconv.ParseFloat(value, 64)
        if err != nil {
            return &errors.TypeError{
                FlowError: errors.FlowError{
                    Code:    errors.ErrCodeTypeMismatch,
                    Message: fmt.Sprintf("cannot convert '%s' to float: %v", value, err),
                    Field:   name,
                },
                ExpectedType: "float",
                ActualType:   "string",
            }
        }
        return fv.SetFloat(name, f)
    default:
        return fmt.Errorf("unknown field type '%s' for field '%s'", def.GetType(), name)
    }
}
```

- [ ] **Step 13: Write test for GetRaw**

```go
func TestFieldValuesGetRaw(t *testing.T) {
    defs := []flows.FieldDef{
        {Name: "name", Type: "string", Default: ""},
    }
    fv := NewFieldValues(defs)

    fv.SetString("name", "Bob")

    val, ok := fv.GetRaw("name")
    if !ok {
        t.Error("expected ok=true")
    }
    if val != "Bob" {
        t.Errorf("expected 'Bob', got %v", val)
    }

    _, ok = fv.GetRaw("unknown")
    if ok {
        t.Error("expected ok=false for unknown field")
    }
}
```

- [ ] **Step 14: Implement GetRaw**

Add to `pkg/flows/variables/field_values.go`:

```go
// GetRaw returns the raw value and true if the field is set, or nil and false if not set
func (fv *FieldValues[T]) GetRaw(name string) (any, bool) {
    if fieldVal, ok := fv.fields[name]; ok && fieldVal.IsSet() {
        // Extract the underlying value from FieldValue
        switch fieldVal.Type() {
        case TypeString:
            val, _ := fieldVal.String()
            return val, true
        case TypeInt:
            val, _ := fieldVal.Int()
            return val, true
        case TypeBool:
            val, _ := fieldVal.Bool()
            return val, true
        case TypeFloat:
            val, _ := fieldVal.Float()
            return val, true
        }
    }
    return nil, false
}
```

- [ ] **Step 15: Run all tests**

```bash
go test ./pkg/flows/variables/ -v
```
Expected: All PASS

- [ ] **Step 16: Commit**

```bash
git add pkg/flows/variables/field_values.go pkg/flows/variables/field_values_test.go
git commit -m "feat: add generic FieldValues type with type-safe accessors"
```

---

## Chunk 3: Move ComputedValues

### Task 3: Move ComputedValues to separate file

**Files:**
- Create: `pkg/flows/variables/computed.go`
- Modify: `pkg/flows/variables/typed.go` (remove ComputedValues)
- Test: Create: `pkg/flows/variables/computed_test.go`

- [ ] **Step 1: Extract ComputedValues from typed.go**

Find the ComputedValues struct and all its methods in `pkg/flows/variables/typed.go` and copy them to a new file `pkg/flows/variables/computed.go`.

Include:
- ComputedValues struct
- NewComputedValues constructor
- All ComputedValues methods (SetString, GetString, SetInt, GetInt, etc.)
- ComputedEvaluator (if related to computed fields)

- [ ] **Step 2: Remove ComputedValues from typed.go**

Delete the ComputedValues-related code from `typed.go`.

- [ ] **Step 3: Copy/create tests for ComputedValues**

If there are existing tests for ComputedValues in typed_test.go, move them to computed_test.go.

- [ ] **Step 4: Run tests**

```bash
go test ./pkg/flows/variables/ -v
```
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/variables/computed.go pkg/flows/variables/computed_test.go pkg/flows/variables/typed.go
git commit -m "refactor: move ComputedValues to separate file"
```

---

## Chunk 4: Update typed.go

### Task 4: Clean up typed.go - remove old types

**Files:**
- Modify: `pkg/flows/variables/typed.go`

- [ ] **Step 1: Remove InputValues type and methods**

Delete from `typed.go`:
- `type InputValues struct`
- `func NewInputValues`
- All InputValues methods (SetString, GetString, SetInt, GetInt, etc.)

- [ ] **Step 2: Remove ContextValues type and methods**

Delete from `typed.go`:
- `type ContextValues struct`
- `func NewContextValues`
- All ContextValues methods

- [ ] **Step 3: Remove OutputValues type and methods**

Delete from `typed.go`:
- `type OutputValues struct`
- `func NewOutputValues`
- All OutputValues methods

- [ ] **Step 4: Verify typed.go only contains FieldValue**

The file should now only contain:
- ValueType enum and constants
- FieldValue struct and constructors
- FieldValue methods

- [ ] **Step 5: Run tests - expect failures**

```bash
go test ./pkg/flows/variables/ -v
```
Expected: Some tests fail (they will be fixed in next chunk)

- [ ] **Step 6: No commit yet** (waiting for context.go updates)

---

## Chunk 5: Update context.go

### Task 5: Update Context to use FieldValues generic

**Files:**
- Modify: `pkg/flows/executor/context.go`
- Test: Modify: `pkg/flows/executor/context_test.go`

- [ ] **Step 1: Update Context struct to use FieldValues generic**

In `pkg/flows/executor/context.go`, update the Context struct:

```go
type Context struct {
    inputBlock    *flows.InputBlock
    inputVals     *variables.FieldValues[flows.FieldDef]
    outputBlock   *flows.OutputBlock
    outputValues  *variables.FieldValues[flows.FieldDef]
    contextBlock  *flows.ContextBlock
    contextValues *variables.FieldValues[flows.ContextField]
    computedBlock *flows.ComputedBlock
    computedVals  *variables.ComputedValues
    eval          *Evaluator
    lastError     *ErrorContext
}
```

- [ ] **Step 2: Update NewContext constructor**

Update the NewContext function to use the generic constructors:

```go
func NewContext(
    inputBlock *flows.InputBlock,
    outputBlock *flows.OutputBlock,
    contextBlock *flows.ContextBlock,
    inputVals map[string]string,
) *Context {
    ctx := &Context{
        inputBlock:    inputBlock,
        inputVals:     variables.NewFieldValues(inputBlock.GetAllFields()),
        outputBlock:   outputBlock,
        outputValues:  variables.NewFieldValues(outputBlock.GetAllFields()),
        contextBlock:  contextBlock,
        contextValues: variables.NewFieldValues(contextBlock.GetAllFields()),
        computedBlock: nil,
        computedVals:  variables.NewComputedValues(nil),
        eval:          NewEvaluator(),
    }
    // ... rest of initialization
}
```

Note: You'll need to add a GetAllFields() method to ContextBlock if it doesn't exist.

- [ ] **Step 3: Update SetComputedBlock**

```go
func (c *Context) SetComputedBlock(block *flows.ComputedBlock) {
    c.computedBlock = block
    // ... rest unchanged
}
```

- [ ] **Step 4: Add GetAllFields method to ContextBlock (if needed)**

In `pkg/flows/types.go`, add:

```go
// GetAllFields returns all context fields as a slice
func (cb *ContextBlock) GetAllFields() []ContextField {
    fields := make([]ContextField, 0)
    fields = append(fields, cb.Strings...)
    fields = append(fields, cb.Ints...)
    fields = append(fields, cb.Bools...)
    fields = append(fields, cb.Floats...)
    return fields
}
```

- [ ] **Step 5: Update GetInput method**

The GetInput method uses inputVals which is now FieldValues[flows.FieldDef]. Update accordingly.

- [ ] **Step 6: Update GetContextField method**

The GetContextField method uses contextValues which is now FieldValues[flows.ContextField]. Update accordingly.

- [ ] **Step 7: Update SetContextField method**

The SetContextField method uses contextValues. Update accordingly.

- [ ] **Step 8: Update SetOutputField method**

The SetOutputField method uses outputValues. Update accordingly.

- [ ] **Step 9: Update GetOutputField method**

The GetOutputField method uses outputValues. Update accordingly.

- [ ] **Step 10: Update buildScope method**

The buildScope method uses inputVals, outputValues, contextValues. Update GetRaw() calls accordingly.

- [ ] **Step 11: Update EvaluateComputed method**

The EvaluateComputed method needs updating. It references inputVals, contextValues, outputValues.

Note: ComputedEvaluator may need updating to work with FieldValues[T].

- [ ] **Step 12: Update GetComputedEvaluator method**

Similar to EvaluateComputed.

- [ ] **Step 13: Run tests**

```bash
go test ./pkg/flows/executor/ -run TestContext -v
```
Expected: PASS after all updates

- [ ] **Step 14: Run all executor tests**

```bash
go test ./pkg/flows/executor/ -v
```
Expected: All PASS

- [ ] **Step 15: Commit**

```bash
git add pkg/flows/executor/context.go pkg/flows/executor/context_test.go pkg/flows/types.go
git commit -m "refactor: update Context to use generic FieldValues"
```

---

## Chunk 6: Optional Type Aliases for Backward Compatibility

### Task 6: Add type aliases (optional, for backward compatibility)

**Files:**
- Modify: `pkg/flows/variables/typed.go`

- [ ] **Step 1: Add type aliases**

Add to `pkg/flows/variables/typed.go`:

```go
// Type aliases for backward compatibility
// Deprecated: Use FieldValues[T] directly instead

// InputValues is an alias for FieldValues[flows.FieldDef]
// Deprecated: Use FieldValues[flows.FieldDef] directly
type InputValues = FieldValues[flows.FieldDef]

// ContextValues is an alias for FieldValues[flows.ContextField]
// Deprecated: Use FieldValues[flows.ContextField] directly
type ContextValues = FieldValues[flows.ContextField]

// OutputValues is an alias for FieldValues[flows.FieldDef]
// Deprecated: Use FieldValues[flows.FieldDef] directly
type OutputValues = FieldValues[flows.FieldDef]

// NewInputValues creates a new InputValues (backward compatibility wrapper)
// Deprecated: Use variables.NewFieldValues(defs) directly
func NewInputValues(block *flows.InputBlock) *InputValues {
    return NewFieldValues(block.GetAllFields())
}

// NewContextValues creates a new ContextValues (backward compatibility wrapper)
// Deprecated: Use variables.NewFieldValues(defs) directly
func NewContextValues(block *flows.ContextBlock) *ContextValues {
    return NewFieldValues(block.GetAllFields())
}

// NewOutputValues creates a new OutputValues (backward compatibility wrapper)
// Deprecated: Use variables.NewFieldValues(defs) directly
func NewOutputValues(block *flows.OutputBlock) *OutputValues {
    return NewFieldValues(block.GetAllFields())
}
```

- [ ] **Step 2: Run all tests**

```bash
go test ./pkg/flows/... -v
```
Expected: All PASS

- [ ] **Step 3: Commit**

```bash
git add pkg/flows/variables/typed.go
git commit -m "feat: add backward compatibility type aliases for InputValues/ContextValues/OutputValues"
```

---

## Chunk 7: Final Verification and Documentation

### Task 7: Final verification and documentation

**Files:**
- Test: All

- [ ] **Step 1: Run all tests in the project**

```bash
go test ./... -v
```
Expected: All PASS

- [ ] **Step 2: Build the project**

```bash
just build
```
Expected: Success, no errors

- [ ] **Step 3: Run the default flow**

```bash
just run
```
Expected: Success, "Set context field" log visible

- [ ] **Step 4: Update documentation**

Update any documentation that references InputValues, ContextValues, or OutputValues to use FieldValues[T] instead.

- [ ] **Step 5: Check for any remaining references**

```bash
grep -r "InputValues\|ContextValues\|OutputValues" --include="*.go" pkg/
```

If any found (excluding type aliases), update them.

- [ ] **Step 6: Final commit**

```bash
git add docs/
git commit -m "docs: update documentation for generic FieldValues refactoring"
```

---

## Completion Criteria

- [ ] All tests pass
- [ ] Code builds successfully
- [ ] Default flow runs successfully
- [ ] No remaining references to old types (except aliases)
- [ ] Documentation updated

## Rollback Plan

If issues arise:
1. Use `git revert` to revert commits in reverse order
2. Or `git reset --hard <commit-before-refactoring>`
3. Delete new files: field_definition.go, field_values.go, computed.go
