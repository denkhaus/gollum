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

// flowExecutorImpl is the private implementation of a flow executor instance
type flowExecutorImpl struct {
	flow             *flows.Flow
	ctx              *Context
	currentState     string
	history          *ExecutionHistory
	startTime        time.Time
	bashToolProvider tools.BashToolProvider
	extService       extensions.ExtensionService
	flowRegistry     flowregistry.FlowRegistry
	hookManager      hooks.HookManager
}

// flowExecutorServiceImpl is the DI service that creates executor instances
type flowExecutorServiceImpl struct {
	bashToolProvider tools.BashToolProvider
	extService       extensions.ExtensionService
	flowRegistry     flowregistry.FlowRegistry
	hookManager      hooks.HookManager
}

// Ensure flowExecutorServiceImpl implements FlowExecutorService
var _ FlowExecutorService = (*flowExecutorServiceImpl)(nil)

// Ensure flowExecutorImpl implements FlowExecutorInstance
var _ FlowExecutorInstance = (*flowExecutorImpl)(nil)

// NewFlowExecutor creates the flow executor service (DI constructor)
func NewFlowExecutor(injector do.Injector) (FlowExecutorService, error) {
	bashToolProvider := do.MustInvoke[tools.BashToolProvider](injector)
	extService := do.MustInvoke[extensions.ExtensionService](injector)
	flowRegistry := do.MustInvoke[flowregistry.FlowRegistry](injector)
	hookManager := do.MustInvoke[hooks.HookManager](injector)

	return &flowExecutorServiceImpl{
		bashToolProvider: bashToolProvider,
		extService:       extService,
		flowRegistry:     flowRegistry,
		hookManager:      hookManager,
	}, nil
}

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
		hookManager:      p.hookManager,
	}
}

// SetInput sets input field values
func (p *flowExecutorImpl) SetInput(vals map[string]any) {
	p.ctx = NewContext(p.flow.Input, vals)
}

// Validate validates the flow before execution
func (p *flowExecutorImpl) Validate() error {
	// Check for initial state
	hasInitial := false
	for _, state := range p.flow.States {
		if state.Initial {
			hasInitial = true
			p.currentState = state.Name
			break
		}
	}

	if !hasInitial {
		return fmt.Errorf("flow %s: no initial state defined", p.flow.Name)
	}

	return nil
}

// Run executes the flow from the initial state
func (p *flowExecutorImpl) Run() error {
	defer p.history.Complete(time.Now())

	if err := p.Validate(); err != nil {
		return err
	}

	// Find initial state
	var initialState *flows.State
	for i := range p.flow.States {
		if p.flow.States[i].Initial {
			initialState = &p.flow.States[i]
			break
		}
	}

	if initialState == nil {
		return fmt.Errorf("no initial state found")
	}

	return p.executeState(initialState)
}

// GetContext returns the execution context (for testing)
func (p *flowExecutorImpl) GetContext() *Context {
	return p.ctx
}

// executeState executes a single state
func (p *flowExecutorImpl) executeState(state *flows.State) error {
	// Record state entry
	p.history.RecordStateEntry(state.Name, time.Now())

	// Evaluate computed fields
	if p.flow.Context != nil {
		if err := p.ctx.EvaluateComputedFields(p.flow.Context); err != nil {
			return fmt.Errorf("computed field evaluation: %w", err)
		}
	}

	// Execute steps
	for _, step := range state.Steps {
		if err := p.executeStep(&step, state.Name); err != nil {
			return p.handleError(err, &step, state)
		}
	}

	// Execute calls
	for _, call := range state.Calls {
		if err := p.executeCall(&call, state.Name); err != nil {
			return p.handleError(err, nil, state)
		}
	}

	// Find and execute transition
	return p.executeTransition(state)
}

// executeTransition evaluates conditions and transitions to next state
func (p *flowExecutorImpl) executeTransition(state *flows.State) error {
	// Build evaluation scope
	scope := p.ctx.buildScope()

	for _, trans := range state.Transitions {
		if trans.Otherwise {
			// Fallback transition
			return p.transitionTo(trans.To)
		}

		if trans.When == "" {
			// Unconditional transition
			return p.transitionTo(trans.To)
		}

		// Evaluate condition
		eval := NewEvaluator()
		result, err := eval.EvaluateExpr(trans.When, scope)
		if err != nil {
			return fmt.Errorf("transition condition: %w", err)
		}

		if boolVal, ok := result.(bool); ok && boolVal {
			return p.transitionTo(trans.To)
		}
	}

	// No transition - terminal state
	p.history.RecordStateExit(state.Name, time.Now())
	return nil
}

