package executor

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/denkhaus/gollum/pkg/extensions"
	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/registry"
	"github.com/denkhaus/gollum/pkg/tools"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
)

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

// FlowExecutorService defines the DI service that creates executor instances
type FlowExecutorService interface {
	// New creates a new executor instance for a flow
	New(flow *flows.Flow) FlowExecutorInstance
}

// flowExecutorImpl is the private implementation of a flow executor instance
type flowExecutorImpl struct {
	flow             *flows.Flow
	ctx              *Context
	currentState     string
	history          *ExecutionHistory
	startTime        time.Time
	bashToolProvider *bashToolProviderWrapper
	flowRegistry     FlowRegistry
	extService       extensions.ExtensionService
}

// Ensure flowExecutorImpl implements FlowExecutorInstance
var _ FlowExecutorInstance = (*flowExecutorImpl)(nil)

// bashToolProviderWrapper wraps tools.BashToolProvider to return executor.BashToolRunner
type bashToolProviderWrapper struct {
	provider tools.BashToolProvider
}

func (w *bashToolProviderWrapper) CreateTool(agentID uuid.UUID) BashToolRunner {
	// *tools.BashTool implicitly satisfies BashToolRunner interface
	return w.provider.CreateTool(agentID)
}

// flowExecutorServiceImpl is the DI service that creates executor instances
type flowExecutorServiceImpl struct {
	bashToolProvider *bashToolProviderWrapper
	extService       extensions.ExtensionService
}

// Ensure flowExecutorServiceImpl implements FlowExecutorService
var _ FlowExecutorService = (*flowExecutorServiceImpl)(nil)

// NewFlowExecutor creates the flow executor service (DI constructor)
func NewFlowExecutor(injector do.Injector) (FlowExecutorService, error) {
	bashToolProvider := do.MustInvoke[tools.BashToolProvider](injector)
	wrapper := &bashToolProviderWrapper{provider: bashToolProvider}

	extService, err := do.Invoke[extensions.ExtensionService](injector)
	if err != nil {
		return nil, fmt.Errorf("get extension service: %w", err)
	}

	return &flowExecutorServiceImpl{
		bashToolProvider: wrapper,
		extService:       extService,
	}, nil
}

// New creates a new executor instance for a specific flow
func (s *flowExecutorServiceImpl) New(flow *flows.Flow) FlowExecutorInstance {
	return &flowExecutorImpl{
		flow:             flow,
		ctx:              NewContext(flow.Input, nil),
		history:          NewExecutionHistory(),
		startTime:        time.Now(),
		bashToolProvider: s.bashToolProvider,
		extService:       s.extService,
	}
}

// SetInput sets input field values
func (e *flowExecutorImpl) SetInput(vals map[string]any) {
	e.ctx = NewContext(e.flow.Input, vals)
}

// Validate validates the flow before execution
func (e *flowExecutorImpl) Validate() error {
	// Check for initial state
	hasInitial := false
	for _, state := range e.flow.States {
		if state.Initial {
			hasInitial = true
			e.currentState = state.Name
			break
		}
	}

	if !hasInitial {
		return fmt.Errorf("flow %s: no initial state defined", e.flow.Name)
	}

	return nil
}

// Run executes the flow from the initial state
func (e *flowExecutorImpl) Run() error {
	defer e.history.Complete(time.Now())

	if err := e.Validate(); err != nil {
		return err
	}

	// Find initial state
	var initialState *flows.State
	for i := range e.flow.States {
		if e.flow.States[i].Initial {
			initialState = &e.flow.States[i]
			break
		}
	}

	if initialState == nil {
		return fmt.Errorf("no initial state found")
	}

	return e.executeState(initialState)
}

// GetContext returns the execution context (for testing)
func (e *flowExecutorImpl) GetContext() *Context {
	return e.ctx
}

// executeState executes a single state
func (e *flowExecutorImpl) executeState(state *flows.State) error {
	// Record state entry
	e.history.RecordStateEntry(state.Name, time.Now())

	// Evaluate computed fields
	if e.flow.Context != nil {
		if err := e.ctx.EvaluateComputedFields(e.flow.Context); err != nil {
			return fmt.Errorf("computed field evaluation: %w", err)
		}
	}

	// Execute steps
	for _, step := range state.Steps {
		if err := e.executeStep(&step, state.Name); err != nil {
			return e.handleError(err, &step, state)
		}
	}

	// Execute calls
	for _, call := range state.Calls {
		if err := e.executeCall(&call, state.Name); err != nil {
			return e.handleError(err, nil, state)
		}
	}

	// Find and execute transition
	return e.executeTransition(state)
}

