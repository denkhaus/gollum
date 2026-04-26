// Package tools provides flow executor tools for setting output fields,
// context fields, getting context, emitting logs, and transitioning states.
// These tools are used within LLM steps during flow execution.
package tools

import (
	"context"
	"fmt"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/registry"
	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
	"go.uber.org/zap"
)

type (

	// setOutputFieldTool sets an output field value during flow execution
	setOutputFieldTool struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		agent       shared.Agent
		flowCtx     flows.FlowContext
	}

	// setContextFieldTool sets a context field value during flow execution
	setContextFieldTool struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		agent       shared.Agent
		flowCtx     flows.FlowContext
	}

	// getContextTool retrieves context fields during flow execution
	getContextTool struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		agent       shared.Agent
		flowCtx     flows.FlowContext
	}

	// emitLogTool emits log messages during flow execution
	emitLogTool struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		agent       shared.Agent
		flowCtx     flows.FlowContext
	}

	// transitionToTool transitions to a new state during flow execution
	transitionToTool struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		agent       shared.Agent
		flowCtx     flows.FlowContext
	}

	flowToolsProvider struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
	}

	// FlowToolsProvider creates flow executor tools via DI
	FlowToolsProvider interface {
		CreateTool(agent shared.Agent, flowCtx flows.FlowContext, toolName shared.ToolName) (gollem.Tool, error)
	}
)

// NewFlowToolsProvider creates a provider for flow executor tools
func NewFlowToolsProvider(injector do.Injector) (FlowToolsProvider, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	hookManager := do.MustInvoke[hooks.HookManager](injector)
	return &flowToolsProvider{
		logService:  logService,
		hookManager: hookManager,
	}, nil
}

// CreateTool creates a flow executor tool with the given context
func (p *flowToolsProvider) CreateTool(
	agent shared.Agent,
	flowCtx flows.FlowContext,
	toolName shared.ToolName,
) (gollem.Tool, error) {
	switch toolName {
	case shared.ToolNameSetOutputField:
		return &setOutputFieldTool{
			logService:  p.logService,
			hookManager: p.hookManager,
			agent:       agent,
			flowCtx:     flowCtx,
		}, nil
	case shared.ToolNameSetContextField:
		return &setContextFieldTool{
			logService:  p.logService,
			hookManager: p.hookManager,
			agent:       agent,
			flowCtx:     flowCtx,
		}, nil
	case shared.ToolNameGetContext:
		return &getContextTool{
			logService:  p.logService,
			hookManager: p.hookManager,
			agent:       agent,
			flowCtx:     flowCtx,
		}, nil
	case shared.ToolNameEmitLog:
		return &emitLogTool{
			logService:  p.logService,
			hookManager: p.hookManager,
			agent:       agent,
			flowCtx:     flowCtx,
		}, nil
	case shared.ToolNameTransitionTo:
		return &transitionToTool{
			logService:  p.logService,
			hookManager: p.hookManager,
			agent:       agent,
			flowCtx:     flowCtx,
		}, nil
	default:
		return nil, fmt.Errorf("unknown flow tool: %s", toolName)
	}
}

// Spec returns the tool specification for SetOutputField
func (t *setOutputFieldTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        shared.ToolNameSetOutputField.String(),
		Description: "Sets an output field value in the flow. The value will be validated against the field type if defined in the flow schema.",
		Parameters: map[string]*gollem.Parameter{
			"name": {
				Type:        gollem.TypeString,
				Description: "The name of the output field to set",
			},
			"value": {
				Type:        gollem.TypeString,
				Description: "The value to set (will be type-validated if field is defined in flow schema). Use JSON string representation for complex types.",
			},
		},
	}
}

// Run executes the SetOutputField tool
func (t *setOutputFieldTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, t.agent.ToSessionContext(), shared.ToolNameSetOutputField, args,
		func() (map[string]any, error) {
			return t.runSetOutputField(ctx, args)
		})
}

