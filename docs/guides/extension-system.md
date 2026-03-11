# Gollum Extension System

Gollum's extension system enables custom functionality through two mechanisms:

## Quick Start

### Add a Func Step

Create `~/.config/gollum/functions/hello.go`:

```go
package main

func Hello(name string) string {
    return "Hello, " + name
}
```

Use in flow:
```xml
<step type="func" function="Hello">
    <params>
        <param name="name" value="${input.user}"/>
    </params>
    <output assign="${output.greeting}"/>
</step>
```

### Add an Extension

Create `~/.config/gollum/extensions/myservice/main.go`:

```go
package main

import "github.com/samber/do/v2"

var injector do.Injector

func Init() error {
    // Register your service
    do.Provide(injector, NewMyService)
    return nil
}
```

## Documentation

- [Extension Development Guide](guides/extension-development.md)
- [Examples](../examples/)

## Architecture

The extension system uses:
- **Scriggo VM** for func steps (compiled, ~10x faster)
- **Yaegi** for extensions (interpreted, full Go support)
- **samber/do** for DI integration

See [design document](plans/2025-03-08-extension-system-design.md) for details.

## Hooks and Lifecycle

Flow execution can be extended using hooks from `pkg/hooks.HookManager`:

- **BeforeFlowStep / AfterFlowStep** - Monitor all step execution
- **BeforeToolExecution / AfterToolExecution** - Modify tool behavior
- **BeforeLLMRequest / AfterLLMResponse** - Modify LLM prompts/responses

See [Executor Hooks Guide](guides/executor-hooks.md) for details.
