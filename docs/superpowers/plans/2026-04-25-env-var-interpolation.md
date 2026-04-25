# Environment Variable Interpolation Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `${env.VAR_NAME}` environment variable interpolation to flow templates with read-only access and hard-failure on missing vars.

**Architecture:** Extend existing flow template system with new `env` scope. Environment variables are validated once at flow startup and cached for performance. Uses existing custom error patterns in `pkg/flows/errors`.

**Tech Stack:** Go 1.23+, existing flow executor, custom errors package, TDD with table-driven tests

---

## File Structure

**New Error Types:**
- `pkg/flows/errors/errors.go` - Add `EnvVarNotFoundError` and `EnvVarEmptyError` structs

**Scope Enum:**
- `pkg/flows/types.go` - Add `FlowVariableScopeEnv` constant

**Template Engine:**
- `pkg/flows/executor/template.go` - Extend regex and add env case

**Context Implementation:**
- `pkg/flows/executor/context.go` - Add `GetEnvField()` and `ValidateEnvVars()`

**Executor Integration:**
- `pkg/flows/executor/executor.go` - Call validation at startup

**Tests:**
- `pkg/flows/executor/context_test.go` - Unit tests for env var handling
- `pkg/flows/executor/env_integration_test.go` - Integration test

**Linter:**
- `pkg/flows/linter/syntax_checker.go` - Validate env scope usage

**Example Flow:**
- `.gollum/flows/examples/env-vars-example.xml` - Demonstrates the feature

---

## Chunk 1: Error Types and Scope Enum

### Task 1: Add EnvVarNotFoundError and EnvVarEmptyError

**Files:**
- Modify: `pkg/flows/errors/errors.go`
- Test: `pkg/flows/errors/errors_test.go`

- [ ] **Step 1: Write the failing test for missing var**

```go
// Test in pkg/flows/errors/errors_test.go
func TestEnvVarNotFoundError(t *testing.T) {
    err := &EnvVarNotFoundError{
        VarName: "API_KEY",
    }
    err.Code = "ENV_VAR_NOT_FOUND"
    err.Message = "environment variable 'API_KEY' not found"

    expected := "ENV_VAR_NOT_FOUND: environment variable 'API_KEY' not found"
    if err.Error() != expected {
        t.Errorf("expected %q, got %q", expected, err.Error())
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/errors -v -run TestEnvVarNotFoundError`
Expected: FAIL with "undefined: EnvVarNotFoundError"

- [ ] **Step 3: Write minimal implementation for missing var**

```go
// Add to pkg/flows/errors/errors.go

// EnvVarNotFoundError indicates an environment variable referenced in flow was not found
type EnvVarNotFoundError struct {
    FlowError
    VarName string
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/errors -v -run TestEnvVarNotFoundError`
Expected: PASS

- [ ] **Step 5: Write test for empty var**

```go
// Test in pkg/flows/errors/errors_test.go
func TestEnvVarEmptyError(t *testing.T) {
    err := &EnvVarEmptyError{
        VarName: "API_KEY",
    }
    err.Code = "ENV_VAR_EMPTY"
    err.Message = "environment variable 'API_KEY' is empty"
    
    expected := "ENV_VAR_EMPTY: environment variable 'API_KEY' is empty"
    if err.Error() != expected {
        t.Errorf("expected %q, got %q", expected, err.Error())
    }
}
```

- [ ] **Step 6: Implement EnvVarEmptyError**

```go
// Add to pkg/flows/errors/errors.go

// EnvVarEmptyError indicates an environment variable is set but empty
type EnvVarEmptyError struct {
    FlowError
    VarName string
}
```

- [ ] **Step 7: Run all error tests**

Run: `go test ./pkg/flows/errors -v`
Expected: All PASS

- [ ] **Step 8: Commit**

```bash
git add pkg/flows/errors/errors.go pkg/flows/errors/errors_test.go
git commit -m "feat(flows/errors): add EnvVarNotFoundError and EnvVarEmptyError types"
```

### Task 2: Add FlowVariableScopeEnv Enum

