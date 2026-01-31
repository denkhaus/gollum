# Coding Conventions

**Analysis Date:** 2026-01-31

## Naming Patterns

### Files and Packages
- **Files**: `snake_case.go` (e.g., `file_state_manager.go`)
- **Packages**: Lowercase with meaningful names matching directory structure
- **Test files**: `*_test.go` suffix for unit tests, `*_spec_test.go` for specification tests
- **Mock files**: `mock_*.go` in centralized `pkg/mocks/` directory

### Functions and Methods
- **Public**: `PascalCase` (e.g., `NewContainer`, `RegisterServices`)
- **Private**: `camelCase` (e.g., `runBashCommand`, `setupTestInjector`)
- **Test functions**: `Test*` prefix (e.g., `TestBashTool_Run_Success`)
- **Spec tests**: `Spec*` prefix (e.g., `EditTool_Spec`)

### Variables and Constants
- **Variables**: `camelCase` (e.g., `agentID`, `bashCfg`, `logService`)
- **Constants**: `PascalCase` (e.g., `TypeValidation`, `TypeInternal`)
- **Constants with types**: `PascalCase` (e.g., `OutputModeFull`)

### Structs and Interfaces
- **Structs**: `PascalCase` (e.g., `BashTool`, `Container`)
- **Interfaces**: `PascalCase` with "er" suffix (e.g., `BashToolProvider`, `LoggerService`)
- **Private struct fields**: `camelCase`

### Types and Enums
- **Custom types**: `PascalCase` (e.g., `Type`, `OutputMode`)
- **Type aliases**: `PascalCase` (e.g., `AgentRegistry`)
- **String enums**: Use string constants with explicit conversion

## Code Style

### Formatting
- **Tool**: `golangci-lint` with standard formatter configuration
- **Code formatting**: `gofmt` and `goimports`
- **Line length**: No explicit limit, but readable and functional
- **Braces**: Always on the same line (K&R style)
- **Indentation**: Tabs for indentation (standard Go convention)

### Linting
**Configuration**: `.golangci.yml` with comprehensive linters enabled:
- `errcheck` - Unhandled errors
- `govet` - Go vet checks
- `staticcheck` - Static analysis
- `unused` - Unused variables and imports
- `ineffassign` - Inefficient assignments
- `misspell` - Misspelling detection
- `unconvert` - Unnecessary type conversions
- `unparam` - Unused function parameters
- `nakedret` - Naked return statements
- `prealloc` - Pre-allocation suggestions
- `gocritic` - Code quality suggestions
- `gocyclo` - Cyclomatic complexity
- `revive` - Revive linter
- `goconst` - String constants
- `forbidigo` - Forbidden patterns

### Error Handling
- **Custom error types**: Use `errs` package structured errors
- **Error wrapping**: Use `Wrap`/`Wrapf` to preserve context
- **Error types**: Use specific error types from `errs` package
- **Nil checks**: Always check for nil errors
- **Error logging**: Use structured logging with context

```go
// Good error handling
if err != nil {
    return nil, errs.Wrap(err, errs.TypeInternal, "failed to create container")
}

// Good error creation
return errs.Validationf("invalid command: %s", cmd)
```

## Import Organization

### Order
1. Standard library packages (alphabetical)
2. Third-party packages (alphabetical)
3. Internal packages (alphabetical)

### Import Format
```go
import (
    "context"
    "fmt"
    "os"

    "github.com/google/uuid"
    "github.com/m-mizutani/gollem"
    "github.com/samber/do/v2"
    "go.uber.org/zap"

    "github.com/denkhaus/gollum/pkg/config"
    "github.com/denkhaus/gollum/pkg/logger"
    "github.com/denkhaus/gollum/pkg/shared"
)
```

### Package Aliases
- Use explicit aliases for clarity
- Avoid `_` import alias

## Structured Patterns

### Dependency Injection
- **Pattern**: Constructor-based DI with `samber/do/v2`
- **Provider interfaces**: Interface for each service
- **Registration**: `do.Provide()` in container setup
- **Resolution**: `do.MustInvoke[]` in tests

```go
// Provider pattern
type BashToolProvider interface {
    CreateTool(agentID uuid.UUID) *BashTool
}

// DI registration
do.Provide(p.injector, tools.NewBashToolProvider)
```

### Tool Implementation Pattern
1. Define tool struct with dependencies
2. Implement provider interface
3. Add to DI container
4. Register in default agent configuration

### Interface Design
- **Minimal interfaces**: Interface only includes necessary methods
- **Specific interfaces**: Separate interfaces for different responsibilities
- **Documentation**: Interface documentation uses Go conventions

### Error Handling Pattern
- Structured error types with context
- Consistent error wrapping
- Type-safe error checking
- Convenience constructors for common error types

## Comments and Documentation

### Function Comments
- Public functions include Go doc comments
- Document purpose, parameters, and returns
- Example for complex functions

```go
// Spec returns the tool specification for the Bash tool
func (t *BashTool) Spec() gollem.ToolSpec {
    return gollem.ToolSpec{
        Name:        shared.ToolNameBash,
        Description: "Executes bash commands and returns the output. Useful for running shell commands, scripts, and system operations.",
        Parameters: map[string]*gollem.Parameter{
            "command": {
                Type:        gollem.TypeString,
                Description: "The bash command to execute (e.g., 'ls -la', 'git status', 'echo hello')",
            },
            "timeout": {
                Type:        gollem.TypeNumber,
                Description: "Optional timeout in seconds (default: 30, max: 120)",
            },
        },
        Required: []string{"command"},
    }
}
```

### Interface Comments
- Describe interface purpose
- Document method contracts
- Document behavior expectations

### Code Comments
- Use sparingly for complex logic
- Explain "why" not "what"
- TODO comments with clear actions

## Module Design

### Package Organization
- **pkg/**: Core application packages
  - `agents/` - Agent implementations
  - `config/` - Configuration management
  - `di/` - Dependency injection
  - `hooks/` - Hook system
  - `llm/` - LLM client management
  - `logger/` - Logging service
  - `registry/` - Agent registry
  - `shared/` - Common types
  - `state/` - File state management
  - `tools/` - Tool implementations
  - `ui/` - User interface components

### Exports
- Minimal public APIs per package
- Private implementation details
- Clear package boundaries

### Barrel Files
- Not used consistently
- Prefer direct imports for clarity

---

*Convention analysis: 2026-01-31*