func (t *setOutputFieldTool) runSetOutputField(ctx context.Context, args ToolRequestParams) (map[string]any, error) {
	name, errResp := args.MustGetString(shared.ParamAgentName)
	if errResp != nil {
		return nil, fmt.Errorf("field name is required")
	}

	value, errResp := args.MustGetString(shared.ParamValue)
	if errResp != nil {
		return nil, fmt.Errorf("field value must be a string")
	}

	if err := t.flowCtx.SetOutputField(name, value); err != nil {
		return nil, err
	}

	// Use enriched logging if flow/step context is available
	if fc := hooks.GetFlowStepContext(ctx); fc != nil {
		t.logService.DebugWithFlowStep(fmt.Sprintf("set_output_field('%s', %v)", name, value),
			fc.FlowName, fc.StateName, fc.StepType)
	} else {
		t.logService.Debugf("set_output_field('%s', %v)", name, value)
	}

	return map[string]any{"success": true}, nil
}

// Spec returns the tool specification for SetContextField
func (t *setContextFieldTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        shared.ToolNameSetContextField.String(),
		Description: "Sets a context field value in the flow. Computed fields cannot be modified.",
		Parameters: map[string]*gollem.Parameter{
			"name": {
				Type:        gollem.TypeString,
				Description: "The name of the context field to set",
			},
			"value": {
				Type:        gollem.TypeString,
				Description: "The value to set (will be converted to appropriate type). Use JSON string representation for complex types.",
			},
		},
	}
}

// Run executes the SetContextField tool
func (t *setContextFieldTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, t.agent.ToSessionContext(), shared.ToolNameSetContextField, args,
		func() (map[string]any, error) {
			return t.runSetContextField(ctx, args)
		})
}

func (t *setContextFieldTool) runSetContextField(ctx context.Context, args ToolRequestParams) (map[string]any, error) {
	name, errResp := args.MustGetString(shared.ParamAgentName)
	if errResp != nil {
		return nil, fmt.Errorf("field name is required")
	}

	value, errResp := args.MustGetString(shared.ParamValue)
	if errResp != nil {
		return nil, fmt.Errorf("field value must be a string")
	}

	if err := t.flowCtx.SetContextField(name, value); err != nil {
		return nil, err
	}

	// Use enriched logging if flow/step context is available
	if fc := hooks.GetFlowStepContext(ctx); fc != nil {
		t.logService.InfoWithFlowStep(fmt.Sprintf("set_context_field('%s', %v)", name, value),
			fc.FlowName, fc.StateName, fc.StepType)
	} else {
		t.logService.Infof("set_context_field('%s', %v)", name, value)
	}

	return map[string]any{"success": true}, nil
}

// Spec returns the tool specification for GetContext
func (t *getContextTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        shared.ToolNameGetContext.String(),
		Description: "Retrieves context field values from the flow. If no fields are specified, returns all context fields.",
		Parameters: map[string]*gollem.Parameter{
			"fields": {
				Type:        gollem.TypeArray,
				Description: "Optional list of field names to retrieve. If omitted, returns all context fields.",
			},
		},
	}
}

// Run executes the GetContext tool
func (t *getContextTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, t.agent.ToSessionContext(), shared.ToolNameGetContext, args,
		func() (map[string]any, error) {
			return t.runGetContext(ctx, args)
		})
}

func (t *getContextTool) runGetContext(ctx context.Context, args ToolRequestParams) (map[string]any, error) {
	fields := args.GetStringSlice(shared.ParamFields)

	result := make(map[string]any)
	if len(fields) == 0 {
		// Return all context fields
		result = t.flowCtx.GetAllContextFields()
	} else {
		for _, fieldName := range fields {
			if val, err := t.flowCtx.GetContextField(fieldName); err == nil {
				result[fieldName] = val
			}
		}
	}

	// Use enriched logging if flow/step context is available
	if fc := hooks.GetFlowStepContext(ctx); fc != nil {
		t.logService.DebugWithFlowStep("get_context_field()", fc.FlowName, fc.StateName, fc.StepType)
	} else {
		t.logService.Debugf("get_context_field()")
	}

	return result, nil
}

