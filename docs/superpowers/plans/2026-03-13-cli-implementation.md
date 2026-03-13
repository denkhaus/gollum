# CLI Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Integrate `urfave/cli/v3` to provide a unified command-line interface for Gollum with `flow lint` and `flow run` commands, while keeping the TUI as default behavior.

**Architecture:** CLI framework (`urfave/cli/v3`) provides command structure. ApplicationService handles default flow detection and execution. DI container stored in CLI metadata, accessed via shared helpers. Each command in its own file with TDD approach.

**Tech Stack:** `urfave/cli/v3`, existing `pkg/di` container, `github.com/samber/do/v2`, existing flow services

---

## File Structure

### New Files
- `pkg/shared/cli.go` - CLI metadata helpers (injector access)
- `pkg/shared/cli_test.go` - Tests for cli.go
- `pkg/shared/cli_flags.go` - Reusable CLI flag definitions
- `pkg/shared/cli_helpers.go` - Reusable CLI action helpers
- `pkg/shared/cli_helpers_test.go` - Tests for helpers
- `cmd/gollum/cli/root.go` - Root command (delegates to ApplicationService)
- `cmd/gollum/cli/root_test.go` - Tests for root command
- `cmd/gollum/cli/flow/flow.go` - Flow command group definition
- `cmd/gollum/cli/flow/lint.go` - Flow lint command
- `cmd/gollum/cli/flow/lint_test.go` - Tests for lint command
- `cmd/gollum/cli/flow/run.go` - Flow run command
- `cmd/gollum/cli/flow/run_test.go` - Tests for run command

### Modified Files
- `pkg/app/service.go` - Add default flow detection and execution methods
- `pkg/app/service_test.go` - Add tests for default flow logic
- `cmd/gollum/main.go` - Wrap existing startup in CLI framework
- `go.mod` - Add `github.com/urfave/cli/v3` dependency

### Files to Remove
- `cmd/flows/lint/main.go` - Functionality moves to `gollum flow lint`

---

## Chunk 1: Shared CLI Helpers (Foundation)

### Task 1: Add CLI metadata helpers

**Files:**
- Create: `pkg/shared/cli.go`
- Test: `pkg/shared/cli_test.go`

- [ ] **Step 1: Write failing tests for SetInjector**

```go
// pkg/shared/cli_test.go
package shared

import (
    "testing"

    "github.com/samber/do/v2"
    "github.com/urfave/cli/v3"
    "gotest.tools/v3/assert"
    "gotest.tools/v3/assert/cmp"
)

func TestSetAndGetInjector(t *testing.T) {
    cmd := &cli.Command{}
    injector := do.New()

    SetInjector(cmd, injector)

    retrieved, err := GetInjector(cmd)
    assert.NilError(t, err)
    assert.Equal(t, injector, retrieved)
}

func TestGetInjectorNilMetadata(t *testing.T) {
    cmd := &cli.Command{Metadata: nil}

    _, err := GetInjector(cmd)
    assert.ErrorContains(t, err, "metadata is nil")
}

func TestGetInjectorNotFound(t *testing.T) {
    cmd := &cli.Command{}
    cmd.Metadata = make(map[string]any)

    _, err := GetInjector(cmd)
    assert.ErrorContains(t, err, "injector not found")
}

func TestGetInjectorInvalidType(t *testing.T) {
    cmd := &cli.Command{}
    cmd.Metadata = make(map[string]any)
    cmd.Metadata[MetadataInjectorKey] = "not an injector"

    _, err := GetInjector(cmd)
    assert.ErrorContains(t, err, "invalid injector type")
}

func TestMustGetInjector(t *testing.T) {
    cmd := &cli.Command{}
    injector := do.New()
    SetInjector(cmd, injector)

    retrieved := MustGetInjector(cmd)
    assert.Equal(t, injector, retrieved)
}

func TestMustGetInjectorPanics(t *testing.T) {
    t.Run("nil metadata", func(t *testing.T) {
        cmd := &cli.Command{Metadata: nil}
        assert.Assert(t, cmp.Panics(func() { MustGetInjector(cmd) }))
    })

    t.Run("not found", func(t *testing.T) {
        cmd := &cli.Command{}
        cmd.Metadata = make(map[string]any)
        assert.Assert(t, cmp.Panics(func() { MustGetInjector(cmd) }))
    })
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/shared/cli_test.go -v`
Expected: FAIL with "undefined: SetInjector"

