# Refactoring: Methods and Shared Helpers Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Consolidate duplicate helper functions into shared package and refactor AST package to use them.

**Architecture:** Create new `conversions.go` in shared package with type-checking and character classification helpers. Refactor AST evaluator and lexer to use shared functions instead of local duplicates.

**Tech Stack:** Go 1.21+, existing test infrastructure

---

## File Structure

| File | Action | Purpose |
|------|--------|---------|
| `pkg/shared/conversions.go` | CREATE | Type-checking helpers (IsInt, IsActualIntType, IsWholeNumber) |
| `pkg/shared/conversions_test.go` | CREATE | Tests for new helpers |
| `pkg/shared/char_helpers.go` | CREATE | Character classification (IsIdentStart, IsIdentPart, IsDigit) |
| `pkg/shared/char_helpers_test.go` | CREATE | Tests for char helpers |
| `pkg/flows/ast/evaluator.go` | MODIFY | Use shared.ConvertToFloat, shared.ConvertToBool, shared.IsInt |
| `pkg/flows/ast/lexer.go` | MODIFY | Use shared.IsIdentStart, shared.IsIdentPart, shared.IsDigit |

---

## Chunk 1: Create Shared Type-Checking Helpers

### Task 1: Create conversions.go with type helpers

**Files:**
- Create: `pkg/shared/conversions.go`
- Create: `pkg/shared/conversions_test.go`

> **Note:** `ConvertToBool` and `ConvertToFloat` already exist in `pkg/shared/flow_result.go`. This task adds only the type-checking helpers (IsInt, IsActualIntType, IsWholeNumber).

- [ ] **Step 1: Write the failing tests**

```go
// pkg/shared/conversions_test.go
package shared

import "testing"

func TestIsInt(t *testing.T) {
	tests := []struct {
		input    any
		expected bool
	}{
		{int(5), true},
		{int64(5), true},
		{int32(5), true},
		{uint(5), true},
		{float64(5.0), true},  // whole number
		{float64(5.5), false}, // not whole
		{float32(5.0), true},
		{"5", false},
		{nil, false},
	}
	for _, tt := range tests {
		if got := IsInt(tt.input); got != tt.expected {
			t.Errorf("IsInt(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestIsActualIntType(t *testing.T) {
	tests := []struct {
		input    any
		expected bool
	}{
		{int(5), true},
		{int64(5), true},
		{uint(5), true},
		{float64(5.0), false}, // float, even if whole
		{"5", false},
	}
	for _, tt := range tests {
		if got := IsActualIntType(tt.input); got != tt.expected {
			t.Errorf("IsActualIntType(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestIsWholeNumber(t *testing.T) {
	tests := []struct {
		input    float64
		expected bool
	}{
		{5.0, true},
		{5.5, false},
		{0.0, true},
		{-3.0, true},
		{-3.14, false},
	}
	for _, tt := range tests {
		if got := IsWholeNumber(tt.input); got != tt.expected {
			t.Errorf("IsWholeNumber(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/denkhaus/dev/gomodules/gollum && go test ./pkg/shared/... -run "TestIsInt|TestIsActualIntType|TestIsWholeNumber" -v`
Expected: FAIL - function not defined

- [ ] **Step 3: Write the implementation**

```go
// pkg/shared/conversions.go
package shared

// IsWholeNumber checks if a float64 is a whole number
func IsWholeNumber(f float64) bool {
	return f == float64(int64(f))
}

// IsInt checks if a value is an integer type or a float representing a whole number
func IsInt(v any) bool {
	switch val := v.(type) {
	case int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8:
		return true
	case float64:
		return IsWholeNumber(val)
	case float32:
		return IsWholeNumber(float64(val))
	default:
		return false
	}
}

// IsActualIntType checks if the value's actual type is an integer (not a float that happens to be whole)
func IsActualIntType(v any) bool {
	switch v.(type) {
	case int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8:
		return true
	default:
		return false
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/denkhaus/dev/gomodules/gollum && go test ./pkg/shared/... -run "TestIsInt|TestIsActualIntType|TestIsWholeNumber" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/shared/conversions.go pkg/shared/conversions_test.go
git commit -m "feat(shared): add IsInt, IsActualIntType, IsWholeNumber helpers"
```