**Files:**
- Modify: `pkg/flows/types.go`
- Test: `pkg/flows/types_test.go`

- [ ] **Step 1: Write the failing test**

```go
// Test in pkg/flows/types_test.go
func TestFlowVariableScopeEnv(t *testing.T) {
    scope := FlowVariableScopeEnv
    if scope != "env" {
        t.Errorf("expected 'env', got %q", scope)
    }

    err := scope.Validate()
    if err != nil {
        t.Errorf("expected nil, got %v", err)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows -v -run TestFlowVariableScopeEnv`
Expected: FAIL with "undefined: FlowVariableScopeEnv"

- [ ] **Step 3: Write minimal implementation**

```go
// Add to pkg/flows/types.go after line 66 (after FlowVariableScopeSys)

// FlowVariableScopeEnv represents environment variable scope
const FlowVariableScopeEnv FlowVariableScope = "env"
```

- [ ] **Step 4: Update Validate() method**

Find the `Validate()` method and make two changes:

1. Add `FlowVariableScopeEnv` to the valid map:

```go
valid := map[FlowVariableScope]bool{
    FlowVariableScopeInput:    true,
    FlowVariableScopeContext:  true,
    FlowVariableScopeOutput:   true,
    FlowVariableScopeComputed: true,
    FlowVariableScopeSys:      true,
    FlowVariableScopeEnv:      true, // ADD THIS LINE
}
```

2. Update the error message to include "env":

```go
// OLD (approximate):
return fmt.Errorf("invalid flow variable scope: %s (expected one of: input, context, output, computed, sys)", p)

// NEW:
return fmt.Errorf("invalid flow variable scope: %s (expected one of: input, context, output, computed, sys, env)", p)
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./pkg/flows -v -run TestFlowVariableScopeEnv`
Expected: PASS

- [ ] **Step 6: Run all flow tests**

Run: `go test ./pkg/flows -v`
Expected: All PASS

- [ ] **Step 7: Commit**

```bash
git add pkg/flows/types.go pkg/flows/types_test.go
git commit -m "feat(flows): add FlowVariableScopeEnv enum value"
```

---

## Chunk 2: Context Interface and Template Extension

### Task 3: Add GetEnvField to ExecutionContext Interface

**Note:** Following proper TDD, we add the interface method before using it in template code.

**Files:**
- Modify: `pkg/flows/executor/context.go`

- [ ] **Step 1: Update ExecutionContext interface**

```go
// In pkg/flows/executor/context.go, find the ExecutionContext interface (around line 83)
// Add this method to the interface:

GetEnvField(field string) (any, error)
```

- [ ] **Step 2: Add stub to contextImpl struct**

```go
// Add to pkg/flows/executor/context.go, after GetSysField method (around line 480)

// GetEnvField returns a cached environment variable value
func (c *contextImpl) GetEnvField(field string) (any, error) {
    // Stub implementation - will be completed in Chunk 3
    return nil, fmt.Errorf("not yet implemented")
}
```

- [ ] **Step 3: Compile check**

Run: `go build ./pkg/flows/executor`
Expected: SUCCESS

- [ ] **Step 4: Commit**

```bash
git add pkg/flows/executor/context.go
git commit -m "feat(flows/executor): add GetEnvField to ExecutionContext interface (stub)"
```

### Task 4: Extend Template Regex and Add Env Case

**Files:**
- Modify: `pkg/flows/executor/template.go`
- Test: `pkg/flows/executor/template_test.go`

- [ ] **Step 1: Write the failing test**

