# Executor Hooks Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Integrate the comprehensive hook system (`pkg/hooks`) into the Flow Executor, replacing unimplemented hook definitions.

**Architecture:** Add Executor category to existing `pkg/hooks` with type-safe `ExecutorPayload`, extend `HookManager` with executor methods, integrate into Flow Executor via DI injection.

**Tech Stack:** Go, generics, DI (samber/do), existing pkg/hooks infrastructure

---

## Task 1: ExecutorPayload Structure

**Files:**
- Create: `pkg/hooks/executor_hooks.go`
- Test: `pkg/hooks/executor_hooks_test.go`

**Step 1: Write the failing test**

Create `pkg/hooks/executor_hooks_test.go`:

```go
package hooks

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestExecutorPayload_FlowStepContext(t *testing.T) {
	flowID := uuid.New()
	sessionID := uuid.New()

	payload := &ExecutorPayload{
		FlowID:       flowID,
		FlowName:     "test-flow",
		SessionID:    sessionID,
		CurrentState: "process",
		StepType:     "llm",
		StepIndex:    0,
		StateName:    "process",
		StepResult:   map[string]any{"output": "test"},
		StepError:    nil,
		Duration:     100 * time.Millisecond,
	}

	assert.Equal(t, flowID, payload.FlowID)
	assert.Equal(t, "test-flow", payload.FlowName)
	assert.Equal(t, sessionID, payload.SessionID)
	assert.Equal(t, "process", payload.CurrentState)
	assert.Equal(t, "llm", payload.StepType)
	assert.Equal(t, 0, payload.StepIndex)
	assert.Equal(t, "process", payload.StateName)
	assert.NotNil(t, payload.StepResult)
	assert.NoError(t, payload.StepError)
	assert.Equal(t, 100*time.Millisecond, payload.Duration)
}

func TestExecutorPayload_WithStepError(t *testing.T) {
	payload := &ExecutorPayload{
		FlowID:       uuid.New(),
		SessionID:    uuid.New(),
		CurrentState: "failed",
		StepType:     "shell",
		StateName:    "failed",
		StepError:    assert.AnError,
		Duration:     50 * time.Millisecond,
	}

	assert.Error(t, payload.StepError)
	assert.Equal(t, 50*time.Millisecond, payload.Duration)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/hooks/executor_hooks_test.go -v`
Expected: FAIL with "undefined: ExecutorPayload"

**Step 3: Write minimal implementation**

Create `pkg/hooks/executor_hooks.go`:

```go
package hooks

import (
	"time"

	"github.com/google/uuid"
)

// ExecutorPayload contains flow execution context for step hooks
type ExecutorPayload struct {
	// Flow identification
	FlowID    uuid.UUID
	FlowName  string
	SessionID uuid.UUID

	// Current execution state
	CurrentState string

	// Step information
	StepType  string // "llm", "shell", "func", "mcp"
	StepIndex int
	StateName string

	// Execution metadata (AfterFlowStep only)
	StepResult map[string]any
	StepError  error
	Duration   time.Duration
}

// Hook points for executor
const (
	// BeforeFlowStep is triggered before any step in a flow executes
	BeforeFlowStep HookPoint = "BeforeFlowStep"

	// AfterFlowStep is triggered after any step in a flow completes
	AfterFlowStep HookPoint = "AfterFlowStep"
)
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/hooks/... -run TestExecutorPayload -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/hooks/executor_hooks.go pkg/hooks/executor_hooks_test.go
git commit -m "feat(hooks): add ExecutorPayload and flow step hook points

- Add ExecutorPayload struct with flow context and step info
- Define BeforeFlowStep and AfterFlowStep hook points
- Add basic tests for payload structure"
```

---

## Task 2: HookManager Extension - Registry

**Files:**
- Modify: `pkg/hooks/manager.go:177-215`
- Modify: `pkg/hooks/manager.go:15-43`
- Test: `pkg/hooks/executor_hooks_test.go`

**Step 1: Write the failing test**

Add to `pkg/hooks/executor_hooks_test.go`:

```go
func TestHookManager_RegisterExecutorHook(t *testing.T) {
	injector := setupTestInjector()
	hm, err := NewHookManager(injector)
	assert.NoError(t, err)

	called := false
	hookFn := func(ctx *TypedHookContext[ExecutorPayload]) error {
		called = true
		return nil
	}

	meta := TypedHookMetadata{
		Name:     "test-executor-hook",
		Point:    BeforeFlowStep,
		Priority: 50,
	}

	err = hm.RegisterExecutorHook(hookFn, meta)
	assert.NoError(t, err)
	assert.True(t, called, "Hook registration should succeed")
}

func TestHookManager_RegisterExecutorHook_DuplicateName(t *testing.T) {
	injector := setupTestInjector()
	hm, _ := NewHookManager(injector)

	hookFn := func(ctx *TypedHookContext[ExecutorPayload]) error {
		return nil
	}

	meta := TypedHookMetadata{
		Name:  "duplicate-hook",
		Point: BeforeFlowStep,
	}

	// First registration should succeed
	err := hm.RegisterExecutorHook(hookFn, meta)
	assert.NoError(t, err)

	// Second registration with same name should fail
	err = hm.RegisterExecutorHook(hookFn, meta)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/hooks/... -run TestHookManager_RegisterExecutorHook -v`
Expected: FAIL with "RegisterExecutorHook undefined"

**Step 3: Write minimal implementation**

