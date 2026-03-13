# CLI Design Specification

**Date:** 2026-03-13
**Version:** 2
**Status:** In Review (Addressing Feedback)
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
1. Check if CWD is inside a workspace (contains `.gollum/` directory)
2. If in workspace: Look for `<workspace>/.gollum/flows/default/main.xml`
3. Fallback: `~/.config/gollum/flows/default/main.xml`
4. Error if not found: "No default flow found. Create one at .gollum/flows/default/main.xml"

**Default Flow Structure:**
- Must be a module directory (not a single file)
- Must contain `main.xml` as entry point
- Can contain additional XML files referenced by `main.xml`

**Rationale:**
- Aligns with "everything is a flow" philosophy
- Workspace-local flows override global
- TUI becomes one channel facade among others

**Workspace Detection:**
```go
func isInWorkspace() bool {
    _, err := os.Stat(".gollum")
    return err == nil
}
```

### 5. Reusable Flags and Helpers

**Decision:** Common flags and helpers in `pkg/shared`.

**Rationale:**
- Avoid duplication across commands
- Consistent CLI experience
- Easier to maintain and extend

**Files:**
- `cli_flags.go`: Common flags (verbose, output format, etc.)
- `cli_helpers.go`: Common functions (GetPathArg, WriteOutput, etc.)

## Implementation Details

### Metadata Storage Implementation

The `urfave/cli/v3` `Command.Metadata` field is `map[string]interface{}`. We use a type-safe wrapper pattern:

```go
// pkg/shared/cli.go
const MetadataInjectorKey = "injector"

func SetInjector(cmd *cli.Command, injector do.Injector) {
    if cmd.Metadata == nil {
        cmd.Metadata = make(map[string]interface{})
    }
    cmd.Metadata[MetadataInjectorKey] = injector
}

func GetInjector(cmd *cli.Command) (do.Injector, error) {
    if cmd.Metadata == nil {
        return nil, fmt.Errorf("command metadata is nil")
    }
    injector, ok := cmd.Metadata[MetadataInjectorKey]
    if !ok {
        return nil, fmt.Errorf("injector not found in metadata")
    }
    typedInjector, ok := injector.(do.Injector)
    if !ok {
        return nil, fmt.Errorf("invalid injector type")
    }
    return typedInjector, nil
}

func MustGetInjector(cmd *cli.Command) do.Injector {
    injector, err := GetInjector(cmd)
    if err != nil {
        panic(err)
    }
    return injector
}
```

### Service Retrieval from DI

Commands retrieve services using the injector:

```go
func LintAction(ctx context.Context, cmd *cli.Command) error {
    injector := shared.MustGetInjector(cmd)
    linterSvc := do.MustInvoke[linter.LinterService](injector)
    // ... use linterSvc
}
```

### DI Shutdown Timing

Using `cli.Command.After` hook ensures cleanup after all actions:

```go
// cmd/gollum/main.go
func main() {
    ctx := context.Background()
    container := di.NewContainer()
    injector := container.RegisterServices(ctx)

    app := &cli.Command{
        Name: "gollum",
        Metadata: map[string]any{
            shared.MetadataInjectorKey: injector,
        },
        After: func(ctx context.Context, cmd *cli.Command) error {
            container.Shutdown()
            return nil
        },
    }

    app.Run(ctx, os.Args)
}
```

### Exit Code Handling

Helper function for consistent exit code propagation:

```go
// pkg/shared/cli_helpers.go
func ExitCode(err error) int {
    if err == nil {
        return 0
    }
    if exitErr, ok := err.(cli.ExitCoder); ok {
        return exitErr.ExitCode()
    }
    return 1 // default error code
}
```

Commands return `cli.Exit` for explicit exit codes:

```go
return cli.Exit("validation failed", 1)
```

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
- Outputs validation results in human-readable format (same as existing `cmd/flows/lint`)
- Supports `--format=json` flag for structured output (future)
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

**Approach:** Write tests before implementation following Red-Green-Refactor cycle.

### Test Coverage Matrix