// Spec returns the tool specification for EmitLog
func (t *emitLogTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        shared.ToolNameEmitLog.String(),
		Description: "Emits a log message at the specified level. Useful for debugging and tracking flow execution progress.",
		Parameters: map[string]*gollem.Parameter{
			"level": {
				Type:        gollem.TypeString,
				Description: "Log level: debug, info, warn, or error. Defaults to 'info'.",
			},
			"message": {
				Type:        gollem.TypeString,
				Description: "The log message to emit",
			},
		},
	}
}

// Run executes the EmitLog tool
func (t *emitLogTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, t.agent.ToSessionContext(), shared.ToolNameEmitLog, args,
		func() (map[string]any, error) {
			return t.runEmitLog(ctx, args)
		})
}

func (t *emitLogTool) runEmitLog(ctx context.Context, args ToolRequestParams) (map[string]any, error) {
	level := args.GetString(shared.ParamLevel, "")
	message := args.GetString(shared.ParamMessage, "")

	if level == "" {
		level = "info"
	}

	// Use enriched logging if flow/step context is available
	logMsg := fmt.Sprintf("emit_log('%s', '%s')", level, message)
	fc := hooks.GetFlowStepContext(ctx)
	if fc != nil {
		switch level {
		case "debug":
			t.logService.DebugWithFlowStep(logMsg, fc.FlowName, fc.StateName, fc.StepType)
		case "info":
			t.logService.InfoWithFlowStep(logMsg, fc.FlowName, fc.StateName, fc.StepType)
		case "warn":
			t.logService.WarnWithFlowStep(logMsg, fc.FlowName, fc.StateName, fc.StepType)
		case "error":
			t.logService.ErrorWithFlowStep(logMsg, fc.FlowName, fc.StateName, fc.StepType)
		default:
			t.logService.InfoWithFlowStep(logMsg, fc.FlowName, fc.StateName, fc.StepType)
		}
	} else {
		// Fall back to regular logging
		switch level {
		case "debug":
			t.logService.Debug("LogTool: " + message)
		case "info":
			t.logService.Info("LogTool: " + message)
		case "warn":
			t.logService.Warn("LogTool: " + message)
		case "error":
			t.logService.Error("LogTool: " + message)
		default:
			t.logService.Info("LogTool: " + message)
		}
	}

	return map[string]any{"success": true}, nil
}

// Spec returns the tool specification for TransitionTo
func (t *transitionToTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        shared.ToolNameTransitionTo.String(),
		Description: "Transitions to a new state in the flow. The transition must be valid from the current state according to the flow definition.",
		Parameters: map[string]*gollem.Parameter{
			"to": {
				Type:        gollem.TypeString,
				Description: "The name of the state to transition to",
			},
		},
	}
}

// Run executes the TransitionTo tool
func (t *transitionToTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	return t.hookManager.WithToolHooks(ctx, t.agent.ToSessionContext(), shared.ToolNameTransitionTo, args,
		func() (map[string]any, error) {
			return t.runTransitionTo(ctx, args)
		})
}

func (t *transitionToTool) runTransitionTo(ctx context.Context, args ToolRequestParams) (map[string]any, error) {
	toState, errResp := args.MustGetString(shared.ParamTo)
	if errResp != nil {
		return nil, fmt.Errorf("target state is required")
	}

	fromState := t.flowCtx.GetCurrentState()

	// Validate transition is allowed
	if err := t.flowCtx.ValidateTransition(fromState, toState); err != nil {
		return nil, err
	}

	// Request the transition - executor will handle it after step completes
	if err := t.flowCtx.RequestTransition(toState); err != nil {
		return nil, fmt.Errorf("failed to request transition to '%s': %w", toState, err)
	}

	// Use enriched logging if flow/step context is available
	msg := fmt.Sprintf("transition_to('%s', '%s')", fromState, toState)
	if fc := hooks.GetFlowStepContext(ctx); fc != nil {
		t.logService.DebugWithFlowStep(msg, fc.FlowName, fc.StateName, fc.StepType)
	} else {
		t.logService.Debug("transition_to() called", zap.String("from", fromState), zap.String("to", toState))
	}

	return map[string]any{
		"success": true,
		"from":    fromState,
		"to":      toState,
	}, nil
}

