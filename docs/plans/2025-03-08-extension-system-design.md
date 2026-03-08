# Gollum Extension System Design

**Date:** 2025-03-08
**Status:** Design Phase
**Author:** Claude (Brainstorming Session)

---

## Executive Summary

This document outlines the design for a hybrid Go interpreter-based extension system for Gollum. The system enables:

1. **Func Steps**: Lightweight functions executed in flows via Scriggo VM (high performance)
2. **Extensions**: Full packages loaded via Yaegi with DI integration (power & flexibility)
3. **DI Gateway**: Simple injector exposure for service registration/consumption
4. **Directory Discovery**: Automatic loading from `~/.config/gollum/` and `<workspace>/.gollum/`

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         Gollum Core                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │   Agent      │  │    Flow      │  │    Tool      │          │
│  │   Registry   │  │   Executor   │  │   Manager    │          │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘          │
│         │                 │                 │                   │
│         └─────────────────┼─────────────────┘                   │
│                           ▼                                     │
│                  ┌──────────────────┐                           │
│                  │   DI Container   │                           │
│                  │  (samber/do/v2)  │                           │
│                  └────────┬─────────┘                           │
└───────────────────────────┼─────────────────────────────────────┘
                            │
           ┌────────────────┴────────────────┐
           │                                  │
           ▼                                  ▼
    ┌──────────────┐                  ┌──────────────┐
    │ DI Gateway   │                  │ DI Gateway   │
    │  (Scriggo)   │                  │   (Yaegi)    │
    │ Injector()   │                  │ Injector()   │
    └──────┬───────┘                  └──────┬───────┘
           │                                  │
           ▼                                  ▼
    ┌──────────────┐                  ┌──────────────┐
    │  Scriggo VM  │                  │   Yaegi      │
    │  Func Steps  │                  │  Extensions  │
    │  (compiled)  │                  │  (evaluated) │
    └──────────────┘                  └──────────────┘
           │                                  │
           ▼                                  ▼
    ┌──────────────┐                  ┌──────────────┐
    │ Func Registry│                  │ Ext Loader   │
    │ .gollum/func │                  │ .gollum/ext  │
    └──────────────┘                  └──────────────┘
```

---

## Component Design

### 1. DIGateway Service

**Purpose:** Expose the DI injector to extensions for service registration and consumption.

**Interface:** `pkg/extensions/gateway.go`

```go
package extensions

import (
    "github.com/samber/do/v2"
)

// DIGateway simply exposes the injector to extensions
type DIGateway interface {
    // Injector returns the raw DI injector for extensions to use
    Injector() do.Injector
}

// gatewayServiceImpl is the private implementation
type gatewayServiceImpl struct {
    injector do.Injector
}

var _ DIGateway = (*gatewayServiceImpl)(nil)

// NewGatewayService creates the DI gateway service
func NewGatewayService(injector do.Injector) (DIGateway, error) {
    return &gatewayServiceImpl{
        injector: injector,
    }, nil
}

func (p *gatewayServiceImpl) Injector() do.Injector {
    return p.injector
}
```

**Key Design Points:**
- Minimal interface: single method returning raw injector
- Extensions can use `do.Provide()` to register services
- Extensions can use `do.MustInvoke[T]()` to consume services
- Can be extended later with convenience methods (non-breaking)

---

### 2. ScriggoRunner Service

**Purpose:** Manage pre-compiled func steps for high-performance execution.

**Interface:** `pkg/extensions/scriggo.go`

```go
package extensions

import (
    "fmt"
    "github.com/open2b/scriggo"
)

// ScriggoRunner manages pre-compiled func steps
type ScriggoRunner interface {
    // LoadFunc compiles and caches a function
    LoadFunc(name, source string) error

    // ExecuteFunc runs a loaded function with arguments
    ExecuteFunc(name string, args map[string]any) (any, error)

    // ListFuncs returns available function names
    ListFuncs() []string
}

// scriggoRunnerImpl is the private implementation
type scriggoRunnerImpl struct {
    vm      *scriggo.VM
    funcs   map[string]*scriggo.CompiledFunc
    gateway DIGateway
}

var _ ScriggoRunner = (*scriggoRunnerImpl)(nil)