```go
// Test in pkg/flows/executor/template_test.go
func TestSubstituteTemplate_EnvScope_Success(t *testing.T) {
    // Mock context with complete interface implementation
    mockCtx := &mockEnvContext{}
    
    result := substituteTemplate(mockCtx, "Workspace: ${env.PWD}")
    expected := "Workspace: /tmp/test"
    
    if result != expected {
        t.Errorf("expected %q, got %q", expected, result)
    }
}

func TestSubstituteTemplate_EnvScope_NotFound(t *testing.T) {
    mockCtx := &mockEnvContext{missingVars: true}
    
    // When env var is missing, template should return original string
    result := substituteTemplate(mockCtx, "${env.MISSING}")
    expected := "${env.MISSING}"
    
    if result != expected {
        t.Errorf("expected %q, got %q", expected, result)
    }
}

func TestSubstituteTemplate_MixedScopes(t *testing.T) {
    mockCtx := &mockEnvContext{user: "testuser"}
    
    result := substituteTemplate(mockCtx, "${env.USER} in ${input.workspace}")
    expected := "testuser in ${input.workspace}"
    
    if result != expected {
        t.Errorf("expected %q, got %q", expected, result)
    }
}

// Complete mock implementing all 11 interface methods
type mockEnvContext struct {
    user         string
    missingVars  bool
}

func (m *mockEnvContext) GetInput(string) any { return nil }
func (m *mockEnvContext) GetContextField(string) (any, error) { return nil, nil }
func (m *mockEnvContext) SetContextField(string, any) error { return nil }
func (m *mockEnvContext) GetOutputField(string) (any, error) { return nil, nil }
func (m *mockEnvContext) SetOutputField(string, any) error { return nil }
func (m *mockEnvContext) GetComputedField(string) (any, error) { return nil, nil }
func (m *mockEnvContext) EvaluateComputed() error { return nil }
func (m *mockEnvContext) GetSysField(string) any { return nil }
func (m *mockEnvContext) SetError(*ErrorContext) {}
func (m *mockEnvContext) SubstituteTemplate(string) string { return "" }
func (m *mockEnvContext) GetEnvField(field string) (any, error) {
    if m.missingVars {
        return nil, fmt.Errorf("not found")
    }
    if field == "PWD" {
        return "/tmp/test", nil
    }
    if field == "USER" && m.user != "" {
        return m.user, nil
    }
    return nil, fmt.Errorf("not found")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/executor -v -run TestSubstituteTemplate_EnvScope`
Expected: FAIL - template doesn't interpolate env

- [ ] **Step 3: Update regex**

```go
// In pkg/flows/executor/template.go, line ~11
// OLD: var subRegex = regexp.MustCompile(`\$\{(input|context|output|computed|sys)\.([^}]+)\}`)
// NEW:
var subRegex = regexp.MustCompile(`\$\{(input|context|output|computed|sys|env)\.([^}]+)\}`)
```

- [ ] **Step 4: Add env case to substituteTemplate (following existing pattern)**

```go
// In pkg/flows/executor/template.go, in substituteTemplate function
// Add this case AFTER the `case flows.FlowVariableScopeSys:` block (after line 51):

case flows.FlowVariableScopeEnv:
    // env scope provides environment variables (read-only)
    value, err = ctx.GetEnvField(field)
    if err != nil {
        // Follow existing pattern: set value to nil on error
        // The nil check below will return the original template string
        value = nil
    }
```

Note: This follows the exact same pattern as the `context` and `output` cases (lines 33-42).

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./pkg/flows/executor -v -run TestSubstituteTemplate_EnvScope`
Expected: PASS

- [ ] **Step 6: Run all executor tests**

Run: `go test ./pkg/flows/executor -v`
Expected: All PASS

- [ ] **Step 7: Commit**

```bash
git add pkg/flows/executor/template.go pkg/flows/executor/template_test.go
git commit -m "feat(flows/executor): extend template interpolation for env scope"
```

---

## Chunk 3: Context Implementation

### Task 5: Add ValidateEnvVars to Interface and Struct Fields

**Files:**
- Modify: `pkg/flows/executor/context.go`

- [ ] **Step 1: Add ValidateEnvVars to ExecutionContext interface**

```go
// In pkg/flows/executor/context.go, find the ExecutionContext interface (around line 83)
// Add this method after GetEnvField:

ValidateEnvVars() error
```

- [ ] **Step 2: Update contextImpl struct**

```go
// In pkg/flows/executor/context.go, find the contextImpl struct (around line 91)
// Add these fields:

