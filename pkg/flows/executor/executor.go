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
	"github.com/denkhaus/gollum/pkg/logger"
	mcpregistry "github.com/denkhaus/gollum/pkg/mcp/registry"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/denkhaus/gollum/pkg/tools"
	"github.com/google/uuid"
	"github.com/samber/do/v2"
)

// FlowExecutorInstance defines the interface for a flow executor instance
type FlowExecutorInstance interface {
	// SetInput sets input field values with validation
	SetInput(vals map[string]string) error
	// Validate validates the flow before execution
	Validate() error
	// Run executes the flow from the initial state
	Run() error
	// GetContext returns the execution context (for testing)
	GetContext() ExecutionContext
	// Close releases resources held by the executor
	Close() error
}

// FlowExecutorService defines the DI service that creates executor instances
type FlowExecutorService interface {
	// New creates a new executor instance for a flow
	New(flow *flows.Flow) FlowExecutorInstance
}

// ErrorContext holds error lifecycle information
type ErrorContext struct {
	StepName  string
	StepType  string
	Message   string
	Timestamp time.Time
}

// flowExecutorImpl is the private implementation of a flow executor instance
type flowExecutorImpl struct {
	flow              *flows.Flow
	ctx               *contextImpl
	currentState      string
	history           *ExecutionHistory
	startTime         time.Time
	logService        logger.LoggerService
	bashToolProvider  tools.BashToolProvider
	extService        extensions.ExtensionService
	flowRegistry      flowregistry.FlowRegistry
	hookManager       hooks.HookManager
	flowToolsProvider tools.FlowToolsProvider
	mcpRegistry       mcpregistry.MCPRegistry
	agentFactory      shared.AgentFactory
	pendingTransition string // Set by transition_to tool to force a state transition
}

// Ensure flowExecutorImpl implements tools.FlowContext
var _ flows.FlowContext = (*flowExecutorImpl)(nil)

// flowExecutorServiceImpl is the DI service that creates executor instances
type flowExecutorServiceImpl struct {
	logService        logger.LoggerService
	bashToolProvider  tools.BashToolProvider
	extService        extensions.ExtensionService
	flowRegistry      flowregistry.FlowRegistry
	hookManager       hooks.HookManager
	flowToolsProvider tools.FlowToolsProvider
	mcpRegistry       mcpregistry.MCPRegistry
	agentFactory      shared.AgentFactory
}

// Ensure flowExecutorServiceImpl implements FlowExecutorService
var _ FlowExecutorService = (*flowExecutorServiceImpl)(nil)

// Ensure flowExecutorImpl implements FlowExecutorInstance
var _ FlowExecutorInstance = (*flowExecutorImpl)(nil)

// NewFlowExecutor creates the flow executor service (DI constructor)
func NewFlowExecutor(injector do.Injector) (FlowExecutorService, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	bashToolProvider := do.MustInvoke[tools.BashToolProvider](injector)
	extService := do.MustInvoke[extensions.ExtensionService](injector)
	flowRegistry := do.MustInvoke[flowregistry.FlowRegistry](injector)
	hookManager := do.MustInvoke[hooks.HookManager](injector)
	flowToolsProvider := do.MustInvoke[tools.FlowToolsProvider](injector)
	mcpRegistry := do.MustInvoke[mcpregistry.MCPRegistry](injector)
	agentFactory := do.MustInvoke[shared.AgentFactory](injector)

	return &flowExecutorServiceImpl{
		logService:        logService,
		bashToolProvider:  bashToolProvider,
		extService:        extService,
		flowRegistry:      flowRegistry,
		hookManager:       hookManager,
		flowToolsProvider: flowToolsProvider,
		mcpRegistry:       mcpRegistry,
		agentFactory:      agentFactory,
	}, nil
}

// New creates a new executor instance for a specific flow
func (p *flowExecutorServiceImpl) New(flow *flows.Flow) FlowExecutorInstance {
	ctx := newContext(flow.Input, flow.Output, flow.Context, nil)

	// Initialize computed fields from ComputedBlock
	if flow.Computed != nil {
		ctx.SetComputedBlock(flow.Computed)
	}

	return &flowExecutorImpl{
		flow:              flow,
		ctx:               ctx,
		history:           NewExecutionHistory(),
		startTime:         time.Now(),
		logService:        p.logService,
		bashToolProvider:  p.bashToolProvider,
		extService:        p.extService,
		flowRegistry:      p.flowRegistry,
		hookManager:       p.hookManager,
		flowToolsProvider: p.flowToolsProvider,
		mcpRegistry:       p.mcpRegistry,
		agentFactory:      p.agentFactory,
	}
}