// NewScriggoRunner creates the Scriggo runner service
func NewScriggoRunner(injector do.Injector) (ScriggoRunner, error) {
    gateway := do.MustInvoke[DIGateway](injector)

    vm := scriggo.NewVM(scriggo.GlobalOptions)

    // Export injector and do functions to Scriggo
    vm.SetValue("injector", gateway.Injector())
    vm.SetValue("do_provide", do.Provide[any])
    vm.SetValue("do_must_invoke", do.MustInvoke[any])
    vm.SetValue("do_invoke", do.Invoke[any])

    return &scriggoRunnerImpl{
        vm:      vm,
        funcs:   make(map[string]*scriggo.CompiledFunc),
        gateway: gateway,
    }, nil
}
```

**Performance Characteristics:**
- Compile once on load, execute many times
- VM bytecode execution: ~13 allocations vs Yaegi's 283M
- ~10x faster than Yaegi for recursive operations

---

### 3. YaegiLoader Service

**Purpose:** Load and manage extension packages with full DI integration.

**Interface:** `pkg/extensions/yaegi.go`

```go
package extensions

import (
    "fmt"
    "path/filepath"

    "github.com/traefik/yaegi/interp"
    "github.com/traefik/yaegi/stdlib"
)

// YaegiLoader manages extension packages
type YaegiLoader interface {
    // LoadExtension loads an extension from directory
    LoadExtension(path string) (*Extension, error)

    // InitExtension calls the extension's init function
    InitExtension(ext *Extension) error

    // UnloadExtension cleans up extension resources
    UnloadExtension(ext *Extension) error

    // GetExtension retrieves a loaded extension by name
    GetExtension(name string) (*Extension, error)

    // ListExtensions returns all loaded extension names
    ListExtensions() []string
}

// Extension represents a loaded Yaegi extension
type Extension struct {
    Name        string
    Path        string
    Interpreter *interp.Interpreter
    InitFunc    func() error
    Hooks       map[string]interface{}
    State       ExtensionState
    Error       error
    LoadedAt    time.Time
}

// yaegiLoaderImpl is the private implementation
type yaegiLoaderImpl struct {
    gateway DIGateway
    exts    map[string]*Extension
}

var _ YaegiLoader = (*yaegiLoaderImpl)(nil)

// NewYaegiLoader creates the Yaegi loader service
func NewYaegiLoader(injector do.Injector) (YaegiLoader, error) {
    gateway := do.MustInvoke[DIGateway](injector)

    return &yaegiLoaderImpl{
        gateway: gateway,
        exts:    make(map[string]*Extension),
    }, nil
}

func (p *yaegiLoaderImpl) LoadExtension(path string) (*Extension, error) {
    i := interp.New(interp.Options{})
    i.Use(stdlib.Symbols)

    // Export the injector to the extension
    i.Use(map[string]map[string]any{
        "github.com/denkhaus/gollum/pkg/extensions": {
            "injector": p.gateway.Injector(),
        },
    })

    // Load main.go
    mainPath := filepath.Join(path, "main.go")
    _, err := i.EvalPath(mainPath)
    if err != nil {
        return nil, fmt.Errorf("extension load: %w", err)
    }

    // Extract Init function
    initFn, err := i.Export("Init")
    if err != nil {
        return nil, fmt.Errorf("missing Init function: %w", err)
    }

    name := filepath.Base(path)
    ext := &Extension{
        Name:        name,
        Path:        path,
        Interpreter: i,
        InitFunc:    initFn.(func() error),
        Hooks:       make(map[string]interface{}),
        State:       StateLoaded,
        LoadedAt:    time.Now(),
    }

    p.exts[name] = ext
    return ext, nil
}
```

---

### 4. ExtensionService

**Purpose:** Orchestrate extension and function discovery/loading.

**Interface:** `pkg/extensions/service.go`

```go
package extensions

import (
    "context"
    "os"
    "path/filepath"

    "github.com/denkhaus/gollum/pkg/logger"
    "github.com/denkhaus/gollum/pkg/workspace"
    "github.com/samber/do/v2"
)

// ExtensionService manages extension and function loading
type ExtensionService interface {
    // LoadAll discovers and loads all extensions and functions
    LoadAll(ctx context.Context) error

    // GetFuncRunner returns the Scriggo function runner
    GetFuncRunner() ScriggoRunner

    // GetExtension returns a loaded extension by name
    GetExtension(name string) (*Extension, error)

    // ListExtensions returns all loaded extension names
    ListExtensions() []string
}