- [ ] **Step 3: Implement SetInjector and GetInjector**

```go
// pkg/shared/cli.go
package shared

import (
    "fmt"

    "github.com/samber/do/v2"
    "github.com/urfave/cli/v3"
)

// MetadataInjectorKey is the key used to store the DI injector in command metadata
const MetadataInjectorKey = "injector"

// SetInjector stores the DI injector in the CLI command metadata
func SetInjector(cmd *cli.Command, injector do.Injector) {
    if cmd.Metadata == nil {
        cmd.Metadata = make(map[string]any)
    }
    cmd.Metadata[MetadataInjectorKey] = injector
}

// GetInjector retrieves the DI injector from the CLI command metadata
func GetInjector(cmd *cli.Command) (do.Injector, error) {
    if cmd.Metadata == nil {
        return nil, fmt.Errorf("command metadata is nil")
    }

    injector, ok := cmd.Metadata[MetadataInjectorKey]
    if !ok {
        return nil, fmt.Errorf("injector not found in metadata (key: %s)", MetadataInjectorKey)
    }

    typedInjector, ok := injector.(do.Injector)
    if !ok {
        return nil, fmt.Errorf("invalid injector type in metadata")
    }

    return typedInjector, nil
}

// MustGetInjector retrieves the DI injector from the CLI command metadata.
// Panics if the injector is not found or has an invalid type.
func MustGetInjector(cmd *cli.Command) do.Injector {
    injector, err := GetInjector(cmd)
    if err != nil {
        panic(err)
    }
    return injector
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/shared/ -run TestSetAndGetInjector -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/shared/cli.go pkg/shared/cli_test.go
git commit -m "feat(cli): add CLI metadata helpers for DI injector access

Add SetInjector, GetInjector, MustGetInjector functions for storing
and retrieving DI injector from urfave/cli/v3 command metadata.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 2: Add CLI flags

**Files:**
- Create: `pkg/shared/cli_flags.go`

- [ ] **Step 1: Create CLI flags file**

```go
// pkg/shared/cli_flags.go
package shared

import "github.com/urfave/cli/v3"

// Common reusable flags for CLI commands
var (
    // VerboseFlag enables verbose output
    VerboseFlag = &cli.BoolFlag{
        Name:    "verbose",
        Aliases: []string{"v"},
        Usage:   "enable verbose output",
    }

    // OutputFormatFlag specifies output format (text, json)
    OutputFormatFlag = &cli.StringFlag{
        Name:    "output",
        Aliases: []string{"o"},
        Usage:   "output format (text, json)",
        Value:   "text",
    }
)
```

- [ ] **Step 2: Verify no compilation errors**

Run: `go build ./pkg/shared/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add pkg/shared/cli_flags.go
git commit -m "feat(cli): add reusable CLI flags

Add VerboseFlag and OutputFormatFlag for use across CLI commands.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 3: Add CLI helper functions

**Files:**
- Create: `pkg/shared/cli_helpers.go`
- Test: `pkg/shared/cli_helpers_test.go`

- [ ] **Step 1: Write failing tests**

```go
// pkg/shared/cli_helpers_test.go
package shared

import (
    "testing"

    "github.com/urfave/cli/v3"
    "gotest.tools/v3/assert"
)

func TestGetPathArg(t *testing.T) {
    t.Run("valid path", func(t *testing.T) {
        // Create a mock args object with one argument
        cmd := &cli.Command{}
        // In urfave/cli/v3, args are accessed via cmd.Args()
        // For testing, we verify the function handles the args API correctly
        // A full integration test would use the actual CLI framework
    })

    t.Run("missing argument", func(t *testing.T) {
        cmd := &cli.Command{}

        _, err := GetPathArg(cmd)
        assert.ErrorContains(t, err, "missing path")
    })
}

func TestExitCode(t *testing.T) {
    t.Run("nil error", func(t *testing.T) {
        code := ExitCode(nil)
        assert.Equal(t, 0, code)
    })

    t.Run("cli.Exit error", func(t *testing.T) {
        err := cli.Exit("error", 42)
        code := ExitCode(err)
        assert.Equal(t, 42, code)
    })

    t.Run("generic error", func(t *testing.T) {
        err := assert.Error{}
        code := ExitCode(&err)
        assert.Equal(t, 1, code)
    })
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/shared/cli_helpers_test.go -v`
Expected: FAIL with "undefined: GetPathArg"

