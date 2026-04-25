# Environment Variable Interpolation for Flows - Design

**Status:** Approved
**Date:** 2026-04-25
**Author:** Claude (Brainstorming Session)

## Overview

Extend the flow template system with a new `env` scope that enables environment variable references via `${env.VAR_NAME}` syntax. The implementation follows existing patterns for other scopes (`input`, `context`, `output`, `computed`, `sys`).

## Core Principles

1. **Read-only access** - No assignments to `env` space
2. **Hard-failure on missing vars** - Flow execution aborts if referenced env var is missing
3. **Early validation + caching** - Hybrid approach: validate at startup, cache for performance
4. **Minimal invasive changes** - Extend existing files only

## Architectural Changes

### 1. New Error Type

**File:** `pkg/flows/errors/errors.go`

```go
// EnvVarNotFoundError indicates an environment variable referenced in flow was not found
type EnvVarNotFoundError struct {
    FlowError
    VarName string
}
```

### 2. New Scope Enum

**File:** `pkg/flows/types.go`

- Add new enum value: `FlowVariableScopeEnv = "env"`
- Extend `Validate()` map to include `FlowVariableScopeEnv: true`

### 3. Template Interpolation

**File:** `pkg/flows/executor/template.go`

- Extend regex: `\$\{(input|context|output|computed|sys|env)\.([^}]+)\}`
- Add new case for `env` scope in `substituteTemplate()`
- Delegate to `ctx.GetEnvField(field)`

### 4. Context Implementation

**File:** `pkg/flows/executor/context.go`

**Interface Extension:**
```go
type ExecutionContext interface {
    // ... existing methods
    GetEnvField(field string) (any, error)
    ValidateEnvVars() error
}
```

**Implementation:**
```go
type contextImpl struct {
    // ... existing fields
    envCache     map[string]string
    envValidated bool
}

func (c *contextImpl) ValidateEnvVars() error {
    // Scan all prompt templates for ${env.*} references
    // Read each env var once with os.Getenv()
    // Cache the value or return EnvVarNotFoundError if missing/empty
}

func (c *contextImpl) GetEnvField(field string) (any, error) {
    if !c.envValidated {
        return nil, errors.New("environment variables not validated, call ValidateEnvVars() first")
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

### 5. Executor Integration

**File:** `pkg/flows/executor/executor.go`

- Call `ctx.ValidateEnvVars()` in `SetInput()` after initialization
- Or call in `Run()` before first state execution

## Data Flow & Lifecycle

```
Flow Start
    ↓
SetInput() - Input values + defaults are set
    ↓
ValidateEnvVars() - Environment variables are validated and cached
    ↓
For each ${env.VAR} reference:
    1. os.Getenv(VAR) is called
    2. If empty → throw EnvVarNotFoundError
    3. Cache in envCache
    ↓
State Execution
    ↓
Template Interpolation (${env.VAR})
    ↓
substituteTemplate() → ctx.GetEnvField(field)
    ↓
Return cached value from envCache
```

## Error Handling

### Scenario 1: Environment Variable Not Found
```
Error: ENV_VAR_NOT_FOUND: environment variable 'API_KEY' not found
  at flow: startup, state: analyze
  referenced in default value: ${env.API_KEY}
```

### Scenario 2: Variable Exists but is Empty
```
Error: ENV_VAR_EMPTY: environment variable 'API_KEY' is empty
  at flow: startup, state: analyze
  Environment variables must not be empty strings
```

### Scenario 3: Syntax Error (Invalid Scope)
```
Error: INVALID_SCOPE: unknown variable scope 'enV'
  (valid: input, context, output, computed, sys, env)
  Did you mean 'env'?
```

## Testing Strategy

### Unit Tests (`context_test.go`)

- `TestGetEnvField_ExistingVariable()` - Successful read
- `TestGetEnvField_MissingVariable()` - Error when var missing
- `TestGetEnvField_EmptyVariable()` - Error when var empty
- `TestValidateEnvVars_AllPresent()` - Validation succeeds
- `TestValidateEnvVars_MissingRequired()` - Early error
- `TestEnvInterpolationInTemplate()` - Template substitution works

### Integration Test (`env_integration_test.go`)

- Test flow with `${env.TEST_VAR}` reference
- Set/unset environment variables in test
- Verify error message on missing variable

### Linter Extension

**File:** `pkg/flows/linter/syntax_checker.go`

- Validate that `env` scope is only read-only
- Warn about `${env.*}` in `assignTo` (not allowed)
- Detect `${env.*}` syntax in templates

## XSD Schema Updates

**File:** `pkg/flows/linter/schema/flow.xsd`

No changes required - the `default` attribute already accepts string values, and `${env.VAR}` syntax is validated at runtime, not schema level.

## Example Flows

**File:** `.gollum/flows/examples/env-vars-example.xml`

```xml
<?xml version="1.0" encoding="UTF-8"?>
<flow name="env-vars-example" version="1.0">
    <description>Demonstrates environment variable usage</description>

    <input>
        <string name="api_key" default="${env.API_KEY}" />
        <string name="workspace" default="${env.PWD}" />
    </input>

    <output>
        <string name="result" />
    </output>

    <agents>
        <agent name="worker" model="anthropic/claude-opus-4-6">
            <prompt>Worker agent</prompt>
        </agent>
    </agents>

    <states>
        <state name="process" initial="true">
            <steps>
                <step type="llm" agent="worker">
                    <prompt><![CDATA[
Workspace: ${input.workspace}
API Key configured: ${input.api_key}
Provide confirmation.
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

## Implementation Files Summary

1. `pkg/flows/errors/errors.go` - Add `EnvVarNotFoundError`
2. `pkg/flows/types.go` - Add `FlowVariableScopeEnv` enum + Validate
3. `pkg/flows/executor/template.go` - Extend regex + case block
4. `pkg/flows/executor/context.go` - Interface + implementation
5. `pkg/flows/executor/executor.go` - ValidateEnvVars() calls
6. `pkg/flows/executor/context_test.go` - Unit tests
7. `pkg/flows/executor/env_integration_test.go` - Integration test
8. `pkg/flows/linter/syntax_checker.go` - Linter validation
9. `.gollum/flows/examples/env-vars-example.xml` - Example flow

**No new packages** - everything stays within existing structure.