| Component | Tests Required | Test Type |
|-----------|----------------|-----------|
| `pkg/shared/cli.go` | SetInjector, GetInjector, MustGetInjector | Unit |
| `pkg/shared/cli_flags.go` | Flag definitions | N/A (declarative) |
| `pkg/shared/cli_helpers.go` | GetPathArg, WriteOutput, etc. | Unit |
| `cmd/gollum/cli/root.go` | Default flow execution, workspace detection | Unit + Integration |
| `cmd/gollum/cli/flow/lint.go` | Lint action, output formatting | Unit + Integration |
| `cmd/gollum/cli/flow/run.go` | Run action, flow execution | Unit + Integration |
| Exit code propagation | All commands | Unit |
| Error handling | All commands | Unit |

### Unit Tests

- **`pkg/shared/cli*.go`**: Test all helpers with mock commands
- **`cmd/gollum/cli/**/*.go`**: Test command actions with mock injectors
- Use table-driven tests for multiple scenarios
- Mock external dependencies (filesystem, DI services)

### Integration Tests

- Real flow files from `.gollum/flows/examples/`
- Real DI container with real services
- Test actual CLI execution paths
- Tagged with `// +build integration` for separate running

### Test Structure

```
cmd/gollum/cli/
├── flow/
│   ├── lint.go
│   ├── lint_test.go              # Unit tests
│   ├── lint_integration_test.go  # Integration tests
│   ├── run.go
│   ├── run_test.go
│   └── run_integration_test.go
└── root.go
    ├── root_test.go
    └── root_integration_test.go
```

### Test Data

Use `.gollum/flows/examples/` for test fixtures:
- `simple-flow.xml` - Valid flow
- `invalid-flow.xml` - Invalid flow (create if needed)
- `test-module/` - Module directory with main.xml

## Migration Plan

### Phase 1: Foundation (No Breaking Changes)

1. **Add shared CLI helpers** (`pkg/shared/cli*.go`)
   - Write tests first (TDD)
   - Implement metadata access functions
   - Add reusable flags and helpers

2. **Create CLI package structure** (`cmd/gollum/cli/`)
   - Add `root.go` with basic command registration
   - Add `flow/` subdirectory

3. **Update main.go minimally**
   - Wrap existing `startup()` in CLI framework
   - Keep TUI as default behavior
   - Verify existing functionality works

### Phase 2: Flow Commands

4. **Implement `gollum flow lint`**
   - Migrate functionality from `cmd/flows/lint/main.go`
   - Add tests
   - Verify output format matches existing

5. **Implement `gollum flow run`**
   - Write tests first
   - Implement using FlowExecutorService
   - Test with example flows

### Phase 3: Default Flow

6. **Implement default flow execution**
   - Add workspace detection
   - Implement flow resolution logic
   - Create example default flow

7. **Clean up**
   - Remove `cmd/flows/lint/main.go`
   - Update documentation

### Verification Checklist

- [ ] TUI still works with `gollum` command
- [ ] `gollum flow lint` produces same output as old `flows-lint`
- [ ] `gollum flow run` executes example flows correctly
- [ ] Exit codes are correct for all scenarios
- [ ] All tests pass
- [ ] No regression in existing functionality

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

## Appendix: Code Examples

### Example 1: Complete `lint.go` Implementation

```go
// cmd/gollum/cli/flow/lint.go
package flow

import (
    "context"
    "fmt"

    "github.com/denkhaus/gollum/pkg/flows/linter"
    "github.com/denkhaus/gollum/pkg/shared"
    "github.com/urfave/cli/v3"
)

// LintCommand returns the flow lint command
func LintCommand() *cli.Command {
    return &cli.Command{
        Name:  "lint",
        Usage: "Validate a flow file or module directory",
        ArgsUsage: "<path>",
        Flags: []cli.Flag{
            shared.VerboseFlag,
            shared.OutputFormatFlag,
        },
        Action: LintAction,
    }
}

// LintAction executes the flow lint command
func LintAction(ctx context.Context, cmd *cli.Command) error {
    path, err := shared.GetPathArg(cmd)
    if err != nil {
        return err
    }

    result := linter.LintModule(path)
    return WriteLintResult(cmd, result)
}

// WriteLintResult outputs the lint result and returns appropriate exit code
func WriteLintResult(cmd *cli.Command, result *linter.ModuleLinterResult) error {
    fmt.Fprint(cmd.Root().Writer, result.String())

    if !result.Valid {
        return cli.Exit("", 1)
    }
    return nil
}
```