Modify `pkg/hooks/manager.go` - Add to HookManager interface (after line 43):

```go
type HookManager interface {
	// ... existing methods ...

	// RegisterExecutorHook registers a typed hook for flow step execution events.
	// Valid hook points: BeforeFlowStep, AfterFlowStep.
	RegisterExecutorHook(fn TypedHookFunc[ExecutorPayload], meta TypedHookMetadata) error
```

Modify `pkg/hooks/manager.go` - Add to hookManagerImpl struct (after line 189):

```go
type hookManagerImpl struct {
	log   logger.LoggerService
	mu    sync.RWMutex
	names map[string]struct{}

	// Typed registries for type-safe hook storage
	toolRegistry    *TypedRegistry[ToolPayload]
	llmRegistry     *TypedRegistry[LLMPayload]
	fileRegistry    *TypedRegistry[FilePayload]
	sessionRegistry *TypedRegistry[SessionPayload]
	agentRegistry   *TypedRegistry[AgentPayload]
	skillRegistry   *TypedRegistry[SkillPayload]
	executorRegistry *TypedRegistry[ExecutorPayload] // NEW
}
```

Modify `pkg/hooks/manager.go` - Update NewHookManager (after line 212):

```go
func NewHookManager(injector do.Injector) (HookManager, error) {
	log := do.MustInvoke[logger.LoggerService](injector)

	log.Debug("HookManager starting")

	p := &hookManagerImpl{
		log:   log,
		names: make(map[string]struct{}),
	}

	// Initialize typed registries for type-safe hook storage
	p.toolRegistry = NewTypedRegistry[ToolPayload]()
	p.llmRegistry = NewTypedRegistry[LLMPayload]()
	p.fileRegistry = NewTypedRegistry[FilePayload]()
	p.sessionRegistry = NewTypedRegistry[SessionPayload]()
	p.agentRegistry = NewTypedRegistry[AgentPayload]()
	p.skillRegistry = NewTypedRegistry[SkillPayload]()
	p.executorRegistry = NewTypedRegistry[ExecutorPayload]() // NEW

	return p, nil
}
```

Modify `pkg/hooks/manager.go` - Add RegisterExecutorHook method (after RegisterSkillHook):

```go
// RegisterExecutorHook registers a typed hook for flow step execution events.
func (p *hookManagerImpl) RegisterExecutorHook(fn TypedHookFunc[ExecutorPayload], meta TypedHookMetadata) error {
	if fn == nil {
		return errs.Validation("hook function cannot be nil")
	}
	if meta.Name == "" {
		return errs.Validation("hook name cannot be empty")
	}
	if err := p.validateExecutorHookPoint(meta.Point); err != nil {
		return err
	}

	// Check for global name uniqueness
	p.mu.Lock()
	if _, exists := p.names[meta.Name]; exists {
		p.mu.Unlock()
		return errs.Conflictf("hook with name '%s' is already registered globally", meta.Name)
	}

	if err := p.executorRegistry.Add(fn, meta); err != nil {
		p.mu.Unlock()
		if err == ErrDuplicateHook {
			return errs.Conflictf("hook with name '%s' already registered", meta.Name)
		}
		return err
	}

	p.names[meta.Name] = struct{}{}
	p.mu.Unlock()

	p.log.Debug("Typed hook registered",
		zap.String("name", meta.Name),
		zap.String("point", meta.Point.String()),
		zap.Int("priority", meta.Priority))

	return nil
}
```

Add validation method (after validateSkillHookPoint):

```go
func (p *hookManagerImpl) validateExecutorHookPoint(point HookPoint) error {
	switch point {
	case BeforeFlowStep, AfterFlowStep:
		return nil
	default:
		return errs.Validationf("invalid executor hook point: %s (expected BeforeFlowStep or AfterFlowStep)", point)
	}
}
```

Update UnregisterHook to include executor registry (after line 241):

```go
func (p *hookManagerImpl) UnregisterHook(name string) bool {
	unregistered := false

	// Check typed registries
	switch {
	case p.toolRegistry.Remove(name):
		p.log.Debug("Hook unregistered from tool registry", zap.String("name", name))
		unregistered = true
	case p.llmRegistry.Remove(name):
		p.log.Debug("Hook unregistered from LLM registry", zap.String("name", name))
		unregistered = true
	case p.fileRegistry.Remove(name):
		p.log.Debug("Hook unregistered from file registry", zap.String("name", name))
		unregistered = true
	case p.sessionRegistry.Remove(name):
		p.log.Debug("Hook unregistered from session registry", zap.String("name", name))
		unregistered = true
	case p.agentRegistry.Remove(name):
		p.log.Debug("Hook unregistered from agent registry", zap.String("name", name))
		unregistered = true
	case p.skillRegistry.Remove(name):
		p.log.Debug("Hook unregistered from skill registry", zap.String("name", name))
		unregistered = true
	case p.executorRegistry.Remove(name): // NEW
		p.log.Debug("Hook unregistered from executor registry", zap.String("name", name))
		unregistered = true
	}

	// Remove from global names map if hook was found and removed
	if unregistered {
		p.mu.Lock()
		delete(p.names, name)
		p.mu.Unlock()
	}

	return unregistered
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/hooks/... -run TestHookManager_RegisterExecutorHook -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/hooks/manager.go pkg/hooks/executor_hooks_test.go
git commit -m "feat(hooks): add RegisterExecutorHook to HookManager

- Add executorRegistry to hookManagerImpl
- Implement RegisterExecutorHook with validation
- Update UnregisterHook to include executor registry
- Add tests for registration and duplicate name handling"
```