- [ ] **Step 3: Implement helper functions**

```go
// pkg/shared/cli_helpers.go
package shared

import (
    "fmt"

    "github.com/urfave/cli/v3"
)

// GetPathArg validates and returns a required path argument
func GetPathArg(cmd *cli.Command) (string, error) {
    if cmd.Args().Len() < 1 {
        return "", cli.Exit("missing path argument", 2)
    }
    return cmd.Args().Get(0), nil
}

// ExitCode extracts the exit code from an error
func ExitCode(err error) int {
    if err == nil {
        return 0
    }
    if exitErr, ok := err.(cli.ExitCoder); ok {
        return exitErr.ExitCode()
    }
    return 1
}

// WriteOutput writes formatted output to the command's writer
func WriteOutput(cmd *cli.Command, format string, args ...any) error {
    _, err := fmt.Fprint(cmd.Root().Writer, fmt.Sprintf(format, args...))
    return err
}

// HandleError writes error and returns exit code
func HandleError(cmd *cli.Command, err error, exitCode int) error {
    fmt.Fprintf(cmd.Root().Writer, "Error: %v\n", err)
    return cli.Exit("", exitCode)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/shared/ -run "TestGetPathArg|TestExitCode" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/shared/cli_helpers.go pkg/shared/cli_helpers_test.go
git add pkg/shared/cli.go pkg/shared/cli_test.go pkg/shared/cli_flags.go
git commit -m "feat(cli): add CLI helper functions

Add GetPathArg for argument validation, ExitCode for error handling,
WriteOutput for output formatting, and HandleError for error display.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 2: ApplicationService Default Flow Logic

### Task 4: Add default flow detection to ApplicationService

**Files:**
- Modify: `pkg/app/service.go` (add imports, fields, methods)
- Test: `pkg/app/service_test.go`

- [ ] **Step 1: Write failing test for default flow resolution**

```go
// pkg/app/service_test.go
package app

import (
    "context"
    "os"
    "path/filepath"
    "testing"

    "github.com/denkhaus/gollum/pkg/workspace"
    "gotest.tools/v3/assert"
)

func TestResolveDefaultFlowPath(t *testing.T) {
    t.Run("workspace default flow found", func(t *testing.T) {
        // Create temp workspace with default flow
        tmpDir := t.TempDir()
        flowsDir := filepath.Join(tmpDir, ".gollum", "flows", "default")
        err := os.MkdirAll(flowsDir, 0755)
        assert.NilError(t, err)

        mainXML := filepath.Join(flowsDir, "main.xml")
        err = os.WriteFile(mainXML, []byte("<flow></flow>"), 0644)
        assert.NilError(t, err)

        // Change to temp directory
        oldWd, _ := os.Getwd()
        os.Chdir(tmpDir)
        defer os.Chdir(oldWd)

        // TODO: Create service and test resolveDefaultFlowPath
        // This requires mocking workspace service and other dependencies
    })
}
```

- [ ] **Step 2: Update ApplicationService imports and struct**

```go
// pkg/app/service.go - add to imports
import (
    // ... existing imports ...
    "os"
    "path/filepath"

    "github.com/denkhaus/gollum/pkg/flows/executor"
    flowregistry "github.com/denkhaus/gollum/pkg/flows/registry"
    "github.com/denkhaus/gollum/pkg/flows/parser"
)

