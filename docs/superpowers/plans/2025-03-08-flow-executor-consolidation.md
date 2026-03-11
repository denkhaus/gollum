# Flow Executor Consolidation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Consolidate two Flow executor implementations into a single, DI-integrated service following `guide.golang.di.md` patterns.

**Architecture:** Merge feature-complete `executor.go` with DI service pattern from `di_service.go`. Create FlowRegistry as DI service. All sub-services (BashToolProvider, ExtensionService, FlowRegistry) injected via DI. Private implementations with `p` receivers.

**Tech Stack:** Go 1.23, samber/do/v2 (DI), GoMock (testing)

---

## Task 1: Create FlowRegistry Service

**Files:**
- Create: `pkg/flows/registry/service.go`
- Modify: `pkg/di/container.go` (add registration)
- Create: `pkg/flows/registry/service_test.go`

**Step 1: Write the failing test**

Create `pkg/flows/registry/service_test.go`:

```go
package registry

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlowRegistryService_GetFlow_ReturnsRegisteredFlow(t *testing.T) {
	// Create service
	svc := &flowRegistryServiceImpl{
		flows: make(map[string]*flows.Flow),
	}

	// Register a flow
	testFlow := &flows.Flow{
		Name:    "test-flow",
		Version: "1.0",
		States: []flows.State{
			{Name: "init", Initial: true},
		},
	}
	svc.flows["test-flow"] = testFlow

	// Get flow
	result, err := svc.GetFlow("test-flow")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "test-flow", result.Name)
}

func TestFlowRegistryService_GetFlow_NotFound_ReturnsError(t *testing.T) {
	svc := &flowRegistryServiceImpl{
		flows: make(map[string]*flows.Flow),
	}

	_, err := svc.GetFlow("nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "flow not found")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/registry/... -v`
Expected: FAIL with "undefined: flowRegistryServiceImpl"

**Step 3: Write minimal implementation**

Create `pkg/flows/registry/service.go`:

```go
package registry

import (
	"fmt"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/samber/do/v2"
)

// FlowRegistry defines the interface for looking up flows by reference
type FlowRegistry interface {
	GetFlow(ref string) (*flows.Flow, error)
}

// flowRegistryServiceImpl is the private implementation
type flowRegistryServiceImpl struct {
	flows map[string]*flows.Flow
}

// Ensure flowRegistryServiceImpl implements FlowRegistry
var _ FlowRegistry = (*flowRegistryServiceImpl)(nil)

// NewFlowRegistryService creates the flow registry service (DI constructor)
func NewFlowRegistryService(injector do.Injector) (FlowRegistry, error) {
	return &flowRegistryServiceImpl{
		flows: make(map[string]*flows.Flow),
	}, nil
}

// Register adds a flow to the registry
func (p *flowRegistryServiceImpl) Register(name string, flow *flows.Flow) {
	p.flows[name] = flow
}

// GetFlow retrieves a flow by reference name
func (p *flowRegistryServiceImpl) GetFlow(ref string) (*flows.Flow, error) {
	flow, ok := p.flows[ref]
	if !ok {
		return nil, fmt.Errorf("flow not found: %s", ref)
	}
	return flow, nil
}

// LoadFromMap loads flows from a map (for initialization)
func (p *flowRegistryServiceImpl) LoadFromMap(flows map[string]*flows.Flow) {
	for name, flow := range flows {
		p.flows[name] = flow
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/flows/registry/... -v`
Expected: PASS

**Step 5: Register in DI container**

Modify `pkg/di/container.go` - add to RegisterServices after line 104:

```go
	// Flows
	do.Provide(p.injector, registry.NewFlowRegistryService)
	do.Provide(p.injector, executor.NewFlowExecutor)
```

**Step 6: Commit**

```bash
git add pkg/flows/registry/service.go pkg/flows/registry/service_test.go pkg/di/container.go
git commit -m "feat(flows): add FlowRegistry DI service

Add FlowRegistry as a DI service for looking up flows by reference.
This is required for executor call step support.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 2: Consolidate Executor - Add DI Interfaces

**Files:**
- Modify: `pkg/flows/executor/executor.go` (add interfaces, remove old constructors)

**Step 1: Add public interfaces at top of executor.go**

After imports, add:

```go
// FlowExecutorService defines the DI service that creates executor instances
type FlowExecutorService interface {
	// New creates a new executor instance for a flow
	New(flow *flows.Flow) FlowExecutorInstance
}

