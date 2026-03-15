# Typed Values Generic Refactoring Design

**Date:** 2026-03-15
**Status:** Approved
**Author:** Claude (brainstorming session)

## Problem Statement

The current `InputValues`, `ContextValues`, and `OutputValues` types in `pkg/flows/variables/typed.go` contain significant code duplication:

- All three have identical `fields map[string]FieldValue` storage
- All three have identical `defs` maps (different types, same pattern)
- All three have identical setter/getter methods (SetString, GetString, SetInt, GetInt, etc.)
- Type conversion logic (`SetFromString`) is duplicated

This duplication makes the code harder to maintain and error-prone when adding new features.

## Solution

Introduce a generic `FieldValues[T]` type that encapsulates the common behavior, eliminating code duplication while maintaining type safety.

## Architecture

### New Files

#### `pkg/flows/variables/field_definition.go`

Defines the `FieldDefinition` interface that abstracts the common properties of field definitions:

```go
type FieldDefinition interface {
    GetName() string
    GetType() string
    GetDefault() string
}
```

Adapter implementations for existing types:
- `func (f flows.FieldDef) GetName() string { return f.Name }`
- `func (f flows.ContextField) GetName() string { return f.Name }`
(Same for GetType, GetDefault)

#### `pkg/flows/variables/field_values.go`

The core generic type:

```go
type FieldValues[T FieldDefinition] struct {
    fields map[string]FieldValue
    defs   map[string]T
}

func NewFieldValues[T FieldDefinition](defs []T) *FieldValues[T]
```

Methods (type-safe, validated against schema):
- `SetString(name, value string) error`
- `GetString(name) (string, error)`
- `SetInt(name string, value int) error`
- `GetInt(name) (int, error)`
- `SetBool(name string, value bool) error`
- `GetBool(name) (bool, error)`
- `SetFloat(name string, value float64) error`
- `GetFloat(name) (float64, error)`
- `SetFromString(name, value string) error` - Converts string to target type
- `GetRaw(name) (any, bool)` - Returns raw value for scope building
- `Has(name) bool` - Checks if field is defined in schema

#### `pkg/flows/variables/computed.go`

Moved from `typed.go` to its own file. Unchanged except for location.
ComputedValues remains a separate type due to its unique dirty-tracking requirements.

### Modified Files

#### `pkg/flows/variables/typed.go`

**Removed:**
- `InputValues` type and methods
- `ContextValues` type and methods
- `OutputValues` type and methods

**Retained:**
- `FieldValue` type and constructors
- `ValueType` enum and constants
- All FieldValue methods (Type, IsSet, String, Int, Bool, Float, SetRaw)

#### Files using InputValues/ContextValues/OutputValues

Update imports and type references:
- `pkg/flows/executor/context.go` - Replace `*variables.InputValues` with `*variables.FieldValues[flows.FieldDef]`
- Similar updates for ContextValues and OutputValues

## Data Flow

```
Flow Definition (XML)
    ↓
Parsed Blocks (InputBlock, ContextBlock, OutputBlock)
    ↓
FieldDefinitions (FieldDef or ContextField)
    ↓
FieldValues[T] - Generic storage with type-safe access
    ↓
Executor accesses values via typed getters/setters
```

## Error Handling

All setter methods return custom errors from `pkg/flows/errors`:
- `UnknownFieldError` - Field not defined in schema
- `TypeError` - Type mismatch in conversion
- `ImmutableFieldError` - (Future) Attempt to modify read-only field

## Backward Compatibility

**Breaking Changes:**
- Type names change: `InputValues` → `FieldValues[flows.FieldDef]`
- This affects external code if these types are exported

**Mitigation:**
- Type aliases can be added if needed for compatibility:
  ```go
  type InputValues = FieldValues[flows.FieldDef]
  type ContextValues = FieldValues[flows.ContextField]
  type OutputValues = FieldValues[flows.FieldDef]
  ```

## Testing Strategy

1. Keep all existing tests passing
2. Add tests for new FieldDefinition interface adapters
3. Add tests for FieldValues generic with both FieldDef and ContextField
4. Ensure SetFromString handles all type conversions correctly

## Future Enhancements

Once the generic is in place:
- Easy to add new field types (e.g., `Duration`, `Timestamp`)
- Can add validation hooks to FieldValues
- Can add change notification support
- Can add metadata support (field descriptions, constraints)

## Implementation Checklist

- [ ] Create `field_definition.go` with interface and adapters
- [ ] Create `field_values.go` with generic type
- [ ] Move ComputedValues to `computed.go`
- [ ] Remove InputValues/ContextValues/OutputValues from `typed.go`
- [ ] Update `context.go` to use new generic types
- [ ] Add type aliases for backward compatibility (if needed)
- [ ] Run all tests
- [ ] Update any other files using the old types