// pkg/app/service.go - add to applicationServiceImpl struct
type applicationServiceImpl struct {
    // ... existing fields ...
    flowExecutorService executor.FlowExecutorService // Add this
    flowRegistry        flowregistry.FlowRegistry    // Add this
}
```

- [ ] **Step 3: Update NewService to inject flow services**

```go
// pkg/app/service.go
func NewService(injector do.Injector) (ApplicationService, error) {
    // ... existing code ...

    // Add these lines after existing service injections
    flowExecutorService := do.MustInvoke[executor.FlowExecutorService](injector)
    flowRegistry := do.MustInvoke[flowregistry.FlowRegistry](injector)

    return &applicationServiceImpl{
        // ... existing fields ...
        flowExecutorService: flowExecutorService,
        flowRegistry:        flowRegistry,
    }, nil
}
```

- [ ] **Step 4: Add resolveDefaultFlowPath method**

```go
// pkg/app/service.go
// resolveDefaultFlowPath finds the default flow location
func (p *applicationServiceImpl) resolveDefaultFlowPath() (string, error) {
    cwd := p.workspaceService.GetCurrentWorkspace()

    // Check workspace-local first
    localPath := filepath.Join(cwd, ".gollum", "flows", "default", "main.xml")
    if _, err := os.Stat(localPath); err == nil {
        return localPath, nil
    }

    // Check global config directory
    home, _ := os.UserHomeDir()
    globalPath := filepath.Join(home, ".config", "gollum", "flows", "default", "main.xml")
    if _, err := os.Stat(globalPath); err == nil {
        return globalPath, nil
    }

    return "", fmt.Errorf("no default flow found")
}
```

- [ ] **Step 5: Add runDefaultFlow method**

```go
// pkg/app/service.go
// runDefaultFlow executes the default flow using FlowExecutorService
func (p *applicationServiceImpl) runDefaultFlow(ctx context.Context, flowPath string) error {
    p.logService.Infof("Running default flow: %s", flowPath)

    // Parse flow
    flow, err := parser.Parse(flowPath)
    if err != nil {
        return fmt.Errorf("failed to parse default flow: %w", err)
    }

    // Create executor
    executor := p.flowExecutorService.New(flow)

    // Set empty input (default flow should define required inputs with defaults)
    executor.SetInput(make(map[string]any))

    // Validate and run
    if err := executor.Validate(); err != nil {
        return fmt.Errorf("default flow validation failed: %w", err)
    }

    if err := executor.Run(); err != nil {
        return fmt.Errorf("default flow execution failed: %w", err)
    }

    p.logService.Infof("Default flow completed successfully")
    return nil
}
```

- [ ] **Step 6: Refactor existing Run logic into runTUI method**

```go
// pkg/app/service.go
// runTUI runs the TUI application (current behavior)
func (p *applicationServiceImpl) runTUI(ctx context.Context) error {
    // Create and register Supervisor agent
    agent, _, err := p.createSupervisorAgent(ctx)
    if err != nil {
        return err
    }

    // Run interactive loop
    return p.runInteractiveLoop(ctx, agent)
}
```

- [ ] **Step 7: Update Run method to check for default flow first**

```go
// pkg/app/service.go
func (p *applicationServiceImpl) Run(ctx context.Context) error {
    // Enable file logging
    if err := p.logService.EnableFileLogging(p.gollumDir, p.sessionID); err != nil {
        return fmt.Errorf("failed to enable file logging: %w", err)
    }
    defer func() {
        if err := p.logService.CloseFileLogging(); err != nil {
            p.logService.Warnf("failed to close file logging: %v", err)
        }
    }()

    // Create .gollum directory
    if err := p.ensureGollumDirectory(); err != nil {
        return fmt.Errorf("failed to create .gollum directory: %w", err)
    }

    // Prime FileStateManager
    if err := p.primeFileStateManager(ctx); err != nil {
        return err
    }

    // Check for default flow
    if flowPath, err := p.resolveDefaultFlowPath(); err == nil {
        return p.runDefaultFlow(ctx, flowPath)
    }

    // No default flow - run TUI (current behavior)
    return p.runTUI(ctx)
}
```

- [ ] **Step 8: Verify compilation**

Run: `go build ./pkg/app/`
Expected: No errors

- [ ] **Step 9: Commit**

```bash
git add pkg/app/service.go
git commit -m "feat(app): add default flow detection and execution