// transitionTo transitions to a new state
func (p *flowExecutorImpl) transitionTo(stateName string) error {
	// Find target state
	var targetState *flows.State
	for i := range p.flow.States {
		if p.flow.States[i].Name == stateName {
			targetState = &p.flow.States[i]
			break
		}
	}

	if targetState == nil {
		return fmt.Errorf("state not found: %s", stateName)
	}

	p.currentState = stateName
	return p.executeState(targetState)
}

// executeStep executes a single step
func (p *flowExecutorImpl) executeStep(step *flows.Step, stateName string) error {
	switch step.Type {
	case "llm":
		return p.executeLLMStep(step, stateName)
	case "shell":
		return p.executeShellStep(step, stateName)
	case "func":
		return p.executeFuncStep(step, stateName)
	case "mcp":
		return p.executeMCPStep(step, stateName)
	default:
		return fmt.Errorf("unknown step type: %s", step.Type)
	}
}

func (p *flowExecutorImpl) executeShellStep(step *flows.Step, stateName string) error {
	// Substitute template variables in command
	cmd := p.substituteTemplate(step.Cmd)

	// Create bash tool
	agentID := uuid.New() // Use a dummy agent ID for shell steps
	bashTool := p.bashToolProvider.CreateTool(agentID)

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
				p.ctx.SetOutputField(fieldName, val)
			}
		}
		// Handle path-based outputs
		for _, path := range step.Output.Paths {
			fieldName := extractFieldName(path.Assign)
			switch path.Path {
			case "stdout":
				if val, ok := result["stdout"]; ok {
					p.ctx.SetOutputField(fieldName, val)
				}
			case "stderr":
				if val, ok := result["stderr"]; ok {
					p.ctx.SetOutputField(fieldName, val)
				}
			case "exit_code":
				// Bash tool returns exit_code as a number
				if val, ok := result["exit_code"]; ok {
					// Convert to int based on type
					switch v := val.(type) {
					case int:
						p.ctx.SetOutputField(fieldName, v)
					case float64:
						p.ctx.SetOutputField(fieldName, int(v))
					case string:
						if code, err := strconv.Atoi(v); err == nil {
							p.ctx.SetOutputField(fieldName, code)
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
func (p *flowExecutorImpl) substituteTemplate(cmd string) string {
	result := cmd

	// Build scope for template substitution
	scope := p.ctx.buildScope()

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

func (p *flowExecutorImpl) executeMCPStep(step *flows.Step, stateName string) error {
	return fmt.Errorf("mcp step execution not yet implemented")
}

// executeCall executes a call step (sub-flow invocation)
func (p *flowExecutorImpl) executeCall(call *flows.Call, stateName string) error {
	// Look up the sub-flow
	subFlow, err := p.flowRegistry.GetFlow(call.Ref)
	if err != nil {
		return fmt.Errorf("flow lookup failed for %s: %w", call.Ref, err)
	}

	// Build input map from call.Input fields with template substitution
	subInput := make(map[string]any)
	for _, field := range call.Input {
		// Substitute template variables in field value
		value := p.substituteTemplate(field.Value)
		subInput[field.Name] = value
	}

	// Create executor for sub-flow using the service
	subExec := &flowExecutorImpl{
		flow:             subFlow,
		ctx:              NewContext(subFlow.Input, subInput),
		history:          NewExecutionHistory(),
		startTime:        time.Now(),
		bashToolProvider: p.bashToolProvider,
		extService:       p.extService,
		flowRegistry:     p.flowRegistry,
		hookManager:      p.hookManager,
	}

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
		p.ctx.SetOutputField(targetField, value)
	}

	return nil
}

// handleError records error in history and returns it
func (p *flowExecutorImpl) handleError(err error, step *flows.Step, state *flows.State) error {
	// Try to handle with on-error transition if available
	if step != nil && step.OnError != nil {
		return p.handleErrorWithErrorTransition(err, step, state)
	}

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