// FlowExecutorInstance defines the interface for a flow executor instance
type FlowExecutorInstance interface {
	// SetInput sets input field values
	SetInput(vals map[string]any)
	// Validate validates the flow before execution
	Validate() error
	// Run executes the flow from the initial state
	Run() error
	// GetContext returns the execution context (for testing)
	GetContext() *Context
}
```

**Step 2: Rename Executor to flowExecutorImpl**

Find: `type Executor struct`
Replace with: `type flowExecutorImpl struct`

**Step 3: Add flowExecutorServiceImpl**

After flowExecutorImpl, add:

```go
// flowExecutorServiceImpl is the DI service that creates executor instances
type flowExecutorServiceImpl struct {
	bashToolProvider tools.BashToolProvider
	extService       extensions.ExtensionService
	flowRegistry     FlowRegistry
}

// Ensure flowExecutorServiceImpl implements FlowExecutorService
var _ FlowExecutorService = (*flowExecutorServiceImpl)(nil)

// Ensure flowExecutorImpl implements FlowExecutorInstance
var _ FlowExecutorInstance = (*flowExecutorImpl)(nil)
```

**Step 4: Add DI constructor**

After the struct definitions, add:

```go
// NewFlowExecutor creates the flow executor service (DI constructor)
func NewFlowExecutor(injector do.Injector) (FlowExecutorService, error) {
	bashToolProvider := do.MustInvoke[tools.BashToolProvider](injector)
	extService := do.MustInvoke[extensions.ExtensionService](injector)
	flowRegistry := do.MustInvoke[registry.FlowRegistry](injector)

	return &flowExecutorServiceImpl{
		bashToolProvider: bashToolProvider,
		extService:       extService,
		flowRegistry:     flowRegistry,
	}, nil
}
```

**Step 5: Add service.New() method**

Add to flowExecutorServiceImpl:

```go
// New creates a new executor instance for a specific flow
func (p *flowExecutorServiceImpl) New(flow *flows.Flow) FlowExecutorInstance {
	return &flowExecutorImpl{
		flow:             flow,
		ctx:              NewContext(flow.Input, nil),
		history:          NewExecutionHistory(),
		startTime:        time.Now(),
		bashToolProvider: p.bashToolProvider,
		extService:       p.extService,
		flowRegistry:     p.flowRegistry,
	}
}
```

**Step 6: Remove old constructors**

Delete these functions:
- `NewExecutor()`
- `NewExecutorWithProvider()`
- `NewExecutorWithRegistry()`
- `NewExecutorWithExtensions()`

**Step 7: Update all method receivers**

Find all `func (e *Executor)` and replace with `func (p *flowExecutorImpl)`

**Step 8: Commit**

```bash
git add pkg/flows/executor/executor.go
git commit -m "refactor(executor): add DI service interfaces

Add FlowExecutorService and FlowExecutorInstance interfaces.
Rename Executor to flowExecutorImpl (private implementation).
Add flowExecutorServiceImpl as DI service.
Add single DI constructor NewFlowExecutor().

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 3: Merge executeFuncStep from executor.go

**Files:**
- Modify: `pkg/flows/executor/executor.go` (replace executeFuncStep)

**Step 1: Locate executeFuncStep in executor.go**

Find the current executeFuncStep implementation (around line 356).

**Step 2: Replace with feature-complete version**

Replace entire executeFuncStep method:

```go
func (p *flowExecutorImpl) executeFuncStep(step *flows.Step, stateName string) error {
	// Use Scriggo runner from extension service
	funcRunner := p.extService.GetFuncRunner()

	// Build args with template substitution
	args := make(map[string]any)
	for _, param := range step.Params {
		value := p.substituteTemplate(param.Value)
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
		p.ctx.SetOutputField(fieldName, result)
	}

	return nil
}
```

**Step 3: Verify FuncError type exists**