// extensionServiceImpl is the private implementation
type extensionServiceImpl struct {
    logService    logger.LoggerService
    gateway       DIGateway
    yaegiLoader   YaegiLoader
    scriggoRunner ScriggoRunner
    workspaceDir  string
}

var _ ExtensionService = (*extensionServiceImpl)(nil)

// NewExtensionServiceWithWorkspace creates the extension service
func NewExtensionServiceWithWorkspace(injector do.Injector) (ExtensionService, error) {
    logService := do.MustInvoke[logger.LoggerService](injector)
    gateway := do.MustInvoke[DIGateway](injector)
    yaegiLoader := do.MustInvoke[YaegiLoader](injector)
    scriggoRunner := do.MustInvoke[ScriggoRunner](injector)
    workspaceService := do.MustInvoke[workspace.Service](injector)

    return &extensionServiceImpl{
        logService:    logService,
        gateway:       gateway,
        yaegiLoader:   yaegiLoader,
        scriggoRunner: scriggoRunner,
        workspaceDir:  workspaceService.GetCurrentWorkspace(),
    }, nil
}

func (p *extensionServiceImpl) LoadAll(ctx context.Context) error {
    p.logService.Info("Extension service: loading extensions and functions...")

    if err := p.loadFuncSteps(); err != nil {
        return err
    }

    if err := p.loadExtensions(); err != nil {
        return err
    }

    p.logService.Infof("Extension service: loaded %d functions, %d extensions",
        len(p.scriggoRunner.ListFuncs()),
        len(p.yaegiLoader.ListExtensions()),
    )

    return nil
}

func (p *extensionServiceImpl) getFunctionsDirs() []string {
    var dirs []string

    // Priority 1: Workspace-local (PRIMARY)
    if p.workspaceDir != "" {
        dirs = append(dirs, filepath.Join(p.workspaceDir, ".gollum", "functions"))
    }

    // Priority 2: User space (for testing/reusable)
    dirs = append(dirs, filepath.Join(getGlobalGollumDir(), "functions"))

    return dirs
}

func (p *extensionServiceImpl) getExtensionsDirs() []string {
    var dirs []string

    // Priority 1: Workspace-local (PRIMARY)
    if p.workspaceDir != "" {
        dirs = append(dirs, filepath.Join(p.workspaceDir, ".gollum", "extensions"))
    }

    // Priority 2: User space (for testing/reusable)
    dirs = append(dirs, filepath.Join(getGlobalGollumDir(), "extensions"))

    return dirs
}

func getGlobalGollumDir() string {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return ""
    }
    return filepath.Join(homeDir, ".config", "gollum")
}
```

---

### 5. HookRegistry Service

**Purpose:** Manage lifecycle hooks for extensions.

**Interface:** `pkg/extensions/hooks.go`

```go
package extensions

import (
    "fmt"
    "time"
)

// HookType defines the hook point
type HookType string

const (
    HookAgentPreExecute  HookType = "agent_pre_execute"
    HookAgentPostExecute HookType = "agent_post_execute"
    HookFlowPreExecute   HookType = "flow_pre_execute"
    HookFlowPostExecute  HookType = "flow_post_execute"
    HookToolPreExecute   HookType = "tool_pre_execute"
    HookToolPostExecute  HookType = "tool_post_execute"
)

// HookContext provides context to hook functions
type HookContext struct {
    Type      HookType
    AgentID   string
    Timestamp time.Time
    Metadata  map[string]any
}

// HookFunction is the signature for hook callbacks
type HookFunction func(ctx *HookContext) error

// HookRegistry manages registered hooks
type HookRegistry interface {
    Register(hookType HookType, name string, fn HookFunction) error
    Unregister(hookType HookType, name string) error
    Execute(hookType HookType, ctx *HookContext) error
}

// hookRegistryImpl is the private implementation
type hookRegistryImpl struct {
    hooks map[HookType]map[string]HookFunction
}

var _ HookRegistry = (*hookRegistryImpl)(nil)

func NewHookRegistry(injector do.Injector) (HookRegistry, error) {
    return &hookRegistryImpl{
        hooks: make(map[HookType]map[string]HookFunction),
    }, nil
}