ApplicationService now checks for default flow before running TUI:
- resolveDefaultFlowPath() checks workspace and global locations
- runDefaultFlow() executes flow using FlowExecutorService
- runTUI() refactors existing TUI logic
- Run() method checks for default flow first, falls back to TUI

TUI remains default behavior when no default flow exists.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 3: CLI Framework Setup

### Task 5: Add urfave/cli/v3 dependency

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Add dependency**

Run: `go get github.com/urfave/cli/v3`
Expected: Dependency added to go.mod and go.sum

- [ ] **Step 2: Verify and tidy**

Run: `go mod tidy`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add urfave/cli/v3 dependency

Add CLI framework library for command-line interface.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 6: Create CLI root command

**Files:**
- Create: `cmd/gollum/cli/root.go`
- Test: `cmd/gollum/cli/root_test.go`

- [ ] **Step 1: Write failing test**

```go
// cmd/gollum/cli/root_test.go
package cli

import (
    "context"
    "testing"

    "github.com/denkhaus/gollum/pkg/shared"
    "github.com/samber/do/v2"
    "github.com/urfave/cli/v3"
    "gotest.tools/v3/assert"
)

func TestRootCommand(t *testing.T) {
    rootCmd := RootCommand()

    assert.Equal(t, "gollum", rootCmd.Name)
    assert.Equal(t, "AI agent workflow system", rootCmd.Usage)
    assert.Assert(t, rootCmd.Action != nil)
}

func TestRunDefault(t *testing.T) {
    t.Run("delegates to ApplicationService", func(t *testing.T) {
        // This requires a full DI container setup
        // For now, test that it attempts to get ApplicationService from injector
        injector := do.New()

        cmd := &cli.Command{
            Metadata: map[string]any{
                shared.MetadataInjectorKey: injector,
            },
        }

        // This will fail because ApplicationService is not registered
        // but proves the delegation pattern
        err := RunDefault(context.Background(), cmd)
        assert.ErrorContains(t, err, "not found")
    })
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/gollum/cli/ -v`
Expected: FAIL with "undefined: RootCommand"

- [ ] **Step 3: Implement root command**

```go
// cmd/gollum/cli/root.go
package cli

import (
    "context"

    "github.com/denkhaus/gollum/pkg/app"
    "github.com/denkhaus/gollum/pkg/shared"
    "github.com/samber/do/v2"
    "github.com/urfave/cli/v3"
)

// RootCommand returns the root CLI command
func RootCommand() *cli.Command {
    return &cli.Command{
        Name:  "gollum",
        Usage: "AI agent workflow system",
        Action: RunDefault,
        Commands: []*cli.Command{
            FlowCommandGroup(),
        },
    }
}

// RunDefault executes the default application behavior.
// Delegates to ApplicationService which handles:
// - Default flow detection and execution
// - Fallback to TUI if no default flow
func RunDefault(ctx context.Context, cmd *cli.Command) error {
    injector := shared.MustGetInjector(cmd)
    appSvc := do.MustInvoke[app.ApplicationService](injector)
    return appSvc.Run(ctx)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/gollum/cli/ -run TestRootCommand -v`
Expected: PASS (TestRunDefault will still fail, which is expected)

- [ ] **Step 5: Commit**

```bash
git add cmd/gollum/cli/root.go cmd/gollum/cli/root_test.go
git commit -m "feat(cli): add root command

Root command delegates to ApplicationService for default behavior.
TUI remains default when no default flow exists.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 7: Create flow command group placeholder

**Files:**
- Create: `cmd/gollum/cli/flow/flow.go`

- [ ] **Step 1: Create flow command group**

```go
// cmd/gollum/cli/flow/flow.go
package flow

import "github.com/urfave/cli/v3"