// SetInput sets input field values with validation
func (p *flowExecutorImpl) SetInput(vals map[string]string) error {
	// Build a map with defaults and provided values
	processedVals := make(map[string]string)

	// First, apply defaults for all fields
	if p.flow.Input != nil {
		for _, field := range p.flow.Input.GetAllFields() {
			if field.Default != "" {
				processedVals[field.Name] = field.Default
			}
		}
	}

	// Then, apply provided values and validate
	if p.flow.Input != nil {
		for fieldName, value := range vals {
			// Check if field exists in input definition
			fieldDef := p.findInputField(fieldName)
			if fieldDef == nil {
				return fmt.Errorf("unknown input field: '%s'", fieldName)
			}

			// Validate type
			if err := validateInputType(value, string(fieldDef.Type)); err != nil {
				return fmt.Errorf("invalid input for '%s': %w", fieldName, err)
			}

			processedVals[fieldName] = value
		}

		// Check for missing required fields
		for _, field := range p.flow.Input.GetAllFields() {
			if field.Required && field.Default == "" {
				if _, exists := processedVals[field.Name]; !exists {
					return fmt.Errorf("missing required input: '%s'", field.Name)
				}
			}
		}
	}

	ctx := newContext(p.flow.Input, p.flow.Output, p.flow.Context, processedVals)

	// Initialize computed fields from ComputedBlock
	if p.flow.Computed != nil {
		ctx.SetComputedBlock(p.flow.Computed)
	}

	p.ctx = ctx
	return nil
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
func (p *flowExecutorImpl) GetContext() ExecutionContext {
	return p.ctx
}

// Close releases resources held by the executor
// This method is idempotent - it can be called multiple times safely
func (p *flowExecutorImpl) Close() error {
	// Clear context to release references
	p.ctx = nil

	// Clear history to release references
	if p.history != nil {
		p.history = nil
	}

	// Clear flow reference
	p.flow = nil

	// Reset current state
	p.currentState = ""

	return nil
}

// executeState executes a single state
func (p *flowExecutorImpl) executeState(state *flows.State) error {
	// Record state entry
	p.history.RecordStateEntry(state.Name, time.Now())

	// Log state entry (with nil check for test scenarios)
	if p.logService != nil {
		p.logService.InfoWithFlowStep(fmt.Sprintf("Entering state: %s", state.Name),
			p.flow.Name, state.Name, "")
	}

	// Evaluate computed fields
	if p.flow.Computed != nil {
		if err := p.ctx.EvaluateComputed(); err != nil {
			return fmt.Errorf("computed field evaluation: %w", err)
		}
	}

	// Initialize output bindings from declarative 'from' attributes
	if p.flow.Output != nil {
		binder := NewOutputBinder()
		if err := binder.InitializeBindings(p.ctx, p.flow.Output); err != nil {
			return fmt.Errorf("output binding initialization: %w", err)
		}
	}

	// Execute steps
	for _, step := range state.Steps {
		if err := p.executeStep(&step, state.Name); err != nil {
			return p.handleError(err, &step, state)
		}

		// Check if a tool requested a transition
		if p.pendingTransition != "" {
			return p.transitionTo(p.pendingTransition)
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

	// Log terminal state (with nil check for test scenarios)
	if p.logService != nil {
		p.logService.InfoWithFlowStep(fmt.Sprintf("Reached terminal state: %s", state.Name),
			p.flow.Name, state.Name, "")
	}

	return nil
}

// transitionTo transitions to a new state
func (p *flowExecutorImpl) transitionTo(stateName string) error {
	fromState := p.currentState

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

	// Log state transition (with nil check for test scenarios)
	if p.logService != nil {
		p.logService.InfoWithFlowStep(fmt.Sprintf("Transitioning from '%s' to '%s'", fromState, stateName),
			p.flow.Name, fromState, "")
	}

	p.currentState = stateName
	return p.executeState(targetState)
}

// executeStep executes a single step
func (p *flowExecutorImpl) executeStep(step *flows.Step, stateName string) error {
	// Create context with flow/step metadata for enriched logging
	ctx := hooks.WithFlowStepContext(context.Background(), &hooks.FlowStepContext{
		FlowName:  p.flow.Name,
		StateName: stateName,
		StepType:  step.Type,
	})

	// Wrap step execution with hooks
	_, err := p.hookManager.WithFlowStepHooks(
		ctx,
		uuid.Nil, // SessionID - will be available when executor is used in session context
		uuid.Nil, // FlowID - flows don't have IDs yet, use Nil for now
		p.flow.Name,
		step.Type,
		stateName,
		func() (map[string]any, error) {
			// Execute the actual step logic with enriched context
			switch step.Type {
			case "llm":
				return nil, p.executeLLMStep(ctx, step, stateName)
			case "shell":
				return nil, p.executeShellStep(ctx, step, stateName)
			case "func":
				return nil, p.executeFuncStep(ctx, step, stateName)
			case "mcp":
				return nil, p.executeMCPStep(ctx, step, stateName)
			default:
				return nil, fmt.Errorf("unknown step type: %s", step.Type)
			}
		},
	)

	return err
}

func (p *flowExecutorImpl) executeShellStep(_ context.Context, step *flows.Step, _ string) error {
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
				_ = p.ctx.SetOutputField(fieldName, anyToString(val))
			}
		}
		// Handle path-based outputs
		for _, path := range step.Output.Paths {
			fieldName := extractFieldName(path.Assign)
			switch path.Path {
			case "stdout":
				if val, ok := result["stdout"]; ok {
					_ = p.ctx.SetOutputField(fieldName, anyToString(val))
				}
			case "stderr":
				if val, ok := result["stderr"]; ok {
					_ = p.ctx.SetOutputField(fieldName, anyToString(val))
				}
			case "exit_code":
				// Bash tool returns exit_code as a number
				if val, ok := result["exit_code"]; ok {
					_ = p.ctx.SetOutputField(fieldName, anyToString(val))
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

// extractFieldName extracts the field name from:
// - ${output.field_name} or ${context.field_name} (legacy format)
// - output.field_name or context.field_name (preferred format)
func extractFieldName(assign string) string {
	// Handle legacy ${} format
	assign = strings.TrimPrefix(assign, "${output.")
	assign = strings.TrimPrefix(assign, "${context.")
	assign = strings.TrimSuffix(assign, "}")

	// Handle plain prefix.field format (preferred)
	assign = strings.TrimPrefix(assign, "output.")
	assign = strings.TrimPrefix(assign, "context.")
	assign = strings.TrimPrefix(assign, "input.")
	assign = strings.TrimPrefix(assign, "computed.")

	return assign
}

func (p *flowExecutorImpl) executeFuncStep(_ context.Context, step *flows.Step, stateName string) error {
	// Check for built-in "assign" function
	if step.Function == "assign" {
		return p.executeAssignStep(step, stateName)
	}

	// Use Yaegi runner from extension service for other functions
	funcRunner := p.extService.GetFuncRunner()

	// Build args with template substitution
	args := make(map[string]any)
	for _, param := range step.Params {
		value := p.substituteTemplate(param.Value)
		args[param.Name] = value
	}

	// Execute via Yaegi runner
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
		if err := p.ctx.SetOutputField(fieldName, result); err != nil {
			return fmt.Errorf("failed to set output field '%s': %w", fieldName, err)
		}
	}

	return nil
}

// executeAssignStep handles the built-in assign function
func (p *flowExecutorImpl) executeAssignStep(step *flows.Step, stateName string) error {
	// Parse parameters
	var (
		from  string
		to    string
		value any
	)

	for _, param := range step.Params {
		switch param.Name {
		case "from":
			from = p.substituteTemplate(param.Value)
		case "to":
			to = p.substituteTemplate(param.Value)
		case "value":
			valueStr := p.substituteTemplate(param.Value)
			// Try to parse as int, float, bool
			var intVal int64
			var floatVal float64
			var boolVal bool
			_, intErr := fmt.Sscanf(valueStr, "%d", &intVal)
			_, floatErr := fmt.Sscanf(valueStr, "%f", &floatVal)
			_, boolErr := strconv.ParseBool(valueStr)

			if intErr == nil && len(valueStr) > 0 && valueStr[0] >= '0' && valueStr[0] <= '9' {
				value = intVal
			} else if floatErr == nil && strings.Contains(valueStr, ".") {
				value = floatVal
			} else if boolErr == nil {
				value = boolVal
			} else {
				value = valueStr
			}
		}
	}

	if to == "" {
		return fmt.Errorf("assign: missing 'to' parameter")
	}

	// Determine value to assign
	var valueToAssign any
	if value != nil {
		valueToAssign = value
	} else if from != "" {
		// Parse source: prefix.field
		parts := strings.SplitN(from, ".", 2)
		if len(parts) != 2 {
			return fmt.Errorf("assign: invalid 'from' format: %s (expected prefix.field)", from)
		}

		prefix := parts[0]
		field := parts[1]

		// Read from source
		switch prefix {
		case "input":
			valueToAssign = p.ctx.GetInput(field)
		case "context":
			var err error
			valueToAssign, err = p.ctx.GetContextField(field)
			if err != nil {
				return fmt.Errorf("assign: failed to read context field '%s': %w", field, err)
			}
		case "computed":
			var err error
			valueToAssign, err = p.ctx.GetComputedField(field)
			if err != nil {
				return fmt.Errorf("assign: failed to read computed field '%s': %w", field, err)
			}
		default:
			return fmt.Errorf("assign: unknown prefix '%s' (expected input, context, or computed)", prefix)
		}
	} else {
		return fmt.Errorf("assign: must provide either 'value' or 'from' parameter")
	}

	// Determine destination: output or context
	if strings.HasPrefix(to, "output.") {
		fieldName := strings.TrimPrefix(to, "output.")
		if err := p.ctx.SetOutputField(fieldName, valueToAssign); err != nil {
			return fmt.Errorf("assign: failed to set output field '%s': %w", fieldName, err)
		}
	} else if strings.HasPrefix(to, "context.") {
		fieldName := strings.TrimPrefix(to, "context.")
		if err := p.ctx.SetContextField(fieldName, valueToAssign); err != nil {
			return fmt.Errorf("assign: failed to set context field '%s': %w", fieldName, err)
		}
	} else {
		// Default to output
		if err := p.ctx.SetOutputField(to, valueToAssign); err != nil {
			return fmt.Errorf("assign: failed to set output field '%s': %w", to, err)
		}
	}

	return nil
}

func (p *flowExecutorImpl) executeMCPStep(_ context.Context, step *flows.Step, stateName string) error {
	// step.Tool format: "server.tool" (e.g., "tavily.search")
	// For now, we use the full tool name directly from step.Tool
	toolName := step.Tool

	// Build args from step params with template substitution
	args := make(map[string]any)
	for _, param := range step.Params {
		// Substitute template variables in param value
		value := p.substituteTemplate(param.Value)
		args[param.Name] = value
	}

	// Get all available MCP tool sets
	toolSets := p.mcpRegistry.GetToolSets()

	// Find the tool by name in any of the available tool sets
	for _, toolSet := range toolSets {
		// Get tool specs from this tool set
		ctx := context.Background()
		specs, err := toolSet.Specs(ctx)
		if err != nil {
			continue
		}

		// Check if any spec matches our tool name
		for _, spec := range specs {
			if spec.Name == toolName {
				// Found the tool, execute it
				result, err := toolSet.Run(ctx, toolName, args)
				if err != nil {
					return &MCPError{
						Server: toolName,
						Tool:   toolName,
						Step:   stateName,
						Err:    err,
					}
				}

				// Map result to output fields if specified
				if step.Output != nil {
					// Handle simple assign
					if step.Output.Assign != "" {
						fieldName := extractFieldName(step.Output.Assign)
						// For MCP tools, we'll map the entire result to the field
						if err := p.ctx.SetOutputField(fieldName, anyToString(result)); err != nil {
							return fmt.Errorf("failed to set output field '%s': %w", fieldName, err)
						}
					}
					// TODO: Handle path-based outputs with JSONPath extraction
					// This would allow mapping specific fields from the result
				}

				return nil
			}
		}
	}

	return &MCPError{
		Server: toolName,
		Tool:   toolName,
		Step:   stateName,
		Err:    fmt.Errorf("tool not found"),
	}
}

// executeCall executes a call step (sub-flow invocation)
func (p *flowExecutorImpl) executeCall(call *flows.Call, _ string) error {
	// Check if call has a condition
	if call.When != "" {
		// Evaluate the condition
		scope := p.ctx.buildScope()
		eval := NewEvaluator()
		result, err := eval.EvaluateExpr(call.When, scope)
		if err != nil {
			return fmt.Errorf("call condition evaluation failed: %w", err)
		}

		// Skip call if condition is false
		if boolVal, ok := result.(bool); !ok || !boolVal {
			// Call is skipped - this is not an error, just don't execute
			return nil
		}
	}

	// Look up the sub-flow
	subFlow, err := p.flowRegistry.GetFlow(call.Ref)
	if err != nil {
		return fmt.Errorf("flow lookup failed for %s: %w", call.Ref, err)
	}

	// Build input map from call.Input fields with template substitution
	subInput := make(map[string]string)
	if call.Input != nil {
		for _, field := range call.Input.GetFields() {
			typedField := field.GetTypedField()
			if typedField != nil {
				// Substitute template variables in field value
				value := p.substituteTemplate(typedField.Value)
				subInput[typedField.Name] = value
			}
		}
	}

	// Create executor for sub-flow using the service
	subCtx := newContext(subFlow.Input, subFlow.Output, subFlow.Context, subInput)

	// Initialize computed fields for sub-flow
	if subFlow.Computed != nil {
		subCtx.SetComputedBlock(subFlow.Computed)
	}

	subExec := &flowExecutorImpl{
		flow:             subFlow,
		ctx:              subCtx,
		history:          NewExecutionHistory(),
		startTime:        time.Now(),
		bashToolProvider: p.bashToolProvider,
		extService:       p.extService,
		flowRegistry:     p.flowRegistry,
		hookManager:      p.hookManager,
		agentFactory:     p.agentFactory,
	}

	// Execute the sub-flow
	if err := subExec.Run(); err != nil {
		return fmt.Errorf("sub-flow execution failed: %w", err)
	}

	// Map output fields back using call.Output
	if call.Output != nil {
		for _, field := range call.Output.GetFields() {
			typedField := field.GetTypedField()
			if typedField != nil {
				// Get the value from sub-flow output
				fieldName := extractFieldName(typedField.Value)
				value, err := subExec.ctx.GetOutputField(fieldName)
				if err != nil {
					continue // Skip if field doesn't exist in sub-flow output
				}

				// Set the value in parent flow output
				targetField := extractFieldName(typedField.Name)
				if err := p.ctx.SetOutputField(targetField, anyToString(value)); err != nil {
					return fmt.Errorf("failed to set output field '%s': %w", targetField, err)
				}
			}
		}
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

// FlowContext interface implementation for tool access

// SetOutputField sets an output field value
func (p *flowExecutorImpl) SetOutputField(name string, value any) error {
	return p.ctx.SetOutputField(name, value)
}

// GetOutputField retrieves an output field value
func (p *flowExecutorImpl) GetOutputField(name string) (any, error) {
	return p.ctx.GetOutputField(name)
}

// SetContextField sets a context field value
func (p *flowExecutorImpl) SetContextField(name string, value any) error {
	return p.ctx.SetContextField(name, value)
}

// GetContextField retrieves a context field value
func (p *flowExecutorImpl) GetContextField(name string) (any, error) {
	return p.ctx.GetContextField(name)
}

// GetFlow returns the flow definition
func (p *flowExecutorImpl) GetFlow() *flows.Flow {
	return p.flow
}

// GetCurrentState returns the current state name
func (p *flowExecutorImpl) GetCurrentState() string {
	return p.currentState
}

// GetAllContextFields returns all context fields that have been set
func (p *flowExecutorImpl) GetAllContextFields() map[string]any {
	if p.ctx.contextValues == nil {
		return make(map[string]any)
	}

	result := make(map[string]any)
	// Iterate through all defined fields and get their values if set
	if p.ctx.contextBlock != nil {
		for _, field := range p.ctx.contextBlock.Strings {
			if val, err := p.ctx.contextValues.GetString(field.Name); err == nil {
				result[field.Name] = val
			}
		}
		for _, field := range p.ctx.contextBlock.Ints {
			if val, err := p.ctx.contextValues.GetInt(field.Name); err == nil {
				result[field.Name] = val
			}
		}
		for _, field := range p.ctx.contextBlock.Bools {
			if val, err := p.ctx.contextValues.GetBool(field.Name); err == nil {
				result[field.Name] = val
			}
		}
		for _, field := range p.ctx.contextBlock.Floats {
			if val, err := p.ctx.contextValues.GetFloat(field.Name); err == nil {
				result[field.Name] = val
			}
		}
	}
	return result
}

// ValidateTransition checks if a transition is valid
func (p *flowExecutorImpl) ValidateTransition(from, to string) error {
	// Find the current state and check if transition is allowed
	for _, s := range p.flow.States {
		if s.Name == from {
			// Check if the transition exists
			for _, trans := range s.Transitions {
				if trans.To == to {
					return nil // Transition is valid
				}
			}
			return fmt.Errorf("transition from %s to %s is not defined", from, to)
		}
	}
	return fmt.Errorf("current state '%s' not found", from)
}

// RequestTransition signals that the flow should transition to the target state
func (p *flowExecutorImpl) RequestTransition(to string) error {
	// Validate the target state exists
	var targetState *flows.State
	for i := range p.flow.States {
		if p.flow.States[i].Name == to {
			targetState = &p.flow.States[i]
			break
		}
	}

	if targetState == nil {
		return fmt.Errorf("target state '%s' not found", to)
	}

	// Set the pending transition - executor will execute it after step completes
	p.pendingTransition = to
	return nil
}

// anyToString converts any value to its string representation
func anyToString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case []byte:
		return string(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// validateInputType validates that a string value matches the expected type
func validateInputType(value, typeName string) error {
	switch typeName {
	case string(flows.TypeString):
		// Strings are always valid
		return nil
	case string(flows.TypeInt):
		_, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer: %s", value)
		}
		return nil
	case string(flows.TypeBool):
		_, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean: %s (expected: true, false, 1, or 0)", value)
		}
		return nil
	case string(flows.TypeFloat):
		_, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float: %s", value)
		}
		return nil
	case string(flows.TypeArray), string(flows.TypeMap), string(flows.TypeObject):
		// Complex types are validated as JSON strings
		// For now, accept any string - JSON validation happens during use
		return nil
	default:
		return fmt.Errorf("unknown type: %s", typeName)
	}
}

// findInputField finds a field definition by name in the input block
func (p *flowExecutorImpl) findInputField(name string) *flows.FieldDef {
	if p.flow.Input == nil {
		return nil
	}

	// Search through all field type arrays
	for _, field := range p.flow.Input.Strings {
		if field.Name == name {
			// Ensure Type is set (for programmatically created flows)
			if field.Type == "" {
				field.Type = flows.TypeString
			}
			return &field
		}
	}
	for _, field := range p.flow.Input.Ints {
		if field.Name == name {
			if field.Type == "" {
				field.Type = flows.TypeInt
			}
			return &field
		}
	}
	for _, field := range p.flow.Input.Bools {
		if field.Name == name {
			if field.Type == "" {
				field.Type = flows.TypeBool
			}
			return &field
		}
	}
	for _, field := range p.flow.Input.Floats {
		if field.Name == name {
			if field.Type == "" {
				field.Type = flows.TypeFloat
			}
			return &field
		}
	}
	for _, field := range p.flow.Input.Arrays {
		if field.Name == name {
			if field.Type == "" {
				field.Type = flows.TypeArray
			}
			return &field
		}
	}
	for _, field := range p.flow.Input.Maps {
		if field.Name == name {
			if field.Type == "" {
				field.Type = flows.TypeMap
			}
			return &field
		}
	}
	for _, obj := range p.flow.Input.Objects {
		if obj.Name == name {
			return &flows.FieldDef{
				XMLName:  obj.XMLName,
				Name:     obj.Name,
				Type:     flows.TypeObject,
				Required: false,
				Default:  obj.Default,
			}
		}
	}

	return nil
}