---

## Task 3: TriggerExecutorHooks Method

**Files:**
- Modify: `pkg/hooks/manager.go:48-76`
- Test: `pkg/hooks/executor_hooks_test.go`

**Step 1: Write the failing test**

Add to `pkg/hooks/executor_hooks_test.go`:

```go
func TestHookManager_TriggerExecutorHooks_BeforeFlowStep(t *testing.T) {
	injector := setupTestInjector()
	hm, _ := NewHookManager(injector)

	executed := false
	var capturedPayload *ExecutorPayload

	hookFn := func(ctx *TypedHookContext[ExecutorPayload]) error {
		executed = true
		capturedPayload = ctx.Payload
		return nil
	}

	hm.RegisterExecutorHook(hookFn, TypedHookMetadata{
		Name:  "test-trigger",
		Point: BeforeFlowStep,
	})

	payload := &ExecutorPayload{
		FlowID:       uuid.New(),
		FlowName:     "trigger-test",
		SessionID:    uuid.New(),
		CurrentState: "initial",
		StepType:     "func",
		StateName:    "initial",
	}

	hookCtx := &TypedHookContext[ExecutorPayload]{
		SessionID: payload.SessionID,
		Payload:   payload,
	}

	result := hm.TriggerExecutorHooks(context.Background(), BeforeFlowStep, hookCtx)

	assert.True(t, executed)
	assert.NotNil(t, capturedPayload)
	assert.Equal(t, "trigger-test", capturedPayload.FlowName)
	assert.Equal(t, "func", capturedPayload.StepType)
	assert.False(t, result.Blocked)
	assert.Nil(t, result.Error)
}

func TestHookManager_TriggerExecutorHooks_AfterFlowStep(t *testing.T) {
	injector := setupTestInjector()
	hm, _ := NewHookManager(injector)

	hookFn := func(ctx *TypedHookContext[ExecutorPayload]) error {
		// Simulate hook that blocks execution
		ctx.Blocked = true
		return nil
	}

	hm.RegisterExecutorHook(hookFn, TypedHookMetadata{
		Name:     "blocking-hook",
		Point:    BeforeFlowStep,
		Priority: 100,
	})

	payload := &ExecutorPayload{
		FlowID:    uuid.New(),
		StepType:  "shell",
		StateName: "blocked",
	}

	hookCtx := &TypedHookContext[ExecutorPayload]{
		Payload: payload,
	}

	result := hm.TriggerExecutorHooks(context.Background(), BeforeFlowStep, hookCtx)

	assert.True(t, result.Blocked, "Hook should block execution")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/hooks/... -run TestHookManager_TriggerExecutorHooks -v`
Expected: FAIL with "TriggerExecutorHooks undefined"

**Step 3: Write minimal implementation**

Add to HookManager interface (after RegisterExecutorHook):

```go
// TriggerExecutorHooks executes all typed executor hooks for a given hook point.
// Valid hook points: BeforeFlowStep, AfterFlowStep.
TriggerExecutorHooks(ctx context.Context, point HookPoint, hookCtx *TypedHookContext[ExecutorPayload]) TypedHookResult[ExecutorPayload]
```

Add implementation after TriggerSkillHooks:

```go
// TriggerExecutorHooks executes all typed executor hooks for a given hook point.
func (p *hookManagerImpl) TriggerExecutorHooks(ctx context.Context, point HookPoint, hookCtx *TypedHookContext[ExecutorPayload]) TypedHookResult[ExecutorPayload] {
	return p.executorRegistry.Trigger(ctx, point, hookCtx)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/hooks/... -run TestHookManager_TriggerExecutorHooks -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/hooks/manager.go pkg/hooks/executor_hooks_test.go
git commit -m "feat(hooks): add TriggerExecutorHooks method

- Implement TriggerExecutorHooks using executorRegistry
- Add tests for BeforeFlowStep and AfterFlowStep triggering
- Test hook blocking behavior"
```

---

## Task 4: WithFlowStepHooks Wrapper Method

**Files:**
- Modify: `pkg/hooks/manager.go:77-103`
- Test: `pkg/hooks/executor_hooks_test.go`

**Step 1: Write the failing test**

Add to `pkg/hooks/executor_hooks_test.go`:

```go
func TestHookManager_WithFlowStepHooks_BeforeAndAfter(t *testing.T) {
	injector := setupTestInjector()
	hm, _ := NewHookManager(injector)

	var beforePayload, afterPayload *ExecutorPayload

	// Before hook
	hm.RegisterExecutorHook(func(ctx *TypedHookContext[ExecutorPayload]) error {
		beforePayload = ctx.Payload
		return nil
	}, TypedHookMetadata{Name: "before", Point: BeforeFlowStep})

	// After hook
	hm.RegisterExecutorHook(func(ctx *TypedHookContext[ExecutorPayload]) error {
		afterPayload = ctx.Payload
		return nil
	}, TypedHookMetadata{Name: "after", Point: AfterFlowStep})

	flowID := uuid.New()
	sessionID := uuid.New()

	step := &mockStep{Type: "llm"}

	result, err := hm.WithFlowStepHooks(
		context.Background(),
		sessionID,
		flowID,
		step,
		"test-state",
		func() (map[string]any, error) {
			return map[string]any{"response": "test"}, nil
		},
	)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test", result["response"])

	// Verify before hook was called
	assert.NotNil(t, beforePayload)
	assert.Equal(t, "llm", beforePayload.StepType)
	assert.Equal(t, "test-state", beforePayload.StateName)
	assert.Nil(t, beforePayload.StepResult) // Before hook has no result

	// Verify after hook was called
	assert.NotNil(t, afterPayload)
	assert.Equal(t, "llm", afterPayload.StepType)
	assert.NotNil(t, afterPayload.StepResult)
	assert.Equal(t, "test", afterPayload.StepResult["response"])
	assert.Greater(t, afterPayload.Duration, time.Duration(0))
}

func TestHookManager_WithFlowStepHooks_ExecutionBlocked(t *testing.T) {
	injector := setupTestInjector()
	hm, _ := NewHookManager(injector)

	// Blocking before hook
	hm.RegisterExecutorHook(func(ctx *TypedHookContext[ExecutorPayload]) error {
		ctx.Blocked = true
		return nil
	}, TypedHookMetadata{Name: "blocker", Point: BeforeFlowStep})

	workCalled := false
	_, err := hm.WithFlowStepHooks(
		context.Background(),
		uuid.New(),
		uuid.New(),
		&mockStep{Type: "func"},
		"blocked-state",
		func() (map[string]any, error) {
			workCalled = true
			return nil, nil
		},
	)

	assert.Error(t, err)
	assert.False(t, workCalled, "Work should not be called when blocked")
	assert.Contains(t, err.Error(), "blocked by BeforeFlowStep hook")
}

func TestHookManager_WithFlowStepHooks_WorkError(t *testing.T) {
	injector := setupTestInjector()
	hm, _ := NewHookManager(injector)

	var afterPayload *ExecutorPayload
	hm.RegisterExecutorHook(func(ctx *TypedHookContext[ExecutorPayload]) error {
		afterPayload = ctx.Payload
		return nil
	}, TypedHookMetadata{Name: "after", Point: AfterFlowStep})

	workErr := errors.New("work failed")

	_, err := hm.WithFlowStepHooks(
		context.Background(),
		uuid.New(),
		uuid.New(),
		&mockStep{Type: "shell"},
		"error-state",
		func() (map[string]any, error) {
			return nil, workErr
		},
	)

	assert.Error(t, err)
	assert.Equal(t, workErr, err)

	// After hook should still be called even on error
	assert.NotNil(t, afterPayload)
	assert.Error(t, afterPayload.StepError)
	assert.Equal(t, workErr, afterPayload.StepError)
}

// Helper mock step
type mockStep struct {
	Type string
}

func (m *mockStep) GetStepType() string {
	return m.Type
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/hooks/... -run TestHookManager_WithFlowStepHooks -v`
Expected: FAIL with "WithFlowStepHooks undefined"

**Step 3: Write minimal implementation**

First, we need to update the interface to accept the step interface. Add to types.go or create a step interface:

Modify `pkg/hooks/manager.go` - Add to HookManager interface:

```go
// WithFlowStepHooks wraps a function with flow step execution hooks.
// BeforeFlowStep hooks can inspect/validate before execution.
// AfterFlowStep hooks can log/audit after execution.
WithFlowStepHooks(
	ctx context.Context,
	sessionID, flowID uuid.UUID,
	step any, // *flows.Step - using any to avoid import cycle
	stateName string,
	work func() (map[string]any, error),
) (map[string]any, error)
```

Add implementation:

```go
// WithFlowStepHooks wraps a function with flow step execution hooks.
func (p *hookManagerImpl) WithFlowStepHooks(
	ctx context.Context,
	sessionID, flowID uuid.UUID,
	step any,
	stateName string,
	work func() (map[string]any, error),
) (map[string]any, error) {
	startTime := time.Now()

	// Extract step type using reflection or interface
	stepType := "unknown"
	if s, ok := step.(*flows.Step); ok {
		stepType = s.Type
	}

	// Create initial payload for BeforeFlowStep
	beforePayload := &ExecutorPayload{
		FlowID:       flowID,
		SessionID:    sessionID,
		CurrentState: stateName,
		StepType:     stepType,
		StateName:    stateName,
	}

	beforeCtx := &TypedHookContext[ExecutorPayload]{
		SessionID: sessionID,
		AgentID:   uuid.Nil,
		Payload:   beforePayload,
	}

	// Execute BeforeFlowStep hooks
	beforeResult := p.executorRegistry.Trigger(ctx, BeforeFlowStep, beforeCtx)
	if beforeResult.Blocked {
		return nil, errs.Validation("step blocked by BeforeFlowStep hook")
	}

	// Execute the actual step
	result, err := work()
	duration := time.Since(startTime)

	// Create payload for AfterFlowStep with execution results
	afterPayload := &ExecutorPayload{
		FlowID:       flowID,
		SessionID:    sessionID,
		CurrentState: stateName,
		StepType:     stepType,
		StateName:    stateName,
		StepResult:   result,
		StepError:    err,
		Duration:     duration,
	}

	afterCtx := &TypedHookContext[ExecutorPayload]{
		SessionID: sessionID,
		AgentID:   uuid.Nil,
		Payload:   afterPayload,
	}

	// Execute AfterFlowStep hooks (always, even on error)
	p.executorRegistry.Trigger(ctx, AfterFlowStep, afterCtx)

	return result, err
}
```