---

## Chunk 2: Create Shared Character Classification Helpers

### Task 2: Create char_helpers.go with lexer utilities

**Files:**
- Create: `pkg/shared/char_helpers.go`
- Create: `pkg/shared/char_helpers_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// pkg/shared/char_helpers_test.go
package shared

import "testing"

func TestIsIdentStart(t *testing.T) {
	tests := []struct {
		ch       byte
		expected bool
	}{
		{'a', true},
		{'Z', true},
		{'_', true},
		{'0', false},
		{'-', false},
		{' ', false},
	}
	for _, tt := range tests {
		if got := IsIdentStart(tt.ch); got != tt.expected {
			t.Errorf("IsIdentStart(%q) = %v, want %v", tt.ch, got, tt.expected)
		}
	}
}

func TestIsIdentPart(t *testing.T) {
	tests := []struct {
		ch       byte
		expected bool
	}{
		{'a', true},
		{'Z', true},
		{'_', true},
		{'0', true},
		{'9', true},
		{'-', false},
		{' ', false},
	}
	for _, tt := range tests {
		if got := IsIdentPart(tt.ch); got != tt.expected {
			t.Errorf("IsIdentPart(%q) = %v, want %v", tt.ch, got, tt.expected)
		}
	}
}

func TestIsDigit(t *testing.T) {
	tests := []struct {
		ch       byte
		expected bool
	}{
		{'0', true},
		{'5', true},
		{'9', true},
		{'a', false},
		{'-', false},
	}
	for _, tt := range tests {
		if got := IsDigit(tt.ch); got != tt.expected {
			t.Errorf("IsDigit(%q) = %v, want %v", tt.ch, got, tt.expected)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/denkhaus/dev/gomodules/gollum && go test ./pkg/shared/... -run "TestIsIdentStart|TestIsIdentPart|TestIsDigit" -v`
Expected: FAIL - function not defined

- [ ] **Step 3: Write the implementation**

```go
// pkg/shared/char_helpers.go
package shared

// IsIdentStart returns true if ch can start an identifier
func IsIdentStart(ch byte) bool {
	return ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

// IsIdentPart returns true if ch can be part of an identifier
func IsIdentPart(ch byte) bool {
	return IsIdentStart(ch) || IsDigit(ch)
}

// IsDigit returns true if ch is a digit
func IsDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/denkhaus/dev/gomodules/gollum && go test ./pkg/shared/... -run "TestIsIdentStart|TestIsIdentPart|TestIsDigit" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/shared/char_helpers.go pkg/shared/char_helpers_test.go
git commit -m "feat(shared): add IsIdentStart, IsIdentPart, IsDigit helpers"
```

---

## Chunk 3: Refactor AST Evaluator to Use Shared Helpers

### Task 3: Update evaluator.go to use shared functions

**Files:**
- Modify: `pkg/flows/ast/evaluator.go`

- [ ] **Step 1: Run existing tests to establish baseline**

Run: `cd /home/denkhaus/dev/gomodules/gollum && go test ./pkg/flows/ast/... -v`
Expected: All tests PASS

- [ ] **Step 2: Replace toFloat64 with shared.ConvertToFloat**

In `pkg/flows/ast/evaluator.go`, replace:

```go
// DELETE these functions (lines 181-297):
func toFloat64(v any) (float64, error) { ... }       // line 181
func toBool(v any) (bool, error) { ... }            // line 198
func isWholeNumber(f float64) bool { ... }          // line 271
func isInt(v any) bool { ... }                      // line 276
func isActualIntType(v any) bool { ... }            // line 290
```

Replace calls in `compare()`, `logicalAnd()`, `logicalOr()`, `logicalNot()`, `arithmetic()`, `divide()`:

```go
// Before:
a, err := toFloat64(args[0])

// After:
a, err := shared.ConvertToFloat(args[0])
```

```go
// Before:
b, err := toBool(arg)

// After:
b, err := shared.ConvertToBool(arg)
```

```go
// Before:
if isWholeNumber(result) && isInt(args[0]) && isInt(args[1])

// After:
if shared.IsWholeNumber(result) && shared.IsInt(args[0]) && shared.IsInt(args[1])
```

```go
// Before:
if isActualIntType(args[0]) && isActualIntType(args[1])

// After:
if shared.IsActualIntType(args[0]) && shared.IsActualIntType(args[1])
```

- [ ] **Step 3: Run tests to verify no regression**

Run: `cd /home/denkhaus/dev/gomodules/gollum && go test ./pkg/flows/ast/... -v`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
git add pkg/flows/ast/evaluator.go
git commit -m "refactor(ast): use shared type conversion helpers in evaluator"
```

---

## Chunk 4: Refactor AST Lexer to Use Shared Helpers

### Task 4: Update lexer.go to use shared functions

**Files:**
- Modify: `pkg/flows/ast/lexer.go`

- [ ] **Step 1: Run existing tests to establish baseline**

Run: `cd /home/denkhaus/dev/gomodules/gollum && go test ./pkg/flows/ast/... -run TestLex -v`
Expected: All tests PASS

- [ ] **Step 2: Add shared import**

In `pkg/flows/ast/lexer.go`, add the shared import:

```go
// Before:
import (
	"unicode"
)

// After:
import (
	"unicode"

	"github.com/denkhaus/gollum/pkg/shared"
)
```

- [ ] **Step 3: Replace local helpers with shared functions**

In `pkg/flows/ast/lexer.go`, delete the local functions and update calls:

```go
// DELETE these functions (lines 125-137):
func isIdentStart(ch byte) bool { ... }  // line 125
func isIdentPart(ch byte) bool { ... }   // line 130
func isDigit(ch byte) bool { ... }       // line 135
```

Update all calls from `isIdentStart(ch)` to `shared.IsIdentStart(ch)`, etc.

```go
// Before (line 45):
if isIdentStart(ch) {

// After:
if shared.IsIdentStart(ch) {
```

```go
// Before (line 48):
for pos < len(input) && isIdentPart(input[pos]) {

// After:
for pos < len(input) && shared.IsIdentPart(input[pos]) {
```

```go
// Before (lines 82, 87, 92):
if isDigit(ch) ...
for pos < len(input) && isDigit(input[pos]) ...

// After:
if shared.IsDigit(ch) ...
for pos < len(input) && shared.IsDigit(input[pos]) ...
```

Also delete the `unicode` import since it's no longer needed (line 4).

- [ ] **Step 4: Run tests to verify no regression**

Run: `cd /home/denkhaus/dev/gomodules/gollum && go test ./pkg/flows/ast/... -v`
Expected: All tests PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/ast/lexer.go
git commit -m "refactor(ast): use shared character classification helpers in lexer"
```

---

## Chunk 5: Final Verification

### Task 5: Full test suite verification

- [ ] **Step 1: Run full test suite**

Run: `cd /home/denkhaus/dev/gomodules/gollum && go test ./... -count=1`
Expected: All tests PASS

- [ ] **Step 2: Run build to verify no compilation errors**

Run: `cd /home/denkhaus/dev/gomodules/gollum && just build`
Expected: Build succeeds

- [ ] **Step 3: Final commit (if any cleanup needed)**

```bash
git status
# If clean, no action needed
```

---

## Summary

| Task | Description | Files Changed |
|------|-------------|---------------|
| 1 | Create type-checking helpers | +2 files |
| 2 | Create char classification helpers | +2 files |
| 3 | Refactor evaluator to use shared | 1 file |
| 4 | Refactor lexer to use shared | 1 file |
| 5 | Final verification | - |

**Total:** 4 new files, 2 modified files