Check that FuncError is defined in `executor/errors.go`. If not, add it.

**Step 4: Commit**

```bash
git add pkg/flows/executor/executor.go
git commit -m "refactor(executor): merge executeFuncStep with ExtensionService support

Use ExtensionService.GetFuncRunner() for func step execution.
Supports both built-in and extension functions.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 4: Merge executeLLMStep from executor.go

**Files:**
- Modify: `pkg/flows/executor/executor.go` (add executeLLMStep)
- Reference: `pkg/flows/executor/llm_step.go`

**Step 1: Read llm_step.go for implementation**

Read `pkg/flows/executor/llm_step.go` to get the full implementation.

**Step 2: Add executeLLMStep method**

Add to flowExecutorImpl:

```go
func (p *flowExecutorImpl) executeLLMStep(step *flows.Step, stateName string) error {
	// Find agent configuration
	var agentConfig *flows.Agent
	for i := range p.flow.Agents {
		if p.flow.Agents[i].Name == step.Agent {
			agentConfig = &p.flow.Agents[i]
			break
		}
	}

	if agentConfig == nil {
		return fmt.Errorf("agent not found: %s", step.Agent)
	}

	// Substitute template variables in prompt
	prompt := p.substituteTemplate(step.Prompt)

	// For now, return error - full LLM integration needed
	return fmt.Errorf("llm step execution not yet implemented: agent=%s prompt=%s", step.Agent, prompt)
}
```

Note: Full LLM integration will be a separate task.

**Step 3: Commit**

```bash
git add pkg/flows/executor/executor.go
git commit -m "refactor(executor): add executeLLMStep stub

Add LLM step execution placeholder.
Full LLM integration to be implemented separately.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 5: Add handleError method

**Files:**
- Modify: `pkg/flows/executor/executor.go`

**Step 1: Add handleError method**

Add to flowExecutorImpl:

```go
func (p *flowExecutorImpl) handleError(err error, step *flows.Step, state *flows.State) error {
	// Record error in history
	stepName := "unknown"
	stepType := "unknown"
	if step != nil {
		stepName = step.Name
		stepType = step.Type
	}
	p.history.RecordError(stepName, stepType, err.Error(), time.Now())
	return err
}
```

**Step 2: Update executeStep to use handleError**

Ensure executeStep calls `p.handleError(err, &step, state)` on errors.

**Step 3: Commit**

```bash
git add pkg/flows/executor/executor.go
git commit -m "refactor(executor): add handleError method

Add error recording to execution history.
Properly captures step name, type, and error message.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 6: Remove nil-check for flowRegistry

**Files:**
- Modify: `pkg/flows/executor/executor.go`

**Step 1: Update executeCall method**

Find executeCall and remove the nil-check:

```go
func (p *flowExecutorImpl) executeCall(call *flows.Call, stateName string) error {
	// Look up the sub-flow
	subFlow, err := p.flowRegistry.GetFlow(call.Ref)
	if err != nil {
		return fmt.Errorf("flow lookup failed for %s: %w", call.Ref, err)
	}

	// Build input map from call.Input fields with template substitution
	subInput := make(map[string]any)
	for _, field := range call.Input {
		value := p.substituteTemplate(field.Value)
		subInput[field.Name] = value
	}

	// Create executor for sub-flow via service
	subExec := NewExecutor(subFlow)
	subExec.SetInput(subInput)

	// Execute the sub-flow
	if err := subExec.Run(); err != nil {
		return fmt.Errorf("sub-flow execution failed: %w", err)
	}

	// Map output fields back using call.Output
	for _, field := range call.Output {
		fieldName := extractFieldName(field.Value)
		value, ok := subExec.ctx.GetOutputField(fieldName)
		if !ok {
			continue
		}

		targetField := extractFieldName(field.Name)
		p.ctx.SetOutputField(targetField, value)
	}

	return nil
}
```

Wait - this needs to use the service, not direct NewExecutor. Let me fix:

```go
func (p *flowExecutorImpl) executeCall(call *flows.Call, stateName string) error {
	// Look up the sub-flow
	subFlow, err := p.flowRegistry.GetFlow(call.Ref)
	if err != nil {
		return fmt.Errorf("flow lookup failed for %s: %w", call.Ref, err)
	}

	// Build input map from call.Input fields with template substitution
	subInput := make(map[string]any)
	for _, field := range call.Input {
		value := p.substituteTemplate(field.Value)
		subInput[field.Name] = value
	}

	// Create sub-executor instance directly
	subExec := &flowExecutorImpl{
		flow:             subFlow,
		ctx:              NewContext(subFlow.Input, subInput),
		history:          NewExecutionHistory(),
		startTime:        time.Now(),
		bashToolProvider: p.bashToolProvider,
		extService:       p.extService,
		flowRegistry:     p.flowRegistry,
	}

	// Execute the sub-flow
	if err := subExec.Run(); err != nil {
		return fmt.Errorf("sub-flow execution failed: %w", err)
	}

	// Map output fields back using call.Output
	for _, field := range call.Output {
		fieldName := extractFieldName(field.Value)
		value, ok := subExec.ctx.GetOutputField(fieldName)
		if !ok {
			continue
		}

		targetField := extractFieldName(field.Name)
		p.ctx.SetOutputField(targetField, value)
	}

	return nil
}
```

**Step 2: Commit**

```bash
git add pkg/flows/executor/executor.go
git commit -m "refactor(executor): remove flowRegistry nil-check