// executeTransition evaluates conditions and transitions to next state
func (e *flowExecutorImpl) executeTransition(state *flows.State) error {
	// Build evaluation scope
	scope := e.ctx.buildScope()

	for _, trans := range state.Transitions {
		if trans.Otherwise {
			// Fallback transition
			return e.transitionTo(trans.To)
		}

		if trans.When == "" {
			// Unconditional transition
			return e.transitionTo(trans.To)
		}

		// Evaluate condition
		eval := NewEvaluator()
		result, err := eval.EvaluateExpr(trans.When, scope)
		if err != nil {
			return fmt.Errorf("transition condition: %w", err)
		}

		if boolVal, ok := result.(bool); ok && boolVal {
			return e.transitionTo(trans.To)
		}
	}

	// No transition - terminal state
	e.history.RecordStateExit(state.Name, time.Now())
	return nil
}

// transitionTo transitions to a new state
func (e *flowExecutorImpl) transitionTo(stateName string) error {
	// Find target state
	var targetState *flows.State
	for i := range e.flow.States {
		if e.flow.States[i].Name == stateName {
			targetState = &e.flow.States[i]
			break
		}
	}

	if targetState == nil {
		return fmt.Errorf("state not found: %s", stateName)
	}

	e.currentState = stateName
	return e.executeState(targetState)
}

// executeStep executes a single step
func (e *flowExecutorImpl) executeStep(step *flows.Step, stateName string) error {
	switch step.Type {
	case "llm":
		return e.executeLLMStep(step, stateName)
	case "shell":
		return e.executeShellStep(step, stateName)
	case "func":
		return e.executeFuncStep(step, stateName)
	case "mcp":
		return e.executeMCPStep(step, stateName)
	default:
		return fmt.Errorf("unknown step type: %s", step.Type)
	}
}

func (e *flowExecutorImpl) executeShellStep(step *flows.Step, stateName string) error {
	// Substitute template variables in command
	cmd := e.substituteTemplate(step.Cmd)

	// Create bash tool
	agentID := uuid.New() // Use a dummy agent ID for shell steps
	bashTool := e.bashToolProvider.CreateTool(agentID)

	// Execute command
	ctx := context.Background()
	args := map[string]any{
		"command": cmd,
	}

	// Parse timeout if specified
	if step.Timeout != "" {
		if timeout, err := time.ParseDuration(step.Timeout); err == nil {
			args["timeout"] = timeout.Seconds()
		}
	}

	result, err := bashTool.Run(ctx, args)
	if err != nil {
		return fmt.Errorf("bash tool execution: %w", err)
	}

	// Map outputs
	if step.Output != nil {
		// Handle simple assign
		if step.Output.Assign != "" {
			fieldName := extractFieldNameForService(step.Output.Assign)
			if val, ok := result["stdout"]; ok {
				e.ctx.SetOutputField(fieldName, val)
			}
		}
		// Handle path-based outputs
		for _, path := range step.Output.Paths {
			fieldName := extractFieldNameForService(path.Assign)
			switch path.Path {
			case "stdout":
				if val, ok := result["stdout"]; ok {
					e.ctx.SetOutputField(fieldName, val)
				}
			case "stderr":
				if val, ok := result["stderr"]; ok {
					e.ctx.SetOutputField(fieldName, val)
				}
			case "exit_code":
				// Bash tool returns exit_code as a number
				if val, ok := result["exit_code"]; ok {
					// Convert to int based on type
					switch v := val.(type) {
					case int:
						e.ctx.SetOutputField(fieldName, v)
					case float64:
						e.ctx.SetOutputField(fieldName, int(v))
					case string:
						if code, err := strconv.Atoi(v); err == nil {
							e.ctx.SetOutputField(fieldName, code)
						}
					}
				}
			}
		}
	}

	// Don't error on non-zero exit codes - the flow can check exit_code
	return nil
}