// executeFlowTool executes a Gollum flow with the given inputs
// This is an AGENT tool (not flow-internal) - it follows the edit.go pattern
type executeFlowTool struct {
	logService   logger.LoggerService
	hookManager  hooks.HookManager
	agent        shared.Agent
	flowRegistry registry.FlowRegistry
	executor     flows.Executor
}

// ExecuteFlowToolProvider creates ExecuteFlowTool instances via DI
type ExecuteFlowToolProvider interface {
	CreateTool(agent shared.Agent) gollem.Tool
}

type executeFlowToolProvider struct {
	logService   logger.LoggerService
	hookManager  hooks.HookManager
	flowRegistry registry.FlowRegistry
	executor     flows.Executor
}

// NewExecuteFlowToolProvider creates a provider for ExecuteFlow tools
func NewExecuteFlowToolProvider(injector do.Injector) (ExecuteFlowToolProvider, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	hookManager := do.MustInvoke[hooks.HookManager](injector)
	flowRegistry := do.MustInvoke[registry.FlowRegistry](injector)
	executor := do.MustInvoke[flows.Executor](injector)

	return &executeFlowToolProvider{
		logService:   logService,
		hookManager:  hookManager,
		flowRegistry: flowRegistry,
		executor:     executor,
	}, nil
}

// NewExecuteFlowTool creates a new ExecuteFlowTool for testing
func NewExecuteFlowTool(flowRegistry registry.FlowRegistry, executor flows.Executor, logService logger.LoggerService) *executeFlowTool {
	return &executeFlowTool{
		logService:   logService,
		hookManager:  nil, // No hook manager in tests
		flowRegistry: flowRegistry,
		executor:     executor,
	}
}

// CreateTool creates a new ExecuteFlowTool with agent
func (p *executeFlowToolProvider) CreateTool(agent shared.Agent) gollem.Tool {
	return &executeFlowTool{
		logService:   p.logService,
		hookManager:  p.hookManager,
		agent:        agent,
		flowRegistry: p.flowRegistry,
		executor:     p.executor,
	}
}

// Spec returns the tool specification for the ExecuteFlow tool
func (t *executeFlowTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        shared.ToolNameExecuteFlow.String(),
		Description: "Executes a Gollum flow by name with the provided inputs. Returns the flow's output fields upon completion.",
		Parameters: map[string]*gollem.Parameter{
			"flowName": {
				Type:        gollem.TypeString,
				Description: "The name of the flow to execute",
			},
			"inputs": {
				Type:        gollem.TypeObject,
				Description: "Optional input parameters for the flow (map of field names to values)",
			},
		},
	}
}

// Run executes the ExecuteFlow tool
func (t *executeFlowTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	if t.hookManager != nil {
		return t.hookManager.WithToolHooks(ctx, t.agent.ToSessionContext(), shared.ToolNameExecuteFlow, args,
			func() (map[string]any, error) {
				return t.runExecuteFlow(ctx, args)
			})
	}
	// For tests without hook manager
	return t.runExecuteFlow(ctx, args)
}

// runExecuteFlow implements the core execute flow logic
func (t *executeFlowTool) runExecuteFlow(ctx context.Context, args ToolRequestParams) (map[string]any, error) {
	// Get flow name
	flowName, errResp := args.MustGetString(shared.ParamFlowName)
	if errResp != nil {
		return errResp, nil
	}

	// Log operation start
	logCtx := shared.SessionContext{}
	if t.agent != nil {
		logCtx = t.agent.ToSessionContext()
	}
	t.logService.InfoWithContext("ExecuteFlow operation started",
		logCtx,
		zap.String("flow_name", flowName))

	// Get inputs (optional)
	inputs := args.GetStringMap(shared.ParamInputs)
	if inputs == nil {
		inputs = make(map[string]string)
	}
	// Convert map[string]string to map[string]any
	inputsAny := make(map[string]any, len(inputs))
	for k, v := range inputs {
		inputsAny[k] = v
	}

	// Get flow from registry
	flow, err := t.flowRegistry.GetFlow(flowName)
	if err != nil {
		t.logService.ErrorWithContext("Flow not found",
			logCtx,
			zap.String("flow_name", flowName),
			zap.Error(err))
		return ErrorResponse("flow not found: %s", flowName), nil
	}

	// Execute the flow
	result, err := t.executor.Execute(ctx, flow, inputsAny)
	if err != nil {
		t.logService.ErrorWithContext("Flow execution failed",
			logCtx,
			zap.String("flow_name", flowName),
			zap.Error(err))
		return ErrorResponse("flow execution failed: %v", err), nil
	}

	// Log success
	t.logService.InfoWithContext("ExecuteFlow operation completed successfully",
		logCtx,
		zap.String("flow_name", flowName),
		zap.Int("output_count", len(result.Outputs)))

	// Return success with outputs
	return SuccessResponse(map[string]any{
		"outputs": result.Outputs,
	}), nil
}