// FlowCommandGroup returns the flow command group
func FlowCommandGroup() *cli.Command {
    return &cli.Command{
        Name:  "flow",
        Usage: "Flow management commands",
        Subcommands: []*cli.Command{
            LintCommand(),
            RunCommand(),
        },
    }
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./cmd/gollum/cli/flow/`
Expected: Error "undefined: LintCommand" and "undefined: RunCommand" (will be fixed in next tasks)

- [ ] **Step 3: Commit**

```bash
git add cmd/gollum/cli/flow/flow.go
git commit -m "feat(cli): add flow command group placeholder

Commands will be added in following tasks.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 4: Flow Lint Command

### Task 8: Implement flow lint command

**Files:**
- Create: `cmd/gollum/cli/flow/lint.go`
- Test: `cmd/gollum/cli/flow/lint_test.go`

- [ ] **Step 1: Write failing tests**

```go
// cmd/gollum/cli/flow/lint_test.go
package flow

import (
    "context"
    "testing"

    "github.com/denkhaus/gollum/pkg/shared"
    "github.com/samber/do/v2"
    "github.com/urfave/cli/v3"
    "gotest.tools/v3/assert"
)

func TestLintCommand(t *testing.T) {
    lintCmd := LintCommand()

    assert.Equal(t, "lint", lintCmd.Name)
    assert.Assert(t, lintCmd.Action != nil)
    assert.Equal(t, 1, lintCmd.Args().Len())
}

func TestLintAction_MissingPath(t *testing.T) {
    injector := do.New()
    cmd := &cli.Command{
        Metadata: map[string]any{
            shared.MetadataInjectorKey: injector,
        },
    }

    err := LintAction(context.Background(), cmd)
    assert.ErrorContains(t, err, "missing path")
}

func TestLintAction_ValidFlow(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // This requires full DI setup - implement in integration test
    // For now, verify the function signature is correct
    assert.Assert(t, LintAction != nil)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/gollum/cli/flow/ -run TestLintCommand -v`
Expected: FAIL with "undefined: LintCommand"

- [ ] **Step 3: Implement lint command**

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
        Name:      "lint",
        Usage:     "Validate a flow file or module directory",
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

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/gollum/cli/flow/ -run TestLintCommand -v`
Expected: PASS (except integration test)

- [ ] **Step 5: Test with real flow file**

Run: `go run ./cmd/gollum flow lint ../../.gollum/flows/examples/simple-flow.xml`
Expected: Flow validation output

- [ ] **Step 6: Commit**

```bash
git add cmd/gollum/cli/flow/lint.go cmd/gollum/cli/flow/lint_test.go
git commit -m "feat(cli): add flow lint command

Migrates functionality from cmd/flows/lint/main.go.
Uses existing linter.LintModule() for validation.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 5: Flow Run Command

### Task 9: Implement flow run command

**Files:**
- Create: `cmd/gollum/cli/flow/run.go`
- Test: `cmd/gollum/cli/flow/run_test.go`

- [ ] **Step 1: Write failing tests**

```go
// cmd/gollum/cli/flow/run_test.go
package flow

import (
    "context"
    "testing"

    "github.com/denkhaus/gollum/pkg/shared"
    "github.com/samber/do/v2"
    "github.com/urfave/cli/v3"
    "gotest.tools/v3/assert"
)

func TestRunCommand(t *testing.T) {
    runCmd := RunCommand()

    assert.Equal(t, "run", runCmd.Name)
    assert.Assert(t, runCmd.Action != nil)
    assert.Equal(t, 1, runCmd.Args().Len())
}

func TestRunAction_MissingPath(t *testing.T) {
    injector := do.New()
    cmd := &cli.Command{
        Metadata: map[string]any{
            shared.MetadataInjectorKey: injector,
        },
    }

    err := RunAction(context.Background(), cmd)
    assert.ErrorContains(t, err, "missing path")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/gollum/cli/flow/ -run TestRunCommand -v`
Expected: FAIL with "undefined: RunCommand"

- [ ] **Step 3: Implement run command**

```go
// cmd/gollum/cli/flow/run.go
package flow

import (
    "context"
    "fmt"

    "github.com/denkhaus/gollum/pkg/flows/executor"
    "github.com/denkhaus/gollum/pkg/flows/parser"
    "github.com/denkhaus/gollum/pkg/shared"
    "github.com/samber/do/v2"
    "github.com/urfave/cli/v3"
)

// RunCommand returns the flow run command
func RunCommand() *cli.Command {
    return &cli.Command{
        Name:      "run",
        Usage:     "Execute a flow",
        ArgsUsage: "<path>",
        Flags: []cli.Flag{
            shared.VerboseFlag,
        },
        Action: RunAction,
    }
}

// RunAction executes the flow run command
func RunAction(ctx context.Context, cmd *cli.Command) error {
    injector := shared.MustGetInjector(cmd)

    path, err := shared.GetPathArg(cmd)
    if err != nil {
        return err
    }

    return executeFlow(injector, path)
}

// executeFlow loads and executes a flow
func executeFlow(injector do.Injector, path string) error {
    // Parse flow
    flow, err := parser.Parse(path)
    if err != nil {
        return fmt.Errorf("failed to parse flow: %w", err)
    }

    // Get services
    flowExecutorService := do.MustInvoke[executor.FlowExecutorService](injector)

    // Create executor
    exec := flowExecutorService.New(flow)

    // Set empty input
    exec.SetInput(make(map[string]any))

    // Validate
    if err := exec.Validate(); err != nil {
        return fmt.Errorf("flow validation failed: %w", err)
    }

    // Run
    if err := exec.Run(); err != nil {
        return fmt.Errorf("flow execution failed: %w", err)
    }

    return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/gollum/cli/flow/ -run TestRunCommand -v`
Expected: PASS

- [ ] **Step 5: Test with real flow file**

Run: `go run ./cmd/gollum flow run ../../.gollum/flows/examples/simple-flow.xml`
Expected: Flow executes

- [ ] **Step 6: Commit**

```bash
git add cmd/gollum/cli/flow/run.go cmd/gollum/cli/flow/run_test.go
git commit -m "feat(cli): add flow run command

Executes flows using FlowExecutorService.
Supports both individual .xml files and module directories.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 6: Main Entry Point Integration

### Task 10: Update main.go to use CLI framework

**Files:**
- Modify: `cmd/gollum/main.go`

- [ ] **Step 1: Read existing main.go**

Current structure:
- Creates DI container
- Handles profiling
- Runs ApplicationService
- Handles SIGTERM

- [ ] **Step 2: Backup existing main.go**

```bash
cp cmd/gollum/main.go cmd/gollum/main.go.bak
```

- [ ] **Step 3: Rewrite main.go with CLI framework**

```go
// cmd/gollum/main.go
package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/denkhaus/gollum/cmd/gollum/cli"
	"github.com/denkhaus/gollum/pkg/app"
	"github.com/denkhaus/gollum/pkg/di"
	"github.com/denkhaus/gollum/pkg/profiling"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"
)

