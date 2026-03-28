# Refactoring: Methods and Shared Helpers

**Date:** 2026-03-28
**Status:** Approved

## Overview

Refactor the codebase to improve code quality by:
1. Converting standalone functions that operate on structs into methods
2. Consolidating duplicate helper functions into the `pkg/shared` package

## Section 1: Method Transformations

### Functions to convert to methods:

| Current Function | Location | New Method On | Rationale |
|-----------------|----------|---------------|-----------|
| `getAllFieldsSafe(block interface{})` | executor/context.go:162 | `flows.ContextBlock` | Operates on block data |
| `getAllContextFieldsSafe(block *flows.ContextBlock)` | executor/context.go:173 | `flows.ContextBlock` | Directly tied to ContextBlock |
| `validateInputType(value, typeName string)` | executor/executor.go:1025 | `flowExecutorImpl` | Used in validation flow |
| `substituteTemplate(ctx ExecutionContext, tmpl string)` | executor/template.go:16 | `ExecutionContext` interface | First param is the receiver |

### Example transformation:

```go
// Before (standalone)
func getAllContextFieldsSafe(block *flows.ContextBlock) []flows.ContextField

// After (method on ContextBlock)
func (b *ContextBlock) GetAllFields() []ContextField
```

## Section 2: Shared Helper Consolidation

### Current duplicates to consolidate:

| AST Evaluator (private) | Shared Package (exported) | Action |
|------------------------|---------------------------|--------|
| `toFloat64(v any)` | `ConvertToFloat(v any)` | Remove AST version, use shared |
| `toBool(v any)` | `ConvertToBool(v any)` | Remove AST version, use shared |
| — | `ConvertToInt(v any)` | Add `IsInt(v any) bool` helper |
| `isInt(v any)`, `isActualIntType(v any)` | — | Move to shared as `IsInt()` |

### New helpers to add to `pkg/shared/conversions.go`:

```go
// Type checking helpers
func IsInt(v any) bool
func IsFloat(v any) bool
func IsNumeric(v any) bool

// Character classification (for lexers/parsers)
func IsIdentStart(ch byte) bool
func IsIdentPart(ch byte) bool
func IsDigit(ch byte) bool
```

### File changes:

| File | Change |
|------|--------|
| `pkg/shared/conversions.go` | **NEW** - consolidate type conversions + add helpers |
| `pkg/shared/flow_result.go` | Keep existing conversion functions |
| `pkg/flows/ast/evaluator.go` | Import shared conversions, remove local helpers |
| `pkg/flows/ast/lexer.go` | Import shared helpers, remove local `isX()` functions |

## Section 3: Implementation Approach

### Execution order (minimizes breaking changes):

1. **Create `pkg/shared/conversions.go`**
   - Add new type-checking helpers (IsInt, IsFloat, IsNumeric)
   - Add character classification (IsIdentStart, IsIdentPart, IsDigit)

2. **Update `pkg/shared/flow_result.go`**
   - Keep existing ConvertToX functions (already exported, used elsewhere)
   - No breaking changes

3. **Refactor `pkg/flows/ast/evaluator.go`**
   - Replace `toFloat64` → `shared.ConvertToFloat`
   - Replace `toBool` → `shared.ConvertToBool` (adjust error handling)
   - Replace `isInt`/`isActualIntType` → `shared.IsInt`

4. **Refactor `pkg/flows/ast/lexer.go`**
   - Replace `isIdentStart` → `shared.IsIdentStart`
   - Replace `isIdentPart` → `shared.IsIdentPart`
   - Replace `isDigit` → `shared.IsDigit`

5. **Add methods to flows package types**
   - Add `GetAllFields()` to `ContextBlock`
   - Update `executor/context.go` to use new methods

6. **Run tests after each step**

### Scope Boundaries

- **In scope:** Conversions, type checks, lexer helpers, ContextBlock methods
- **Out of scope:** Major API changes, interface modifications, test file refactors