// listFlowsTool lists all available flows
// This is an AGENT tool (not flow-internal) - it follows the edit.go pattern
type listFlowsTool struct {
	logService   logger.LoggerService
	hookManager  hooks.HookManager
	agent        shared.Agent
	flowRegistry registry.FlowRegistry
}

// ListFlowsToolProvider creates ListFlowsTool instances via DI
type ListFlowsToolProvider interface {
	CreateTool(agent shared.Agent) gollem.Tool
}

type listFlowsToolProvider struct {
	logService   logger.LoggerService
	hookManager  hooks.HookManager
	flowRegistry registry.FlowRegistry
}

// NewListFlowsToolProvider creates a provider for ListFlows tools
func NewListFlowsToolProvider(injector do.Injector) (ListFlowsToolProvider, error) {
	logService := do.MustInvoke[logger.LoggerService](injector)
	hookManager := do.MustInvoke[hooks.HookManager](injector)
	flowRegistry := do.MustInvoke[registry.FlowRegistry](injector)

	return &listFlowsToolProvider{
		logService:   logService,
		hookManager:  hookManager,
		flowRegistry: flowRegistry,
	}, nil
}

// NewListFlowsTool creates a new ListFlowsTool for testing
func NewListFlowsTool(flowRegistry registry.FlowRegistry, logService logger.LoggerService) *listFlowsTool {
	return &listFlowsTool{
		logService:   logService,
		hookManager:  nil, // No hook manager in tests
		flowRegistry: flowRegistry,
	}
}

// CreateTool creates a new ListFlowsTool with agent
func (p *listFlowsToolProvider) CreateTool(agent shared.Agent) gollem.Tool {
	return &listFlowsTool{
		logService:   p.logService,
		hookManager:  p.hookManager,
		agent:        agent,
		flowRegistry: p.flowRegistry,
	}
}

// Spec returns the tool specification for the ListFlows tool
func (t *listFlowsTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        shared.ToolNameListFlows.String(),
		Description: "Lists all available flows in the registry with their metadata including input/output fields and states.",
		Parameters:  map[string]*gollem.Parameter{},
	}
}

// Run executes the ListFlows tool
func (t *listFlowsTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	if t.hookManager != nil {
		return t.hookManager.WithToolHooks(ctx, t.agent.ToSessionContext(), shared.ToolNameListFlows, args,
			func() (map[string]any, error) {
				return t.runListFlows(ctx, args)
			})
	}
	// For tests without hook manager
	return t.runListFlows(ctx, args)
}

// runListFlows implements the core list flows logic
func (t *listFlowsTool) runListFlows(ctx context.Context, args ToolRequestParams) (map[string]any, error) {
	// Setup logging context (may be empty for tests)
	logCtx := shared.SessionContext{}
	if t.agent != nil {
		logCtx = t.agent.ToSessionContext()
	}

	// Log operation start
	t.logService.InfoWithContext("ListFlows operation started",
		logCtx)

	// Get all flows from registry
	flows, err := t.flowRegistry.ListFlows()
	if err != nil {
		t.logService.ErrorWithContext("Failed to list flows",
			logCtx,
			zap.Error(err))
		return ErrorResponse("failed to list flows: %v", err), nil
	}

	// Log success
	t.logService.InfoWithContext("ListFlows operation completed successfully",
		logCtx,
		zap.Int("flow_count", len(flows)))

	// Return success with flows array
	return SuccessResponse(map[string]any{
		"flows": flows,
	}), nil
}