func (p *hookRegistryImpl) Register(hookType HookType, name string, fn HookFunction) error {
    if p.hooks[hookType] == nil {
        p.hooks[hookType] = make(map[string]HookFunction)
    }
    p.hooks[hookType][name] = fn
    return nil
}

func (p *hookRegistryImpl) Unregister(hookType HookType, name string) error {
    if p.hooks[hookType] != nil {
        delete(p.hooks[hookType], name)
    }
    return nil
}

func (p *hookRegistryImpl) Execute(hookType HookType, ctx *HookContext) error {
    for name, fn := range p.hooks[hookType] {
        if err := fn(ctx); err != nil {
            return fmt.Errorf("hook %s failed: %w", name, err)
        }
    }
    return nil
}
```

---

## Directory Structure

### File Layout

```
~/.config/gollum/                    # Global (user-level)
├── extensions/
│   └── notification/                # Example: reusable extension
│       └── main.go
└── functions/
    └── validate_email.go            # Example: common validation

<workspace>/.gollum/                # Workspace-local (PRIMARY)
├── extensions/
│   ├── project-hooks/               # Main location for project extensions
│   │   └── main.go
│   └── custom-processor/
│       └── main.go
└── functions/
    ├── transform_data.go            # Main location for project func steps
    └── format_output.go
```

### Priority Order

Workspace-local definitions **override** user space:
1. `<workspace>/.gollum/functions` (highest priority)
2. `~/.config/gollum/functions` (fallback, for reusable/testing)
3. Same pattern for extensions

---

## Extension API

### Func Step Template

```go
// ~/.config/gollum/functions/transform_data.go

package main

import (
    "encoding/json"
    "fmt"
)

// TransformData transforms input data
// Available built-ins: injector, do_provide, do_must_invoke, do_invoke
func TransformData(input string, rules map[string]string) (string, error) {
    result := input

    for key, value := range rules {
        result = fmt.Sprintf(result, key, value)
    }

    return result, nil
}
```

### Extension Template

```go
// ~/.config/gollum/extensions/database/main.go

package main

import (
    "fmt"
    "database/sql"

    "github.com/samber/do/v2"
)

var injector do.Injector

// DatabaseService interface
type DatabaseService interface {
    Query(query string, args ...any) (*sql.Rows, error)
    Exec(query string, args ...any) (sql.Result, error)
    Close() error
}

// myDatabaseService implements DatabaseService
type myDatabaseService struct {
    db *sql.DB
}

func NewDatabaseService(injector do.Injector) (DatabaseService, error) {
    db, err := sql.Open("postgres", "connection-string")
    if err != nil {
        return nil, err
    }
    return &myDatabaseService{db: db}, nil
}

func (p *myDatabaseService) Query(query string, args ...any) (*sql.Rows, error) {
    return p.db.Query(query, args...)
}

func (p *myDatabaseService) Exec(query string, args ...any) (sql.Result, error) {
    return p.db.Exec(query, args...)
}

func (p *myDatabaseService) Close() error {
    return p.db.Close()
}

// Init is required for all extensions
func Init() error {
    fmt.Println("Database extension: registering service...")

    // Register service directly with DI container
    do.Provide(injector, NewDatabaseService)

    return nil
}
```

---

## Flow Integration

### Func Step Execution

```xml
<flow name="process-data" version="1.0">
    <input>
        <string name="timestamp" required="true"/>
        <string name="format" default="2006-01-02"/>
    </input>

    <output>
        <string name="formatted_date"/>
    </output>

    <states>
        <state name="process" initial="true">
            <steps>
                <step type="func" function="format_date">
                    <params>
                        <param name="ts" value="${input.timestamp}"/>
                        <param name="layout" value="${input.format}"/>
                    </params>
                    <output assign="${output.formatted_date}"/>
                </step>
            </steps>
        </state>
    </states>
</flow>
```

### Executor Integration

```go
// pkg/flows/executor/executor.go

type Executor struct {
    // ... existing fields ...
    extService extensions.ExtensionService
}

