# Extension Development Guide

This guide explains how to create extensions and func steps for Gollum.

## Overview

Gollum supports two types of extensions:

1. **Func Steps** - Lightweight functions executed via Scriggo VM (~10x faster than Yaegi)
2. **Extensions** - Full packages loaded via Yaegi with DI integration

## Func Steps

### What are Func Steps?

Func steps are simple Go functions that can be called from flows. They are compiled once and executed many times via bytecode, making them extremely efficient.

### Creating a Func Step

1. Create a `.go` file in `~/.config/gollum/functions/` or `<workspace>/.gollum/functions/`
2. Add exported functions
3. Restart Gollum or trigger reload

### Example

```go
// ~/.config/gollum/functions/double.go
package main

func Double(x int) int {
    return x * 2
}
```

### Usage in Flow

```xml
<step type="func" function="Double">
    <params>
        <param name="x" value="${input.value}"/>
    </params>
    <output assign="${output.doubled}"/>
</step>
```

### Limitations

- No package imports other than standard library
- No access to external services (use Extensions instead)
- Single file per function
- Simple parameter types (string, int, bool, float, map, array)

## Extensions

### What are Extensions?

Extensions are full Go packages that can register services with the DI container and access all Gollum services.

### Creating an Extension

1. Create a directory in `~/.config/gollum/extensions/<name>/` or `<workspace>/.gollum/extensions/<name>/`
2. Add `main.go` with `Init() error` function
3. Register services in `Init()`
4. Restart Gollum

### Example

```go
// ~/.config/gollum/extensions/myservice/main.go
package main

import "github.com/samber/do/v2"

var injector do.Injector

type MyService interface {
    DoWork() error
}

type myServiceImpl struct{}

func NewMyService(inj do.Injector) (MyService, error) {
    return &myServiceImpl{}, nil
}

func (s *myServiceImpl) DoWork() error {
    // Implementation
    return nil
}

func Init() error {
    do.Provide(injector, NewMyService)
    return nil
}
```

### DI Integration

Extensions have full access to the DI container:

```go
func Init() error {
    // Register a service
    do.Provide(injector, NewMyService)

    // Consume existing services
    log := do.MustInvoke[logger.LoggerService](injector)
    log.Info("Extension loaded")

    return nil
}
```

### Hook Registration

Extensions can register lifecycle hooks:

```go
func Init() error {
    hooks := do.MustInvoke[extensions.HookRegistry](injector)

    hooks.Register(extensions.HookAgentPreExecute, "my-hook", func(ctx *extensions.HookContext) error {
        fmt.Printf("Agent %s is about to execute\n", ctx.AgentID)
        return nil
    })

    return nil
}
```

## Directory Structure

```
~/.config/gollum/                    # Global (user-level)
├── extensions/
│   └── notification/                # Example extension
│       └── main.go
└── functions/
    ├── string_utils.go              # Example func step
    └── math_utils.go

<workspace>/.gollum/                # Workspace-local (higher priority)
├── extensions/
│   └── project-hooks/               # Project-specific extensions
│       └── main.go
└── functions/
    └── transform_data.go            # Project-specific func steps
```

## Priority

Workspace-local definitions **override** user space:
1. `<workspace>/.gollum/` (highest priority)
2. `~/.config/gollum/` (fallback)

## Best Practices

### Func Steps
- Keep functions simple and pure
- Use for data transformations
- Avoid side effects
- Document parameters and return types

### Extensions
- Use clear interfaces
- Handle initialization errors gracefully
- Clean up resources if needed
- Log important events
- Validate configuration

## Security

- All extension and function names are validated
- No path traversal allowed
- Execution timeouts enforced
- Sandboxed execution environment

## Troubleshooting

### Extension fails to load

Check logs for:
- Compilation errors
- Missing Init function
- DI resolution failures

### Func step not found

Verify:
- File is in correct directory
- File has `.go` extension
- Function name matches usage in flow
- No compilation errors

## Examples

See `examples/extensions/` and `examples/functions/` for complete examples.
