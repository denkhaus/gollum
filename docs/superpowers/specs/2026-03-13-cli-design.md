# CLI Design Specification

**Date:** 2026-03-13
**Status:** Design Approved
**Author:** AI Agent
**Reviewer:** Pending

## Overview

Integrate `urfave/cli/v3` to provide a proper command-line interface for Gollum. The default behavior runs the "default flow" from `.gollum/flows/default`, with subcommands for specific flow operations.

## Background

Gollum currently has:
- A TUI application started directly from `cmd/gollum/main.go`
- A standalone flow linter at `cmd/flows/lint/main.go`
- No unified CLI interface
- Flows defined in XML, loaded from `~/.config/gollum/flows` and workspace `.gollum/flows`

The goal is to:
1. Unify CLI entry points
2. Make "everything a flow" - even the default TUI loop
3. Provide `flow lint` and `flow run` commands
4. Maintain clean separation between CLI and core services

## Architecture

### Directory Structure

```
cmd/gollum/
├── main.go                 # Entry point, creates DI container, sets up CLI
└── cli/
    ├── root.go             # Root command handling, default flow execution
    └── flow/
        ├── flow.go         # Flow command group definition
        ├── lint.go         # gollum flow lint <path>
        └── run.go          # gollum flow run <path>
```

### Shared Package Extensions

```
pkg/shared/
├── cli.go              # CLI metadata helpers (injector access)
├── cli_flags.go        # Reusable CLI flag definitions
└── cli_helpers.go      # Reusable CLI action helpers
```

## Design Decisions

### 1. Reuse Existing DI Container

**Decision:** Use existing `pkg/di` container, no new DI layer.

**Rationale:**
- `di.NewContainer()` already provides all needed services
- Avoids duplicate DI logic
- Single source of truth for service instantiation

**Implementation:**
```go
container := di.NewContainer()
injector := container.RegisterServices(ctx)
defer container.Shutdown()
```

### 2. Metadata-Based Injector Storage

**Decision:** Store injector in CLI command metadata, accessed via shared helpers.

**Rationale:**
- Clean separation: CLI framework doesn't know about DI
- Type-safe access via helper functions
- Easy to test with mock injectors

**Helpers in `pkg/shared/cli.go`:**
- `SetInjector(cmd *cli.Command, injector do.Injector)`
- `GetInjector(cmd *cli.Command) (do.Injector, error)`
- `MustGetInjector(cmd *cli.Command) do.Injector`

### 3. One File Per Command

**Decision:** Each command in its own file with dedicated action functions.

**Rationale:**
- Clear separation of concerns
- Easy to locate and modify specific commands
- Natural fit for test files alongside implementation

**Pattern:**
```go
// cmd/gollum/cli/flow/lint.go
func LintAction(ctx context.Context, cmd *cli.Command) error {
    // Dedicated function, not inline
}
```

### 4. Default Flow Execution

**Decision:** Running `gollum` with no args executes the default flow.

**Resolution Order:**
1. `<workspace>/.gollum/flows/default/main.xml`
2. `~/.config/gollum/flows/default/main.xml`
3. Error if not found

**Rationale:**
- Aligns with "everything is a flow" philosophy
- Workspace-local flows override global
- TUI becomes one channel facade among others

### 5. Reusable Flags and Helpers

**Decision:** Common flags and helpers in `pkg/shared`.

**Rationale:**
- Avoid duplication across commands
- Consistent CLI experience
- Easier to maintain and extend

**Files:**
- `cli_flags.go`: Common flags (verbose, output format, etc.)
- `cli_helpers.go`: Common functions (GetPathArg, WriteOutput, etc.)

## Commands

### Root Command: `gollum`

**Usage:** `gollum`

**Behavior:**
- No arguments: Execute default flow
- Otherwise: Show help or run subcommand

**Exit Codes:**
- 0: Success
- 1: Execution error
- 2: Usage error

### Flow Command Group: `gollum flow`

#### `gollum flow lint <path>`

**Description:** Validate a flow file or module directory

**Arguments:**
- `path`: Path to `.xml` file or module directory

**Behavior:**
- Uses `pkg/flows/linter.LintModule()`
- Outputs validation results
- Exits 1 if invalid, 0 if valid

**Example:**
```bash
gollum flow lint .gollum/flows/examples/simple-flow.xml
gollum flow lint .gollum/flows/my-module/
```

#### `gollum flow run <path>`

**Description:** Execute a flow

**Arguments:**
- `path`: Path to `.xml` file or module directory

**Behavior:**
- Loads flow via `FlowRegistry` or parser
- Creates executor via `FlowExecutorService`
- Validates inputs
- Executes flow from initial state
- Returns output or error

**Example:**
```bash
gollum flow run .gollum/flows/examples/simple-flow.xml
gollum flow run .gollum/flows/my-module/
```

## Error Handling

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (validation failed, execution error) |
| 2 | Usage error (invalid arguments) |

### Error Handling Strategy

- **CLI errors:** Return `cli.Exit` with code and message
- **Service errors:** Wrap with context, return as CLI error
- **Validation errors:** Show clear, actionable messages

### Example

```go
func LintAction(ctx context.Context, cmd *cli.Command) error {
    path, err := shared.GetPathArg(cmd)
    if err != nil {
        return err // Returns cli.Exit with code 2
    }

    result := linter.LintModule(path)
    return shared.WriteLintResult(cmd, result)
}
```

## Testing Strategy

### Test-Driven Development (TDD)

**Approach:** Write tests before implementation.

### Unit Tests

- `pkg/shared/cli*.go`: Test all helpers
- `cmd/gollum/cli/**/*.go`: Test command actions with mocks

### Integration Tests

- Real flow files from `.gollum/flows/examples/`
- Real DI container with real services

### Test Structure

```
cmd/gollum/cli/
├── flow/
│   ├── lint.go
│   ├── lint_test.go
│   ├── run.go
│   └── run_test.go
└── root.go
    └── root_test.go
```

## Migration Notes

### To Be Removed

- `cmd/flows/lint/main.go` - Functionality moves to `gollum flow lint`

### To Be Modified

- `cmd/gollum/main.go` - Add CLI setup, keep TUI as default flow

### Backward Compatibility

- Existing TUI behavior preserved via default flow
- No breaking changes to flow definitions or loading

## Future Extensions

This design allows for:

- Additional flow commands (`validate`, `list`, `info`)
- Extension-added CLI commands via plugin system
- Skill commands
- Configuration management commands

## References

- `pkg/di/container.go` - DI container implementation
- `pkg/flows/` - Flow types, parser, executor, linter
- `pkg/shared/` - Shared utilities and types
- `github.com/urfave/cli/v3` - CLI framework documentation