FlowRegistry is now a required DI service, never nil.
Update executeCall to create sub-executor directly.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 7: Update executor_test.go to use DI

**Files:**
- Modify: `pkg/flows/executor/executor_test.go`

**Step 1: Add test helper with DI**

Add at top of file after imports:

```go
import (
	"github.com/denkhaus/gollum/pkg/extensions"
	"github.com/denkhaus/gollum/pkg/flows/registry"
	"github.com/denkhaus/gollum/pkg/tools"
	"github.com/samber/do/v2"
)

// setupTestDI creates a test DI injector with required services
func setupTestDI(t *testing.T) do.Injector {
	injector := do.New()

	// Mock services
	do.ProvideValue(injector, tools.MockBashToolProvider())
	do.ProvideValue(injector, extensions.MockExtensionService())
	do.ProvideValue(injector, registry.NewFlowRegistryService(injector))

	return injector
}
```

**Step 2: Update TestNewExecutor_CreatesExecutorWithFlow**

```go
func TestFlowExecutorService_New_CreatesExecutorInstance(t *testing.T) {
	flow := &flows.Flow{
		Name: "test-flow",
		States: []flows.State{
			{Name: "init", Initial: true},
			{Name: "done"},
		},
	}

	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)

	instance := svc.New(flow)

	assert.NotNil(t, instance)
	assert.Equal(t, "init", instance.GetContext().currentState)
}
```

**Step 3: Update TestExecutor_Validate_ReturnsErrorForInvalidFlow**

```go
func TestFlowExecutorInstance_Validate_ReturnsErrorForInvalidFlow(t *testing.T) {
	flow := &flows.Flow{
		Name: "invalid-flow",
		States: []flows.State{
			{Name: "init"}, // No initial state
		},
	}

	injector := setupTestDI(t)
	svc := do.MustInvoke[FlowExecutorService](injector)
	instance := svc.New(flow)

	err := instance.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no initial state")
}
```

**Step 4: Run tests to verify**

Run: `go test ./pkg/flows/executor/executor_test.go -v`
Expected: FAIL (needs mock implementations)

**Step 5: Create mock implementations in tools package**

For now, create simple mock in test:

```go
// MockBashToolProvider returns a mock bash tool provider
func MockBashToolProvider() tools.BashToolProvider {
	// Simple mock - implement properly later
	return nil
}
```

**Step 6: Commit**