### Example 2: Complete `root.go` Implementation

```go
// cmd/gollum/cli/root.go
package cli

import (
    "context"
    "fmt"
    "os"
    "path/filepath"

    "github.com/denkhaus/gollum/pkg/shared"
    "github.com/urfave/cli/v3"
)

// RootCommand returns the root CLI command
func RootCommand() *cli.Command {
    return &cli.Command{
        Name:  "gollum",
        Usage: "AI agent workflow system",
        Action: RunDefaultFlow,
        Commands: []*cli.Command{
            flow.FlowCommandGroup(),
        },
    }
}

// RunDefaultFlow executes the default flow
func RunDefaultFlow(ctx context.Context, cmd *cli.Command) error {
    flowPath, err := resolveDefaultFlowPath()
    if err != nil {
        return fmt.Errorf("default flow not found: %w\n"+
            "Create a default flow at .gollum/flows/default/main.xml", err)
    }

    injector := shared.MustGetInjector(cmd)
    return executeFlow(injector, flowPath)
}

// resolveDefaultFlowPath finds the default flow location
func resolveDefaultFlowPath() (string, error) {
    // Check workspace-local first
    if isInWorkspace() {
        cwd, _ := os.Getwd()
        localPath := filepath.Join(cwd, ".gollum", "flows", "default", "main.xml")
        if _, err := os.Stat(localPath); err == nil {
            return localPath, nil
        }
    }

    // Check global config directory
    home, _ := os.UserHomeDir()
    globalPath := filepath.Join(home, ".config", "gollum", "flows", "default", "main.xml")
    if _, err := os.Stat(globalPath); err == nil {
        return globalPath, nil
    }

    return "", fmt.Errorf("no default flow found")
}

// isInWorkspace checks if current directory is a gollum workspace
func isInWorkspace() bool {
    _, err := os.Stat(".gollum")
    return err == nil
}
```

### Example 3: Mock Injector Test

```go
// cmd/gollum/cli/flow/lint_test.go
package flow_test

import (
    "context"
    "testing"

    "github.com/denkhaus/gollum/cmd/gollum/cli/flow"
    "github.com/denkhaus/gollum/pkg/shared"
    "github.com/samber/do/v2"
    "github.com/urfave/cli/v3"
    "gotest.tools/v3/assert"
)

func TestLintAction_InvalidPath(t *testing.T) {
    // Setup mock injector
    injector := do.New()
    cmd := &cli.Command{
        Metadata: map[string]any{
            shared.MetadataInjectorKey: injector,
        },
    }
    cmd.SetArgs([]string{"nonexistent.xml"})

    err := flow.LintAction(context.Background(), cmd)
    assert.Error(t, err)
}
```

### Example 4: Integration Test

```go
// cmd/gollum/cli/flow/lint_integration_test.go
package flow_test

import (
    "context"
    "os"
    "testing"

    "github.com/denkhaus/gollum/cmd/gollum/cli/flow"
    "github.com/denkhaus/gollum/pkg/di"
    "github.com/denkhaus/gollum/pkg/shared"
    "github.com/urfave/cli/v3"
    "gotest.tools/v3/assert"
)

func TestLintAction_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Setup real DI container
    ctx := context.Background()
    container := di.NewContainer()
    injector := container.RegisterServices(ctx)
    defer container.Shutdown()

    // Setup command with real injector
    cmd := &cli.Command{
        Metadata: map[string]any{
            shared.MetadataInjectorKey: injector,
        },
        Root: &cli.Command{Writer: os.Stdout},
    }

    // Test with example flow
    cmd.SetArgs([]string{"../../.gollum/flows/examples/simple-flow.xml"})
    err := flow.LintAction(ctx, cmd)
    assert.NilError(t, err)
}
```

## References

- `pkg/di/container.go` - DI container implementation
- `pkg/flows/` - Flow types, parser, executor, linter
- `pkg/shared/` - Shared utilities and types
- `github.com/urfave/cli/v3` - CLI framework documentation
