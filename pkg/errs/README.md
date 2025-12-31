# Package `errs`

Unified error handling package for the Gollum project. Provides structured error types with context, wrapping, and categorization.

## Overview

This package provides a consistent error handling framework that:
- Categorizes errors by type (validation, not found, permission, internal, conflict)
- Supports error wrapping with cause tracking
- Adds structured context to errors
- Integrates with Go's standard error interfaces

## Error Types

```go
const (
    TypeValidation  // Input validation failures
    TypeNotFound    // Resource not found
    TypePermission  // Authorization/permission failures
    TypeInternal    // Internal system errors
    TypeConflict    // State conflicts (e.g., concurrent modification)
)
```

## Usage

### Basic Error Creation

```go
import "github.com/denkhaus/gollum/pkg/errs"

// Simple error
err := errs.Validation("email is required")

// Formatted error
err := errs.NotFoundf("agent %s not found", agentID)

// With context
err := errs.Permission("access denied").
    WithContext("agent_id", agentID).
    WithContext("resource", "agents")
```

### Error Wrapping

```go
// Wrap an underlying error
if err := db.Query(agentID); err != nil {
    return errs.Wrap(err, errs.TypeInternal, "failed to fetch agent")
}

// Wrap with formatted message
return errs.Wrapf(err, errs.TypeValidation, "invalid %s field", fieldName)
```

### Error Checking

```go
import "errors"

// Check if error is a specific type
if errs.IsType(err, errs.TypeValidation) {
    // Handle validation error
}

// Convert to our Error type
if e := errs.AsError(err); e != nil {
    fmt.Printf("Error type: %s\n", e.GetType())
    fmt.Printf("Context: %v\n", e.GetContext())
}

// Standard errors.Is and errors.As work with our errors
if errors.Is(err, someError) {
    // Handle specific error
}
```

### Error Context

```go
err := errs.Validation("invalid input").
    WithContext("field", "email").
    WithContext("value", input).
    WithContext("rule", "must contain @")

// Access context
if e := errs.AsError(err); e != nil {
    ctx := e.GetContext()
    field := ctx["field"]
}
```

## Examples

### Registry Error

```go
func (r *registry) GetAgent(id uuid.UUID) (Agent, error) {
    agent, exists := r.agents[id]
    if !exists {
        return nil, errs.NotFoundf("agent %s not found", id).
            WithContext("agent_id", id)
    }
    return agent, nil
}
```

### Tool Error with Context

```go
func (t *Tool) Run(ctx context.Context, args map[string]any) error {
    path, ok := args["path"].(string)
    if !ok || path == "" {
        return errs.Validation("path is required").
            WithContext("provided_type", fmt.Sprintf("%T", args["path"]))
    }
    // ...
}
```

### Wrapped Error

```go
func (s *Service) CreateAgent(config AgentConfig) error {
    if err := s.db.Save(config); err != nil {
        return errs.Wrap(err, errs.TypeInternal, "failed to persist agent config").
            WithContext("agent_id", config.ID)
    }
    return nil
}
```

## Best Practices

1. **Use appropriate error types** - Match the error type to the failure category
2. **Add helpful context** - Include relevant IDs, field names, values
3. **Wrap with care** - Only wrap when adding meaningful context
4. **Consistent messages** - Use clear, actionable error messages
5. **Don't double-wrap** - Check before wrapping errors

## Testing

```go
func TestSomething(t *testing.T) {
    err := MyFunction()
    
    // Check error type
    if !errs.IsType(err, errs.TypeValidation) {
        t.Errorf("expected validation error, got %v", err)
    }
    
    // Check context
    if e := errs.AsError(err); e != nil {
        if e.GetContext()["field"] != "email" {
            t.Error("missing context")
        }
    }
}
```

## Integration with Other Packages

To use in other Gollum packages:

```go
import "github.com/denkhaus/gollum/pkg/errs"

// Replace standard errors with typed errors
return errs.NotFound("not found")           // instead of errors.New("not found")
return errs.Validation("invalid input")      // instead of fmt.Errorf("invalid: %s", input)
```

## Design Decisions

- **Package name `errs`**: Avoids collision with stdlib `errors` package
- **Type enumeration**: Uses `int` for extensibility and performance
- **Optional context**: Context map is created only when needed
- **Standard interface compliance**: Implements `error` and `Unwrap()`
- **No dependencies**: Pure Go, no external dependencies

## Migration Guide

When migrating existing code:

```go
// Before
return fmt.Errorf("agent %s not found", id)

// After  
return errs.NotFoundf("agent %s not found", id).WithContext("agent_id", id)

// Before
return fmt.Errorf("validation failed: %w", err)

// After
return errs.Wrap(err, errs.TypeValidation, "validation failed")
```