type contextImpl struct {
    // ... existing fields
    envCache     map[string]string
    envValidated bool
}
```

- [ ] **Step 3: Initialize envCache in constructor**

Find the `newContext()` function and initialize the map:

```go
envCache: make(map[string]string),
envValidated: false,
```

- [ ] **Step 4: Commit**

```bash
git add pkg/flows/executor/context.go
git commit -m "feat(flows/executor): add env fields to context struct"
```

### Task 6: Implement GetEnvField (Replacing Stub)

**Files:**
- Modify: `pkg/flows/executor/context.go`
- Test: `pkg/flows/executor/context_test.go`

- [ ] **Step 1: Write the failing test**

```go
// Test in pkg/flows/executor/context_test.go
func TestGetEnvField_BeforeValidation(t *testing.T) {
    flow := &flows.Flow{
        Input: &flows.InputBlock{},
    }
    ctx := newContext(flow, nil, nil, nil)

    _, err := ctx.GetEnvField("PWD")
    if err == nil {
        t.Error("expected error when not validated")
    }
}

func TestGetEnvField_AfterValidation(t *testing.T) {
    t.Setenv("TEST_VAR", "test_value")

    flow := &flows.Flow{
        Input: &flows.InputBlock{},
    }
    ctx := newContext(flow, nil, nil, nil)

    // Mock validation to bypass template scanning
    ctx.envCache = map[string]string{"TEST_VAR": "test_value"}
    ctx.envValidated = true

    val, err := ctx.GetEnvField("TEST_VAR")
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }
    if val != "test_value" {
        t.Errorf("expected 'test_value', got %v", val)
    }
}