func (e *Executor) executeFuncStep(step *flows.Step, stateName string) error {
    funcRunner := e.extService.GetFuncRunner()

    // Build args with template substitution
    args := make(map[string]any)
    for _, param := range step.Params {
        value := e.substituteTemplate(param.Value)
        args[param.Name] = value
    }

    // Execute via Scriggo runner
    result, err := funcRunner.ExecuteFunc(step.Function, args)
    if err != nil {
        return &FuncError{
            Function: step.Function,
            Step:     stateName,
            Err:      err,
        }
    }

    // Map result to output
    if step.Output != nil && step.Output.Assign != "" {
        fieldName := extractFieldName(step.Output.Assign)
        e.ctx.SetOutputField(fieldName, result)
    }

    return nil
}
```

---

## Error Handling

### Error Types

```go
// ExtensionError represents errors during extension operations
type ExtensionError struct {
    Extension string
    Operation string // "load", "init", "exec", "unload"
    Err       error
}

func (p *ExtensionError) Error() string {
    return fmt.Sprintf("extension %s: %s failed: %v", p.Extension, p.Operation, p.Err)
}

func (p *ExtensionError) Unwrap() error {
    return p.Err
}

// FuncError represents errors during func step execution
type FuncError struct {
    Function string
    Step     string
    Err      error
}

func (p *FuncError) Error() string {
    return fmt.Sprintf("func step '%s' in step '%s': %v", p.Function, p.Step, p.Err)
}

func (p *FuncError) Unwrap() error {
    return p.Err
}
```

### Extension Lifecycle States

```go
type ExtensionState int

const (
    StateLoaded       ExtensionState = iota
    StateInitializing
    StateReady
    StateFailed
    StateUnloading
    StateUnloaded
)
```

---

## Performance Benchmarks

### Interpreter Comparison

| Benchmark | Scriggo | Yaegi | Ratio |
|-----------|---------|-------|-------|
| Fibonacci (35) | 2.5s | 25.2s | **10x faster** |
| Allocations (Fib) | 41 KB | 12.5 GB | **300,000x less** |
| Allocs (Fib) | 13 | 283M | **22Mx fewer** |
| Closures | 30ms | 78ms | **2.6x faster** |

**Why Scriggo for Func Steps:**
- VM-based compilation to bytecode
- Extremely memory-efficient
- Pre-compile once, execute many times

**Why Yaegi for Extensions:**
- Can use pre-compiled Go types directly
- Simpler DI integration via `i.Use()`
- Production-proven (Traefik)

---

## Implementation Checklist

### Phase 1: Core Infrastructure
- [ ] Create `pkg/extensions` package
- [ ] Implement `DIGateway` service
- [ ] Implement `ScriggoRunner` service
- [ ] Implement `YaegiLoader` service
- [ ] Implement `HookRegistry` service
- [ ] Implement `ExtensionService` with directory discovery
- [ ] Register all services in `pkg/di/container.go`

### Phase 2: Flow Executor Integration
- [ ] Add `extService` field to `Executor`
- [ ] Update `executeFuncStep` to use `ScriggoRunner`
- [ ] Implement template substitution for func parameters
- [ ] Add executor tests with func steps

### Phase 3: Extension System
- [ ] Create directory structure helpers
- [ ] Implement extension loading from both directories
- [ ] Implement function loading from both directories
- [ ] Implement workspace overrides global
- [ ] Add extension lifecycle management

### Phase 4: Security & Limits
- [ ] Implement execution timeouts
- [ ] Implement memory limits
- [ ] Add audit logging
- [ ] Add extension validation

### Phase 5: Documentation & Examples
- [ ] Create extension template
- [ ] Create func step examples
- [ ] Write extension development guide
- [ ] Document DI integration patterns

---

## Dependencies

```go
require (
    github.com/open2b/scriggo v0.61.0  // For func steps
    github.com/traefik/yaegi          v1.12.2  // For extensions
    github.com/samber/do/v2            v2.22.0  // DI framework
)
```

---

## Future Enhancements

The DIGateway interface can be extended non-breaking:

```go
type DIGateway interface {
    // Core method (keep forever)
    Injector() do.Injector

    // Future convenience methods (additive)
    RegisterHook(hookType HookType, name string, fn HookFunction) error
    GetService[T any](name string) (T, error)
    // etc...
}
```

---

## References

- [Scriggo Documentation](https://scriggo.com/)
- [Yaegi Documentation](https://github.com/traefik/yaegi)
- [samber/do Documentation](https://github.com/samber/do)
- [Gollum DI Guide](/home/denkhaus/dev/kb/guides/guide.golang.di.md)