// substituteTemplate replaces ${input.field}, ${output.field}, ${context.field} placeholders
func (e *flowExecutorImpl) substituteTemplate(cmd string) string {
	result := cmd

	// Build scope for template substitution
	scope := e.ctx.buildScope()

	// Replace input references
	if inputScope, ok := scope["input"].(map[string]any); ok {
		for k, v := range inputScope {
			placeholder := fmt.Sprintf("${input.%s}", k)
			result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", v))
		}
	}

	// Replace context references
	if ctxScope, ok := scope["context"].(map[string]any); ok {
		for k, v := range ctxScope {
			placeholder := fmt.Sprintf("${context.%s}", k)
			result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", v))
		}
	}

	// Replace output references
	if outScope, ok := scope["output"].(map[string]any); ok {
		for k, v := range outScope {
			placeholder := fmt.Sprintf("${output.%s}", k)
			result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", v))
		}
	}

	return result
}

func extractFieldNameForService(assign string) string {
	// Remove ${output. or ${context. prefix
	assign = strings.TrimPrefix(assign, "${output.")
	assign = strings.TrimPrefix(assign, "${context.")
	// Remove trailing }
	return strings.TrimSuffix(assign, "}")
}

func (e *flowExecutorImpl) executeLLMStep(step *flows.Step, stateName string) error {
	// For now, create a temporary Executor to use the existing LLM step
	oldExec := &Executor{
		flow: e.flow,
		ctx:  e.ctx,
	}
	// Use the existing llm step implementation
	// The executeLLMStepOriginal is in llm_step.go
	return oldExec.executeLLMStep(step, stateName)
}

func (e *flowExecutorImpl) executeFuncStep(step *flows.Step, stateName string) error {
	// Get the built-in registry
	reg := registry.GetBuiltinRegistry()

	// Build args map from step params with template substitution
	args := make(map[string]any)
	for _, param := range step.Params {
		// Substitute template variables in parameter value
		value := e.substituteTemplate(param.Value)
		args[param.Name] = value
	}

	// Execute the function
	result, err := reg.Execute(step.Function, args)
	if err != nil {
		return fmt.Errorf("func execution: %w", err)
	}

	// Map result to output field
	if step.Output != nil {
		if step.Output.Assign != "" {
			fieldName := extractFieldNameForService(step.Output.Assign)
			e.ctx.SetOutputField(fieldName, result)
		}
	}

	return nil
}

func (e *flowExecutorImpl) executeMCPStep(step *flows.Step, stateName string) error {
	return fmt.Errorf("mcp step execution not yet implemented")
}

func (e *flowExecutorImpl) executeCall(call *flows.Call, stateName string) error {
	// Check if registry is available
	if e.flowRegistry == nil {
		return fmt.Errorf("flow registry not configured - cannot execute call step")
	}

	// Look up the sub-flow
	subFlow, err := e.flowRegistry.GetFlow(call.Ref)
	if err != nil {
		return fmt.Errorf("flow lookup failed for %s: %w", call.Ref, err)
	}

	// Build input map from call.Input fields with template substitution
	subInput := make(map[string]any)
	for _, field := range call.Input {
		// Substitute template variables in field value
		value := e.substituteTemplate(field.Value)
		subInput[field.Name] = value
	}

	// Create executor for sub-flow
	subExec := NewExecutor(subFlow)
	subExec.SetInput(subInput)

	// Execute the sub-flow
	if err := subExec.Run(); err != nil {
		return fmt.Errorf("sub-flow execution failed: %w", err)
	}

	// Map output fields back using call.Output
	for _, field := range call.Output {
		// Get the value from sub-flow output
		fieldName := extractFieldNameForService(field.Value)
		value, ok := subExec.ctx.GetOutputField(fieldName)
		if !ok {
			continue // Skip if field doesn't exist in sub-flow output
		}

		// Set the value in parent flow output
		targetField := extractFieldNameForService(field.Name)
		e.ctx.SetOutputField(targetField, value)
	}

	return nil
}

func (e *flowExecutorImpl) handleError(err error, step *flows.Step, state *flows.State) error {
	// Record error in history
	stepName := "unknown"
	stepType := "unknown"
	if step != nil {
		stepName = step.Name
		stepType = step.Type
	}
	e.history.RecordError(stepName, stepType, err.Error(), time.Now())
	return err
}