func TestGetEnvField_MissingVar(t *testing.T) {
    flow := &flows.Flow{
        Input: &flows.InputBlock{},
    }
    ctx := newContext(flow, nil, nil, nil)

    ctx.envCache = map[string]string{}
    ctx.envValidated = true

    _, err := ctx.GetEnvField("MISSING")
    if err == nil {
        t.Error("expected error for missing var")
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/flows/executor -v -run TestGetEnvField`
Expected: FAIL - stub returns "not yet implemented"

- [ ] **Step 3: Implement GetEnvField (replacing stub)**

```go
// In pkg/flows/executor/context.go, replace the stub implementation:

// GetEnvField returns a cached environment variable value
func (c *contextImpl) GetEnvField(field string) (any, error) {
    if !c.envValidated {
        return nil, fmt.Errorf("environment variables not validated, call ValidateEnvVars() first")
    }

    val, ok := c.envCache[field]
    if !ok {
        return nil, &errors.EnvVarNotFoundError{
            VarName: field,
        }
    }
    return val, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/flows/executor -v -run TestGetEnvField`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/flows/executor/context.go pkg/flows/executor/context_test.go
git commit -m "feat(flows/executor): implement GetEnvField method"
```

### Task 7: Implement ValidateEnvVars with Input Default Scanning

**Files:**
- Modify: `pkg/flows/executor/context.go`
- Test: `pkg/flows/executor/context_test.go`

- [ ] **Step 1: Write the failing test**

```go
// Test in pkg/flows/executor/context_test.go
func TestValidateEnvVars_NoEnvRefs(t *testing.T) {
    flow := &flows.Flow{
        Input: &flows.InputBlock{},
    }
    ctx := newContext(flow, nil, nil, nil)

    err := ctx.ValidateEnvVars()
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }
    if !ctx.envValidated {
        t.Error("expected validated flag to be true")
    }
}

func TestValidateEnvVars_InputDefault(t *testing.T) {
    t.Setenv("WORKSPACE", "/tmp/test")

    flow := &flows.Flow{
        Input: &flows.InputBlock{
            Fields: []flows.FieldDef{
                {Name: "workspace", Type: flows.TypeString, Default: "${env.WORKSPACE}"},
            },
        },
    }
    ctx := newContext(flow, nil, nil, nil)

    err := ctx.ValidateEnvVars()
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }

    val, ok := ctx.envCache["WORKSPACE"]
    if !ok {
        t.Error("expected WORKSPACE in cache")
    }
    if val != "/tmp/test" {
        t.Errorf("expected '/tmp/test', got %q", val)
    }
}

func TestValidateEnvVars_StatePrompt(t *testing.T) {
    t.Setenv("API_KEY", "secret123")

    flow := &flows.Flow{
        Input: &flows.InputBlock{},
        States: &flows.StatesBlock{
            States: []*flows.State{
                {
                    Name: "test",
                    Steps: []*flows.Step{
                        {
                            Prompt: "Key: ${env.API_KEY}",
                        },
                    },
                },
            },
        },
    }
    ctx := newContext(flow, nil, nil, nil)

    err := ctx.ValidateEnvVars()
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }

    val, ok := ctx.envCache["API_KEY"]
    if !ok {
        t.Error("expected API_KEY in cache")
    }
    if val != "secret123" {
        t.Errorf("expected 'secret123', got %q", val)
    }
}

func TestValidateEnvVars_MissingVar(t *testing.T) {
    // Ensure the env var is NOT set
    os.Unsetenv("NONEXISTENT_VAR")

    flow := &flows.Flow{
        Input: &flows.InputBlock{},
        States: &flows.StatesBlock{
            States: []*flows.State{
                {
                    Name: "test",
                    Steps: []*flows.Step{
                        {
                            Prompt: "${env.NONEXISTENT_VAR}",
                        },
                    },
                },
            },
        },
    }
    ctx := newContext(flow, nil, nil, nil)

    err := ctx.ValidateEnvVars()
    if err == nil {
        t.Error("expected error for missing env var")
    }

    envErr, ok := err.(*errors.EnvVarNotFoundError)
    if !ok {
        t.Fatalf("expected EnvVarNotFoundError, got %T", err)
    }
    if envErr.VarName != "NONEXISTENT_VAR" {
        t.Errorf("expected 'NONEXISTENT_VAR', got %q", envErr.VarName)
    }
}

func TestValidateEnvVars_EmptyVar(t *testing.T) {
    t.Setenv("EMPTY_VAR", "")

    flow := &flows.Flow{
        Input: &flows.InputBlock{},
        States: &flows.StatesBlock{
            States: []*flows.State{
                {
                    Name: "test",
                    Steps: []*flows.Step{
                        {
                            Prompt: "${env.EMPTY_VAR}",
                        },
                    },
                },
            },
        },
    }
    ctx := newContext(flow, nil, nil, nil)

    err := ctx.ValidateEnvVars()
    if err == nil {
        t.Error("expected error for empty env var")
    }

    emptyErr, ok := err.(*errors.EnvVarEmptyError)
    if !ok {
        t.Fatalf("expected EnvVarEmptyError, got %T", err)
    }
}

func TestValidateEnvVars_MissingVsEmpty(t *testing.T) {
    // Test 1: Missing var (not set)
    os.Unsetenv("MISSING_VAR")
    
    flow1 := &flows.Flow{
        Input: &flows.InputBlock{},
        States: &flows.StatesBlock{
            States: []*flows.State{
                {
                    Name: "test",
                    Steps: []*flows.Step{
                        {Prompt: "${env.MISSING_VAR}"},
                    },
                },
            },
        },
    }
    ctx1 := newContext(flow1, nil, nil, nil)
    
    err1 := ctx1.ValidateEnvVars()
    if _, ok := err1.(*errors.EnvVarNotFoundError); !ok {
        t.Errorf("missing var should return EnvVarNotFoundError, got %T", err1)
    }
    
    // Test 2: Empty var (set but empty)
    t.Setenv("EMPTY_VAR", "")
    
    flow2 := &flows.Flow{
        Input: &flows.InputBlock{},
        States: &flows.StatesBlock{
            States: []*flows.State{
                {
                    Name: "test",
                    Steps: []*flows.Step{
                        {Prompt: "${env.EMPTY_VAR}"},
                    },
                },
            },
        },
    }
    ctx2 := newContext(flow2, nil, nil, nil)
    
    err2 := ctx2.ValidateEnvVars()
    if _, ok := err2.(*errors.EnvVarEmptyError); !ok {
        t.Errorf("empty var should return EnvVarEmptyError, got %T", err2)
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/flows/executor -v -run TestValidateEnvVars`
Expected: FAIL - "ValidateEnvVars not implemented"

- [ ] **Step 3: Implement extractEnvVarRefs helper**

```go
// Add to pkg/flows/executor/context.go (after the GetSysField method)

// extractEnvVarRefs finds all ${env.VAR} references in prompt templates and input defaults
func (c *contextImpl) extractEnvVarRefs() map[string]bool {
    refs := make(map[string]bool)

    // Scan input field defaults for env var references
    if c.flow.Input != nil {
        for _, field := range c.flow.Input.Fields {
            if field.Default != "" {
                matches := subRegex.FindAllStringSubmatch(field.Default, -1)
                for _, match := range matches {
                    if len(match) >= 3 {
                        scope := flows.FlowVariableScope(match[1])
                        if scope == flows.FlowVariableScopeEnv {
                            refs[match[2]] = true
                        }
                    }
                }
            }
        }
    }

    // Scan all steps for env var references
    if c.flow.States != nil {
        for _, state := range c.flow.States.States {
            for _, step := range state.Steps {
                // Check prompt template
                if step.Prompt != "" {
                    matches := subRegex.FindAllStringSubmatch(step.Prompt, -1)
                    for _, match := range matches {
                        if len(match) >= 3 {
                            scope := flows.FlowVariableScope(match[1])
                            if scope == flows.FlowVariableScopeEnv {
                                refs[match[2]] = true
                            }
                        }
                    }
                }
            }
        }
    }

    return refs
}
```

- [ ] **Step 4: Implement ValidateEnvVars**

```go
// Add to pkg/flows/executor/context.go

// ValidateEnvVars validates and caches all environment variables referenced in the flow
func (c *contextImpl) ValidateEnvVars() error {
    // Find all env var references
    refs := c.extractEnvVarRefs()

    // No env vars referenced? Skip validation
    if len(refs) == 0 {
        c.envValidated = true
        return nil
    }

    // Validate each referenced env var
    for varName := range refs {
        // Use LookupEnv to distinguish missing from empty
        value, exists := os.LookupEnv(varName)
        if !exists {
            // Variable not set at all
            return &errors.EnvVarNotFoundError{
                VarName: varName,
            }
        }
        if value == "" {
            // Variable set but empty
            return &errors.EnvVarEmptyError{
                VarName: varName,
            }
        }
        c.envCache[varName] = value
    }

    c.envValidated = true
    return nil
}
```

- [ ] **Step 5: Add required imports**

Make sure these are imported in context.go:
```go
import (
    "os"  // Add this for os.LookupEnv
    // ... existing imports
)
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./pkg/flows/executor -v -run TestValidateEnvVars`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add pkg/flows/executor/context.go pkg/flows/executor/context_test.go
git commit -m "feat(flows/executor): implement ValidateEnvVars with missing/empty distinction"
```

---

## Chunk 4: Executor Integration

### Task 8: Call ValidateEnvVars in Executor

**Files:**
- Modify: `pkg/flows/executor/executor.go`
- Test: `pkg/flows/executor/executor_test.go`

- [ ] **Step 1: Find where to add validation**

Locate the `SetInput()` method in executor.go (around line 282). We need to call `ValidateEnvVars()` after the context is created.

- [ ] **Step 2: Add validation call**

In `SetInput()`, after line 322 (after `ctx := newContext(...)`), add:

```go
// Validate environment variables referenced in flow
if err := ctx.ValidateEnvVars(); err != nil {
    return fmt.Errorf("environment variable validation failed: %w", err)
}
```

- [ ] **Step 3: Run existing tests**

Run: `go test ./pkg/flows/executor -v`
Expected: All PASS (existing flows without env vars should work)

- [ ] **Step 4: Write integration test with context import**

```go
// Test in pkg/flows/executor/env_integration_test.go (new file)

package executor

import (
    "context"
    "os"
    "testing"

    "github.com/denkhaus/gollum/pkg/flows"
)

func TestFlowWithEnvVars_Success(t *testing.T) {
    t.Setenv("WORKSPACE_PATH", "/tmp/test")
    t.Setenv("API_KEY", "secret_key")

    flow := &flows.Flow{
        Name:    "test",
        Version: "1.0",
        Input: &flows.InputBlock{
            Fields: []flows.FieldDef{
                {Name: "workspace", Type: flows.TypeString, Default: "${env.WORKSPACE_PATH}"},
            },
        },
        States: &flows.StatesBlock{
            States: []*flows.State{
                {
                    Name: "done",
                },
            },
        },
    }

    exec := NewFlowExecutor()
    inputs := map[string]any{}

    result, err := exec.Execute(context.Background(), flow, inputs)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result.Status != flows.FlowStatusSuccess {
        t.Errorf("expected success, got %v", result.Status)
    }
}

func TestFlowWithEnvVars_MissingVar(t *testing.T) {
    // Ensure NOT set
    os.Unsetenv("MISSING_VAR")

    flow := &flows.Flow{
        Name:    "test",
        Version: "1.0",
        States: &flows.StatesBlock{
            States: []*flows.State{
                {
                    Name: "test",
                    Steps: []*flows.Step{
                        {Prompt: "${env.MISSING_VAR}"},
                    },
                },
            },
        },
    }

    exec := NewFlowExecutor()
    _, err := exec.Execute(context.Background(), flow, nil)
    if err == nil {
        t.Error("expected error for missing env var")
    }
}

func TestFlowWithEnvVars_EmptyVar(t *testing.T) {
    t.Setenv("EMPTY_VAR", "")

    flow := &flows.Flow{
        Name:    "test",
        Version: "1.0",
        States: &flows.StatesBlock{
            States: []*flows.State{
                {
                    Name: "test",
                    Steps: []*flows.Step{
                        {Prompt: "${env.EMPTY_VAR}"},
                    },
                },
            },
        },
    }

    exec := NewFlowExecutor()
    _, err := exec.Execute(context.Background(), flow, nil)
    if err == nil {
        t.Error("expected error for empty env var")
    }
}

func TestFlowWithEnvVars_InputDefault(t *testing.T) {
    t.Setenv("DEFAULT_WORKSPACE", "/home/user/project")

    flow := &flows.Flow{
        Name:    "test",
        Version: "1.0",
        Input: &flows.InputBlock{
            Fields: []flows.FieldDef{
                {Name: "workspace", Type: flows.TypeString, Default: "${env.DEFAULT_WORKSPACE}"},
            },
        },
        States: &flows.StatesBlock{
            States: []*flows.State{
                {
                    Name: "done",
                },
            },
        },
    }

    exec := NewFlowExecutor()
    inputs := map[string]any{}

    result, err := exec.Execute(context.Background(), flow, inputs)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result.Status != flows.FlowStatusSuccess {
        t.Errorf("expected success, got %v", result.Status)
    }
}
```

- [ ] **Step 5: Run integration tests**

Run: `go test ./pkg/flows/executor -v -run TestFlowWithEnvVars`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add pkg/flows/executor/executor.go pkg/flows/executor/env_integration_test.go
git commit -m "feat(flows/executor): integrate env var validation into executor"
```

---

## Chunk 5: Linter Updates

### Task 9: Add Linter Validation for Env Scope

**Files:**
- Modify: `pkg/flows/linter/syntax_checker.go`
- Test: `pkg/flows/linter/syntax_checker_test.go`

- [ ] **Step 1: Update template regex in linter**

Find the regex in syntax_checker.go and add `env` to the pattern.

- [ ] **Step 2: Add validation rule**

Add a linter rule to ensure `${env.*}` is NOT used in `assignTo` (env is read-only):

```go
// In the appropriate linter phase
func (s *SyntaxChecker) checkEnvInAssignTo(flow *flows.Flow, result *flows.LinterResult) {
    if flow.Output != nil {
        for _, field := range flow.Output.GetAllFields() {
            if field.AssignFrom != "" {
                if strings.HasPrefix(field.AssignFrom, "env.") {
                    result.AddError(flows.LinterError{
                        Message: fmt.Sprintf("assignTo cannot use env scope (read-only): %s", field.AssignFrom),
                    })
                }
            }
        }
    }
}
```

- [ ] **Step 3: Run linter tests**

Run: `go test ./pkg/flows/linter -v`
Expected: All PASS

- [ ] **Step 4: Commit**

```bash
git add pkg/flows/linter/syntax_checker.go pkg/flows/linter/syntax_checker_test.go
git commit -m "feat(flows/linter): add env scope validation"
```

---

## Chunk 6: Example Flow with Current Model

### Task 10: Create Example Flow

**Files:**
- Create: `.gollum/flows/examples/env-vars-example.xml`

- [ ] **Step 1: Create example flow with current model**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<flow name="env-vars-example" version="1.0">
    <description>Demonstrates environment variable usage</description>

    <input>
        <string name="workspace" default="${env.PWD}" />
    </input>

    <output>
        <string name="result" />
    </output>

    <agents>
        <agent name="worker" model="anthropic/claude-sonnet-4-20250514">
            <prompt>Worker agent</prompt>
            <temperature>0.0</temperature>
        </agent>
    </agents>

    <states>
        <state name="process" initial="true">
            <steps>
                <step type="llm" agent="worker">
                    <prompt><![CDATA[
Workspace path is: ${input.workspace}

Provide a brief confirmation that includes this path.
                    ]]></prompt>
                    <timeout>10s</timeout>
                    <result assignTo="output.result"/>
                </step>
            </steps>
            <transitions>
                <transition to="done"/>
            </transitions>
        </state>
        <state name="done"/>
    </states>
</flow>
```

- [ ] **Step 2: Test the example flow**

Run: `gollum flow run .gollum/flows/examples/env-vars-example.xml`
Expected: Flow executes successfully, output contains workspace path

- [ ] **Step 3: Test with missing env var**

Run: `PWD="" gollum flow run .gollum/flows/examples/env-vars-example.xml`
Expected: Error about missing/empty PWD variable

- [ ] **Step 4: Test input default with env var**

Run: `MY_VAR="test value" gollum flow run .gollum/flows/examples/env-vars-example.xml`
Expected: Input field default resolves from environment variable

- [ ] **Step 5: Commit**

```bash
git add .gollum/flows/examples/env-vars-example.xml
git commit -m "docs(examples): add env vars example flow with current model"
```

---

## Summary

**Files Modified:**
1. `pkg/flows/errors/errors.go` - Added error types for missing and empty vars
2. `pkg/flows/types.go` - Added env scope enum
3. `pkg/flows/executor/template.go` - Extended template interpolation
4. `pkg/flows/executor/context.go` - Added GetEnvField and ValidateEnvVars with input default scanning
5. `pkg/flows/executor/executor.go` - Integrated validation
6. `pkg/flows/executor/context_test.go` - Unit tests including missing vs empty distinction
7. `pkg/flows/executor/env_integration_test.go` - Integration tests with context import (new)
8. `pkg/flows/linter/syntax_checker.go` - Linter validation
9. `.gollum/flows/examples/env-vars-example.xml` - Example flow with current model (new)

**Key Improvements from Original Plan:**
- Removed duplicate Task 5 (GetEnvField was already added as stub in Chunk 2)
- Added input field default scanning in extractEnvVarRefs()
- Used os.LookupEnv() to distinguish missing vs empty vars
- Added context import to integration tests
- Added test for input default with env var interpolation
- Updated example flow to use current model version

**Test Coverage:**
- Unit tests for GetEnvField with validation state checks
- Unit tests for ValidateEnvVars including missing vs empty distinction
- Unit tests for input default env var resolution
- Integration tests for executor with env vars
- Linter tests for env scope validation
- Manual testing with example flow

**Next Steps:**
1. Run all tests: `go test ./...`
2. Test with real flows
3. Update documentation