**Wait - we have an import cycle issue**. We need to use `any` for step and extract type. Let's update the test helper and add a simple step extractor.

Actually, let me simplify - we'll pass stepType directly instead of the whole step object:

**Alternative simpler approach** - Modify signature to accept stepType directly:

```go
// WithFlowStepHooks wraps a function with flow step execution hooks.
WithFlowStepHooks(
	ctx context.Context,
	sessionID, flowID uuid.UUID,
	stepType, stateName string,
	work func() (map[string]any, error),
) (map[string]any, error)
```

And update tests accordingly.

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/hooks/... -run TestHookManager_WithFlowStepHooks -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/hooks/manager.go pkg/hooks/executor_hooks_test.go
git commit -m "feat(hooks): add WithFlowStepHooks wrapper method

- Implement WithFlowStepHooks for before/after step execution
- Support blocking via BeforeFlowStep hooks
- Always execute AfterFlowStep hooks even on work error
- Include duration tracking in AfterFlowStep payload"
```

---

## Task 5: NoOpHookManager Executor Methods

**Files:**
- Modify: `pkg/hooks/types.go:112-219`

**Step 1: Write the failing test**

```go
func TestNoOpHookManager_ExecutorMethods(t *testing.T) {
	noOp := NewNoOpHookManager()

	// Should not panic
	meta := TypedHookMetadata{Name: "test", Point: BeforeFlowStep}
	err := noOp.RegisterExecutorHook(nil, meta)
	assert.NoError(t, err)

	payload := &ExecutorPayload{FlowID: uuid.New()}
	ctx := &TypedHookContext[ExecutorPayload]{Payload: payload}

	result := noOp.TriggerExecutorHooks(context.Background(), BeforeFlowStep, ctx)
	assert.False(t, result.Blocked)

	result, err = noOp.WithFlowStepHooks(
		context.Background(),
		uuid.New(),
		uuid.New(),
		"llm",
		"test",
		func() (map[string]any, error) {
			return map[string]any{"test": "data"}, nil
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, "data", result["test"])
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/hooks/... -run TestNoOpHookManager_ExecutorMethods -v`
Expected: FAIL with missing methods

**Step 3: Write minimal implementation**

Add to NoOpHookManager in `pkg/hooks/types.go` (after RegisterSkillHook):

```go
// RegisterExecutorHook is a no-op implementation of HookManager.RegisterExecutorHook.
func (n *NoOpHookManager) RegisterExecutorHook(_ TypedHookFunc[ExecutorPayload], _ TypedHookMetadata) error {
	return nil
}

// TriggerExecutorHooks is a no-op implementation of HookManager.TriggerExecutorHooks.
func (n *NoOpHookManager) TriggerExecutorHooks(_ context.Context, _ HookPoint, hookCtx *TypedHookContext[ExecutorPayload]) TypedHookResult[ExecutorPayload] {
	return TypedHookResult[ExecutorPayload]{Payload: hookCtx.Payload}
}
```

Add after WithLLMHooks:

```go
// WithFlowStepHooks is a no-op implementation of HookManager.WithFlowStepHooks.
func (n *NoOpHookManager) WithFlowStepHooks(_ context.Context, _, _ uuid.UUID, _, _ string, work func() (map[string]any, error)) (map[string]any, error) {
	return work()
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/hooks/... -run TestNoOpHookManager_ExecutorMethods -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/hooks/types.go pkg/hooks/executor_hooks_test.go
git commit -m "test(hooks): add NoOpHookManager executor methods

- Add no-op implementations for executor hook methods
- Enable testing without full hook manager setup"
```

---

## Task 6: Executor Integration - DI Setup

**Files:**
- Modify: `pkg/flows/executor/executor.go:1-85`
- Modify: `pkg/di/container.go`

**Step 1: Add hookManager to executor structs**

Modify `pkg/flows/executor/executor.go`:

```go
package executor

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/denkhaus/gollum/pkg/extensions"
	"github.com/denkhaus/gollum/pkg/flows"
	flowregistry "github.com/denkhaus/gollum/pkg/flows/registry"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/tools"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
)
```

Update flowExecutorImpl struct:

```go
type flowExecutorImpl struct {
	flow             *flows.Flow
	ctx              *Context
	currentState     string
	history          *ExecutionHistory
	startTime        time.Time
	bashToolProvider tools.BashToolProvider
	extService       extensions.ExtensionService
	flowRegistry     flowregistry.FlowRegistry
	hookManager      hooks.HookManager // NEW
}
```

Update flowExecutorServiceImpl struct:

```go
type flowExecutorServiceImpl struct {
	bashToolProvider tools.BashToolProvider
	extService       extensions.ExtensionService
	flowRegistry     flowregistry.FlowRegistry
	hookManager      hooks.HookManager // NEW
}
```

Update NewFlowExecutor constructor:

```go
func NewFlowExecutor(injector do.Injector) (FlowExecutorService, error) {
	bashToolProvider := do.MustInvoke[tools.BashToolProvider](injector)
	extService := do.MustInvoke[extensions.ExtensionService](injector)
	flowRegistry := do.MustInvoke[flowregistry.FlowRegistry](injector)
	hookManager := do.MustInvoke[hooks.HookManager](injector) // NEW

	return &flowExecutorServiceImpl{
		bashToolProvider: bashToolProvider,
		extService:       extService,
		flowRegistry:     flowRegistry,
		hookManager:      hookManager, // NEW
	}, nil
}
```

Update New method:

```go
func (p *flowExecutorServiceImpl) New(flow *flows.Flow) FlowExecutorInstance {
	return &flowExecutorImpl{
		flow:             flow,
		ctx:              NewContext(flow.Input, nil),
		history:          NewExecutionHistory(),
		startTime:        time.Now(),
		bashToolProvider: p.bashToolProvider,
		extService:       p.extService,
		flowRegistry:     p.flowRegistry,
		hookManager:      p.hookManager, // NEW
	}
}
```

**Step 2: Verify HookManager is registered in DI**

Check `pkg/di/container.go` for:

```go
do.Provide(p.injector, hooks.NewHookManager)
```

If not present, add it to the hooks section.

**Step 3: Write integration test**

Create `pkg/flows/executor/executor_hooks_integration_test.go`:

```go
package executor

import (
	"context"
	"testing"
	"time"

	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlowExecutor_WithHookManagerIntegration(t *testing.T) {
	injector := setupTestInjector()

	// Register hook manager
	do.Provide(injector, hooks.NewHookManager)

	// Register a test hook
	hm := do.MustInvoke[hooks.HookManager](injector)
	var beforeCalled, afterCalled bool

	hm.RegisterExecutorHook(func(ctx *TypedHookContext[ExecutorPayload]) error {
		beforeCalled = true
		return nil
	}, hooks.TypedHookMetadata{Name: "test-before", Point: hooks.BeforeFlowStep})

	hm.RegisterExecutorHook(func(ctx *TypedHookContext[ExecutorPayload]) error {
		afterCalled = true
		return nil
	}, hooks.TypedHookMetadata{Name: "test-after", Point: hooks.AfterFlowStep})

	// Create executor
	execService, err := NewFlowExecutor(injector)
	require.NoError(t, err)

	flow := &flows.Flow{
		ID:   uuid.New(),
		Name: "test-flow",
		Input: map[string]flowpb.InputField{
			"name": {Name: "name", Type: "string", Required: true},
		},
		States: []*flows.State{
			{
				Name:    "initial",
				Initial: true,
				Steps: []*flows.Step{
					{Type: "func", Function: "test"},
				},
				Transitions: []*flows.Transition{
					{Target: "final", Condition: &flows.Condition{Expression: "true"}},
				},
			},
			{
				Name: "final",
				Final: true,
			},
		},
	}

	instance := execService.New(flow)
	instance.SetInput(map[string]any{"name": "test"})

	err = instance.Run()
	assert.NoError(t, err)

	// Note: Hooks won't be called yet - that's Task 7
	// This just verifies the executor can be created with HookManager
}
```

**Step 4: Run test**

Run: `go test ./pkg/flows/executor/... -run TestFlowExecutor_WithHookManagerIntegration -v`
Expected: PASS (executor creates successfully)

**Step 5: Commit**

```bash
git add pkg/flows/executor/executor.go pkg/flows/executor/executor_hooks_integration_test.go
git commit -m "feat(executor): add HookManager DI integration

- Add hookManager field to executor structs
- Inject HookManager via DI in NewFlowExecutor
- Add integration test for HookManager setup"
```

---

## Task 7: Executor Step Hook Invocation

**Files:**
- Modify: `pkg/flows/executor/executor.go` (executeStep method)
- Test: `pkg/flows/executor/executor_hooks_integration_test.go`

**Step 1: Write failing test**

Update `pkg/flows/executor/executor_hooks_integration_test.go`:

```go
func TestFlowExecutor_ExecuteStepWithHooks(t *testing.T) {
	injector := setupTestInjector()
	do.Provide(injector, hooks.NewHookManager)

	hm := do.MustInvoke[hooks.HookManager](injector)

	var capturedBefore, capturedAfter *hooks.ExecutorPayload

	hm.RegisterExecutorHook(func(ctx *TypedHookContext[hooks.ExecutorPayload]) error {
		capturedBefore = ctx.Payload
		return nil
	}, hooks.TypedHookMetadata{Name: "before", Point: hooks.BeforeFlowStep})

	hm.RegisterExecutorHook(func(ctx *TypedHookContext[hooks.ExecutorPayload]) error {
		capturedAfter = ctx.Payload
		return nil
	}, hooks.TypedHookMetadata{Name: "after", Point: hooks.AfterFlowStep})

	execService, _ := NewFlowExecutor(injector)

	flow := &flows.Flow{
		ID:   uuid.New(),
		Name: "hook-test-flow",
		Input: map[string]flowpb.InputField{
			"name": {Name: "name", Type: "string", Required: true},
		},
		States: []*flows.State{
			{
				Name:    "initial",
				Initial: true,
				Steps: []*flows.Step{
					{Type: "func", Function: "noop"}, // Will be mocked
				},
				Transitions: []*flows.Transition{
					{Target: "final", Condition: &flows.Condition{Expression: "true"}},
				},
			},
			{Name: "final", Final: true},
		},
	}

	instance := execService.New(flow)
	instance.SetInput(map[string]any{"name": "test"})

	err := instance.Run()

	assert.NoError(t, err)

	// Verify hooks were called
	assert.NotNil(t, capturedBefore, "Before hook should be called")
	assert.NotNil(t, capturedAfter, "After hook should be called")
	assert.Equal(t, "hook-test-flow", capturedBefore.FlowName)
	assert.Equal(t, "func", capturedBefore.StepType)
	assert.Equal(t, "initial", capturedBefore.StateName)
	assert.Greater(t, capturedAfter.Duration, time.Duration(0))
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/executor/... -run TestFlowExecutor_ExecuteStepWithHooks -v`
Expected: FAIL (hooks not being called)

**Step 3: Modify executeStep to use hooks**

Find the executeStep method in `pkg/flows/executor/executor.go` and modify it:

```go
func (p *flowExecutorImpl) executeStep(step *flows.Step, stateName string) error {
	// Wrap step execution with hooks
	_, err := p.hookManager.WithFlowStepHooks(
		context.Background(),
		uuid.Nil, // SessionID - will be available when executor is used in session context
		p.flow.ID,
		step.Type,
		stateName,
		func() (map[string]any, error) {
			// Execute the actual step logic
			switch step.Type {
			case "llm":
				return nil, p.executeLLMStep(step, stateName)
			case "shell":
				return nil, p.executeShellStep(step, stateName)
			case "func":
				return nil, p.executeFuncStep(step, stateName)
			case "mcp":
				return nil, p.executeMCPStep(step, stateName)
			default:
				return nil, fmt.Errorf("unknown step type: %s", step.Type)
			}
		},
	)

	return err
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/executor/... -run TestFlowExecutor_ExecuteStepWithHooks -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/flows/executor/executor.go pkg/flows/executor/executor_hooks_integration_test.go
git commit -m "feat(executor): integrate hooks into executeStep

- Wrap step execution with WithFlowStepHooks
- Call BeforeFlowStep hooks before step execution
- Call AfterFlowStep hooks after step execution
- Track execution duration in AfterFlowStep payload"
```

---

## Task 8: Hook Blocking Support

**Files:**
- Test: `pkg/flows/executor/executor_hooks_integration_test.go`

**Step 1: Write failing test**

```go
func TestFlowExecutor_HookCanBlockStep(t *testing.T) {
	injector := setupTestInjector()
	do.Provide(injector, hooks.NewHookManager)

	hm := do.MustInvoke[hooks.HookManager](injector)

	// Register blocking hook for shell steps
	hm.RegisterExecutorHook(func(ctx *TypedHookContext[hooks.ExecutorPayload]) error {
		if ctx.Payload.StepType == "shell" {
			ctx.Blocked = true
		}
		return nil
	}, hooks.TypedHookMetadata{
		Name:     "no-shell",
		Point:    hooks.BeforeFlowStep,
		Priority: 100,
	})

	execService, _ := NewFlowExecutor(injector)

	flow := &flows.Flow{
		ID:   uuid.New(),
		Name: "blocking-test",
		Input: map[string]flowpb.InputField{
			"name": {Name: "name", Type: "string", Required: true},
		},
		States: []*flows.State{
			{
				Name:    "initial",
				Initial: true,
				Steps: []*flows.Step{
					{Type: "shell", Command: "echo blocked"},
				},
				Transitions: []*flows.Transition{
					{Target: "final", Condition: &flows.Condition{Expression: "true"}},
				},
			},
			{Name: "final", Final: true},
		},
	}

	instance := execService.New(flow)
	instance.SetInput(map[string]any{"name": "test"})

	err := instance.Run()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blocked")
}
```

**Step 2: Run test**

Run: `go test ./pkg/flows/executor/... -run TestFlowExecutor_HookCanBlockStep -v`
Expected: Should PASS if blocking works correctly from Task 7

**Step 3: If failing, ensure error propagation**

The WithFlowStepHooks implementation should return error when blocked. Verify the error message is clear.

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/executor/... -run TestFlowExecutor_HookCanBlockStep -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/flows/executor/executor_hooks_integration_test.go
git commit -m "test(executor): add hook blocking test

- Test that BeforeFlowStep hooks can block step execution
- Verify error is propagated correctly"
```

---

## Task 9: Cleanup - Remove Unused Extension Hooks

**Files:**
- Modify: `pkg/extensions/hooks.go`
- Test: Update affected tests

**Step 1: Verify pkg/extensions/hooks.go is unused**

```bash
grep -r "HookAgentPreExecute\|HookAgentPostExecute\|HookFlowPreExecute\|HookFlowPostExecute\|HookToolPreExecute\|HookToolPostExecute" --include="*.go" | grep -v "pkg/extensions/hooks.go"
```

Expected: No results (hooks are not used)

**Step 2: Add deprecation notice or remove**

Since these hooks were never implemented, we can either:
- A) Deprecate with notice
- B) Remove entirely

Let's deprecate for now:

Modify `pkg/extensions/hooks.go` header:

```go
// Package extensions provides core services for the Gollum extension system.
//
// DEPRECATED: The hook definitions in this file are replaced by the comprehensive
// hook system in pkg/hooks. Use pkg/hooks.HookManager for flow execution hooks:
//
// - BeforeFlowStep / AfterFlowStep (instead of HookFlowPreExecute/PostExecute)
// - BeforeToolExecution / AfterToolExecution (instead of HookToolPreExecute/PostExecute)
// - BeforeAgentSpawn / AfterAgentSpawn (instead of HookAgentPreExecute/PostExecute)
//
// This file is kept for backward compatibility and will be removed in a future version.
package extensions
```

**Step 3: Run tests to ensure nothing breaks**

Run: `go test ./pkg/extensions/... -v`
Expected: All existing tests still pass

**Step 4: Commit**

```bash
git add pkg/extensions/hooks.go
git commit -m "docs(extensions): deprecate unused hook definitions

- Add deprecation notice to pkg/extensions/hooks.go
- Document migration path to pkg/hooks system
- Keep for backward compatibility, will remove in future"
```

---

## Task 10: Documentation and Examples

**Files:**
- Create: `docs/guides/executor-hooks.md`

**Step 1: Create hook usage guide**

Create `docs/guides/executor-hooks.md`:

```markdown
# Executor Hooks Guide

## Overview

The Flow Executor supports hooks at the step level through the comprehensive hook system in `pkg/hooks`. These hooks allow you to inspect, validate, and monitor flow execution.

## Available Hooks

### BeforeFlowStep

Triggered before any step in a flow executes.

**Use cases:**
- Logging step execution
- Validating step permissions
- Monitoring/debugging
- Blocking certain step types in production

**Payload:**
```go
type ExecutorPayload struct {
    FlowID       uuid.UUID
    FlowName     string
    SessionID    uuid.UUID
    CurrentState string
    StepType     string  // "llm", "shell", "func", "mcp"
    StateName    string
}
```

### AfterFlowStep

Triggered after any step in a flow completes.

**Use cases:**
- Logging step results
- Recording metrics
- Error tracking
- Performance monitoring

**Additional fields:**
```go
StepResult map[string]any
StepError  error
Duration   time.Duration
```

## Registration

```go
import "github.com/denkhaus/gollum/pkg/hooks"

// Get HookManager from DI
hm := do.MustInvoke[hooks.HookManager](injector)

// Register BeforeFlowStep hook
hm.RegisterExecutorHook(func(ctx *TypedHookContext[ExecutorPayload]) error {
    log.Info("Executing step",
        zap.String("flow", ctx.Payload.FlowName),
        zap.String("type", ctx.Payload.StepType),
        zap.String("state", ctx.Payload.StateName))
    return nil
}, hooks.TypedHookMetadata{
    Name:     "flow-logger",
    Point:    hooks.BeforeFlowStep,
    Priority: 50,
})

// Register AfterFlowStep hook for metrics
hm.RegisterExecutorHook(func(ctx *TypedHookContext[ExecutorPayload]) error {
    metrics.RecordStepDuration(
        ctx.Payload.StepType,
        ctx.Payload.Duration,
    )
    if ctx.Payload.StepError != nil {
        metrics.RecordStepError(ctx.Payload.StepType)
    }
    return nil
}, hooks.TypedHookMetadata{
    Name:  "metrics-collector",
    Point: hooks.AfterFlowStep,
})
```

## Blocking Execution

Hooks can block step execution by setting `ctx.Blocked = true`:

```go
hm.RegisterExecutorHook(func(ctx *TypedHookContext[ExecutorPayload]) error {
    // Block shell steps in production
    if ctx.Payload.StepType == "shell" && isProduction() {
        ctx.Blocked = true
        return nil
    }
    return nil
}, hooks.TypedHookMetadata{
    Name:     "prod-guard",
    Point:    hooks.BeforeFlowStep,
    Priority: 100,
})
```

## Hook Layering

The executor has hooks at multiple levels:

| Level | Hooks | Scope |
|-------|-------|-------|
| **Flow** | BeforeFlowStep / AfterFlowStep | All steps in flow |
| **Tool** | BeforeToolExecution / AfterToolExecution | Tool calls only |
| **LLM** | BeforeLLMRequest / AfterLLMResponse | LLM calls only |

This allows both general flow-level monitoring and specific tool/LLM behavior modification.
```

**Step 2: Update existing docs**

Update `docs/extension-system.md` to reference the new hooks:

```markdown
## Hooks and Lifecycle

Flow execution can be extended using hooks from `pkg/hooks.HookManager`:

- **BeforeFlowStep / AfterFlowStep** - Monitor all step execution
- **BeforeToolExecution / AfterToolExecution** - Modify tool behavior
- **BeforeLLMRequest / AfterLLMResponse** - Modify LLM prompts/responses

See [Executor Hooks Guide](guides/executor-hooks.md) for details.
```

**Step 3: Commit**

```bash
git add docs/guides/executor-hooks.md docs/extension-system.md
git commit -m "docs: add executor hooks usage guide

- Document BeforeFlowStep and AfterFlowStep hooks
- Provide examples for logging, metrics, and blocking
- Explain hook layering (Flow vs Tool vs LLM)
- Update extension-system.md with hook references"
```

---

## Summary

This plan implements executor hooks in 10 tasks following TDD principles:

1. ✅ ExecutorPayload structure
2. ✅ HookManager registry extension
3. ✅ TriggerExecutorHooks method
4. ✅ WithFlowStepHooks wrapper
5. ✅ NoOpHookManager executor methods
6. ✅ Executor DI integration
7. ✅ executeStep hook invocation
8. ✅ Hook blocking support
9. ✅ Cleanup deprecated hooks
10. ✅ Documentation

Each task follows the TDD cycle: write failing test → implement → verify pass → commit.

**Total estimated time:** 2-3 hours

**Key files created/modified:**
- `pkg/hooks/executor_hooks.go` (new)
- `pkg/hooks/executor_hooks_test.go` (new)
- `pkg/hooks/manager.go` (modified)
- `pkg/hooks/types.go` (modified)
- `pkg/flows/executor/executor.go` (modified)
- `pkg/flows/executor/executor_hooks_integration_test.go` (new)
- `pkg/extensions/hooks.go` (deprecated)
- `docs/guides/executor-hooks.md` (new)