```bash
git add pkg/flows/executor/executor_test.go
git commit -m "test(executor): update to use DI pattern

Update tests to use FlowExecutorService interface.
Add setupTestDI helper for test injector.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 8: Update remaining test files

**Files:**
- Modify: `pkg/flows/executor/integration_test.go`
- Modify: `pkg/flows/executor/llm_step_test.go`
- Modify: `pkg/flows/executor/shell_step_test.go`
- Modify: `pkg/flows/executor/func_step_test.go`
- Modify: `pkg/flows/executor/call_step_test.go`
- Modify: `pkg/flows/executor/mcp_step_test.go`

**Step 1: Update each test file**

Pattern for each file:
1. Add `setupTestDI(t *testing.T) do.Injector` helper
2. Change `NewExecutor()` to `svc.New()` where `svc` is invoked from DI
3. Update assertions to use `FlowExecutorInstance` interface

**Step 2: Run all executor tests**

Run: `go test ./pkg/flows/executor/... -v`
Expected: All PASS after updates

**Step 3: Commit each file separately**

```bash
git add pkg/flows/executor/integration_test.go
git commit -m "test(executor): update integration_test.go to use DI"

git add pkg/flows/executor/llm_step_test.go
git commit -m "test(executor): update llm_step_test.go to use DI"

# ... repeat for each test file
```

---

## Task 9: Remove di_service.go

**Files:**
- Delete: `pkg/flows/executor/di_service.go`

**Step 1: Verify no imports of di_service**

Run: `grep -r "executor/di_service" --include="*.go" | grep -v "_test.go"`
Expected: No results

**Step 2: Delete di_service.go**

Run: `rm pkg/flows/executor/di_service.go`

**Step 3: Run tests to verify**

Run: `go test ./pkg/flows/executor/... -v`
Expected: All PASS

**Step 4: Commit**

```bash
git add pkg/flows/executor/di_service.go
git commit -m "refactor(executor): remove di_service.go

Consolidated into executor.go with proper DI integration.
All functionality preserved in single implementation.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 10: Remove flow_registry.go (test-only)

**Files:**
- Delete: `pkg/flows/executor/flow_registry.go`

**Step 1: Check if flow_registry.go is used**

Run: `grep -r "SimpleFlowRegistry" --include="*.go"`
Expected: Only in tests or not at all

**Step 2: Update tests to use FlowRegistry from registry package**

If tests use SimpleFlowRegistry, replace with `registry.NewFlowRegistryService()`

**Step 3: Delete flow_registry.go**

Run: `rm pkg/flows/executor/flow_registry.go`

**Step 4: Commit**

```bash
git add pkg/flows/executor/flow_registry.go
git commit -m "refactor(executor): remove flow_registry.go

Test-only SimpleFlowRegistry replaced by proper DI service.
FlowRegistry now in pkg/flows/registry package.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 11: Final Verification

**Files:**
- Test: All executor tests

**Step 1: Run full test suite**

Run: `go test ./pkg/flows/... -v`
Expected: All PASS

**Step 2: Verify DI registration**

Run: `grep -n "NewFlowExecutor" pkg/di/container.go`
Expected: Line shows `do.Provide(p.injector, executor.NewFlowExecutor)`

**Step 3: Verify no duplicate code**

Run: `grep -n "extractFieldName" pkg/flows/executor/executor.go`
Expected: Only one definition (not `extractFieldName` and `extractFieldNameForService`)

**Step 4: Build verification**

Run: `go build ./...`
Expected: No errors

**Step 5: Final commit**

```bash
git add docs/plans/2025-03-08-flow-executor-consolidation.md
git commit -m "docs(executor): add implementation plan

Add detailed implementation plan for flow executor consolidation.
Break down work into 11 bite-sized tasks.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Completion Checklist

- [ ] FlowRegistry service created and registered
- [ ] Executor uses DI service pattern
- [ ] All sub-services injected via DI
- [ ] executeFuncStep uses ExtensionService
- [ ] executeLLMStep merged from executor.go
- [ ] handleError method added
- [ ] flowRegistry never nil (no nil-checks)
- [ ] All tests updated to use DI
- [ ] di_service.go deleted
- [ ] flow_registry.go deleted
- [ ] All tests pass
- [ ] No duplicate code
- [ ] Build succeeds

---

## References

- DI Guidance: `docs/guides/guide.golang.di.md`
- Design Document: `docs/plans/2025-03-08-flow-executor-consolidation-design.md`
- Current Executor: `pkg/flows/executor/executor.go`
- DI Service (to delete): `pkg/flows/executor/di_service.go`