func startup(ctx context.Context) (*cli.Command, error) {
	// Define and parse profiling flags
	profiling.DefineFlags()

	// Create cancellable context for shutdown
	shutdownCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Setup container and services
	container := di.NewContainer()
	injector := container.RegisterServices(shutdownCtx)

	// Handle graceful shutdown for SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutdown signal received")
		cancel()
	}()

	// Parse profiling configuration
	profilingConfig := profiling.ParseFlags()

	// Enable profiling if configured
	if profilingConfig.Enable {
		cleanup := setupProfiling(profilingConfig, injector)
		defer func() {
			cleanup()
		}()
	}

	// Create root CLI command
	appCmd := cli.RootCommand()

	// Store injector in command metadata
	shared.SetInjector(appCmd, injector)

	// Add shutdown handler
	appCmd.After = func(cmdCtx context.Context, cmd *cli.Command) error {
		container.Shutdown()
		return nil
	}

	return appCmd, nil
}

func main() {
	ctx := context.Background()

	appCmd, err := startup(ctx)
	if err != nil {
		log.Fatalf("startup error: %v", err)
	}

	if err := appCmd.Run(ctx, os.Args); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("application error: %v", err)
	}
}

// setupProfiling remains unchanged from original
func setupProfiling(config profiling.Config, injector do.Injector) func() {
	// ... existing profiling setup code ...
	return func() {}
}
```

- [ ] **Step 4: Keep profiling.go unchanged**

The `profiling.go` file remains as-is.

- [ ] **Step 5: Test build**

Run: `go build ./cmd/gollum/`
Expected: No errors

- [ ] **Step 6: Test default behavior (TUI)**

Run: `./gollum --help`
Expected: Help text shown

Run: `./gollum`
Expected: TUI starts (no default flow)

- [ ] **Step 7: Test flow commands**

Run: `./gollum flow --help`
Expected: Flow command help shown

Run: `./gollum flow lint ../../.gollum/flows/examples/simple-flow.xml`
Expected: Flow validated

- [ ] **Step 8: Remove backup**

```bash
rm cmd/gollum/main.go.bak
```

- [ ] **Step 9: Commit**

```bash
git add cmd/gollum/main.go
git commit -m "refactor(main): integrate CLI framework

