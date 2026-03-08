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
	"github.com/google/uuid"
)

// BashToolRunner is the interface for running bash commands
// This interface is defined here to avoid import cycles when mocking
type BashToolRunner interface {
	Run(ctx context.Context, args map[string]any) (map[string]any, error)
}

// BashToolProvider interface for creating bash tools
type BashToolProvider interface {
	CreateTool(agentID uuid.UUID) BashToolRunner
}

// Executor executes flow state machines
type Executor struct {
	flow             *flows.Flow
	ctx              *Context
	currentState     string
	history          *ExecutionHistory
	startTime        time.Time
	bashToolProvider BashToolProvider
	flowRegistry     FlowRegistry
	extService       extensions.ExtensionService
}

// NewExecutor creates a new executor (without bash tool provider for tests)
func NewExecutor(flow *flows.Flow) *Executor {
	return NewExecutorWithProvider(flow, nil)
}

// NewExecutorWithProvider creates a new executor with bash tool provider
func NewExecutorWithProvider(flow *flows.Flow, provider BashToolProvider) *Executor {
	exec := &Executor{
		flow:            flow,
		ctx:             NewContext(flow.Input, nil),
		history:         NewExecutionHistory(),
		startTime:       time.Now(),
		bashToolProvider: provider,
	}

	// Find and set initial state
	for _, state := range flow.States {
		if state.Initial {
			exec.currentState = state.Name
			break
		}
	}

	return exec
}

// NewExecutorWithRegistry creates a new executor with flow registry (for call steps)
func NewExecutorWithRegistry(flow *flows.Flow, registry FlowRegistry) *Executor {
	exec := &Executor{
		flow:         flow,
		ctx:          NewContext(flow.Input, nil),
		history:      NewExecutionHistory(),
		startTime:    time.Now(),
		flowRegistry: registry,
	}

	// Find and set initial state
	for _, state := range flow.States {
		if state.Initial {
			exec.currentState = state.Name
			break
		}
	}

	return exec
}

// NewExecutorWithExtensions creates a new executor with extension service (for func steps)
func NewExecutorWithExtensions(flow *flows.Flow, extService extensions.ExtensionService) *Executor {
	exec := &Executor{
		flow:       flow,
		ctx:        NewContext(flow.Input, nil),
		history:    NewExecutionHistory(),
		startTime:  time.Now(),
		extService: extService,
	}

	// Find and set initial state
	for _, state := range flow.States {
		if state.Initial {
			exec.currentState = state.Name
			break
		}
	}

	return exec
}

// SetInput sets input field values
func (e *Executor) SetInput(vals map[string]any) {
	e.ctx = NewContext(e.flow.Input, vals)
}

// Validate validates the flow before execution
func (e *Executor) Validate() error {
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
func (e *Executor) Run() error {
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

// executeState executes a single state
func (e *Executor) executeState(state *flows.State) error {
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
func (e *Executor) executeTransition(state *flows.State) error {
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
func (e *Executor) transitionTo(stateName string) error {
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
func (e *Executor) executeStep(step *flows.Step, stateName string) error {
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

func (e *Executor) executeShellStep(step *flows.Step, stateName string) error {
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
			fieldName := extractFieldName(step.Output.Assign)
			if val, ok := result["stdout"]; ok {
				e.ctx.SetOutputField(fieldName, val)
			}
		}
		// Handle path-based outputs
		for _, path := range step.Output.Paths {
			fieldName := extractFieldName(path.Assign)
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
func (e *Executor) substituteTemplate(cmd string) string {
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

// extractFieldName extracts the field name from ${output.field_name} or ${context.field_name}
func extractFieldName(assign string) string {
	// Remove ${output. or ${context. prefix
	assign = strings.TrimPrefix(assign, "${output.")
	assign = strings.TrimPrefix(assign, "${context.")
	// Remove trailing }
	return strings.TrimSuffix(assign, "}")
}

func (e *Executor) executeFuncStep(step *flows.Step, stateName string) error {
	// Check if extension service is available
	if e.extService == nil {
		// Fall back to built-in registry for backwards compatibility
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
			return &FuncError{
				Function: step.Function,
				Step:     stateName,
				Err:      err,
			}
		}

		// Map result to output field
		if step.Output != nil {
			if step.Output.Assign != "" {
				fieldName := extractFieldName(step.Output.Assign)
				e.ctx.SetOutputField(fieldName, result)
			}
		}

		return nil
	}

	// Use Scriggo runner from extension service
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

func (e *Executor) executeMCPStep(step *flows.Step, stateName string) error {
	return fmt.Errorf("mcp step execution not yet implemented")
}

// executeCall executes a call step (placeholder)
func (e *Executor) executeCall(call *flows.Call, stateName string) error {
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
		fieldName := extractFieldName(field.Value)
		value, ok := subExec.ctx.GetOutputField(fieldName)
		if !ok {
			continue // Skip if field doesn't exist in sub-flow output
		}

		// Set the value in parent flow output
		targetField := extractFieldName(field.Name)
		e.ctx.SetOutputField(targetField, value)
	}

	return nil
}