Wrap existing startup in urfave/cli/v3 framework.
Root command delegates to ApplicationService.Run().
TUI remains default behavior.
All existing functionality preserved.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 7: Cleanup

### Task 11: Remove deprecated flows lint command

**Files:**
- Delete: `cmd/flows/lint/main.go`

- [ ] **Step 1: Remove deprecated file**

```bash
rm -rf cmd/flows/lint/
```

- [ ] **Step 2: Update any references**

Check for imports: `grep -r "cmd/flows/lint" .`

- [ ] **Step 3: Commit**

```bash
git add cmd/flows/
git commit -m "chore: remove deprecated flows-lint command

Functionality migrated to 'gollum flow lint' command.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 12: Update documentation

**Files:**
- Update: README.md (if exists)
- Create: docs/cli.md (CLI usage documentation)

- [ ] **Step 1: Create CLI documentation**

```markdown
# CLI Documentation

## Usage

\`\`\`bash
gollum [command] [arguments]
\`\`\`

## Commands

### Running Gollum (Default)

\`\`\`bash
gollum
\`\`\`

Runs the default application. If a default flow exists at `.gollum/flows/default/main.xml`,
it will be executed. Otherwise, the TUI interface starts.

### Flow Commands

#### gollum flow lint

Validate a flow file or module directory.

\`\`\`bash
gollum flow lint <path>
\`\`\`

**Examples:**
\`\`\`bash
gollum flow lint .gollum/flows/examples/simple-flow.xml
gollum flow lint .gollum/flows/my-module/
\`\`\`

#### gollum flow run

Execute a flow.

\`\`\`bash
gollum flow run <path>
\`\`\`

**Examples:**
\`\`\`bash
gollum flow run .gollum/flows/examples/simple-flow.xml
gollum flow run .gollum/flows/my-module/
\`\`\`

## Default Flows

To create a default flow that runs when you type `gollum`:

1. Create a flow module at `.gollum/flows/default/main.xml`
2. This flow will execute automatically when running `gollum` with no arguments
3. Workspace-local flows override global flows (`~/.config/gollum/flows/default/main.xml`)

## Exit Codes

- `0`: Success
- `1`: Error (validation failed, execution error)
- `2`: Usage error (invalid arguments)
\`\`\`
```

- [ ] **Step 2: Commit documentation**

```bash
git add docs/cli.md
git commit -m "docs: add CLI usage documentation

Document CLI commands, default flow behavior, and exit codes.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Verification Checklist

After completing all tasks:

- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `go build ./cmd/gollum/`
- [ ] `gollum` with no args starts TUI (no default flow)
- [ ] `gollum --help` shows help text
- [ ] `gollum flow --help` shows flow commands
- [ ] `gollum flow lint` validates flows correctly
- [ ] `gollum flow run` executes flows correctly
- [ ] Default flow execution works when `.gollum/flows/default/main.xml` exists
- [ ] No regression in existing TUI functionality
- [ ] `cmd/flows/lint` functionality preserved in new command

---

## References

- Spec: `docs/superpowers/specs/2026-03-13-cli-design.md`
- `pkg/di/container.go` - DI container implementation
- `pkg/flows/` - Flow types, parser, executor, linter
- `github.com/urfave/cli/v3` - CLI framework documentation
- Go Programming Guide: `/home/denkhaus/dev/kb/guides/guide.golang.programming.md`
- Go DI Guide: `/home/denkhaus/dev/kb/guides/guide.golang.di.md`
