// Package tools provides flow executor tools for setting output fields,
// context fields, getting context, emitting logs, and transitioning states.
// These tools are used within LLM steps during flow execution.
package tools

import (
	"context"
	"fmt"

	"github.com/denkhaus/gollum/pkg/hooks"
	"github.com/denkhaus/gollum/pkg/logger"
	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/google/uuid"
	"github.com/m-mizutani/gollem"
	"github.com/samber/do/v2"
)

type (

	// setOutputFieldTool sets an output field value during flow execution
	setOutputFieldTool struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		agentID     uuid.UUID
		flowCtx     shared.FlowContext
	}

	// setContextFieldTool sets a context field value during flow execution
	setContextFieldTool struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		agentID     uuid.UUID
		flowCtx     shared.FlowContext
	}

	// getContextTool retrieves context fields during flow execution
	getContextTool struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		agentID     uuid.UUID
		flowCtx     shared.FlowContext
	}

	// emitLogTool emits log messages during flow execution
	emitLogTool struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		agentID     uuid.UUID
		flowCtx     shared.FlowContext
	}

	// transitionToTool transitions to a new state during flow execution
	transitionToTool struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		agentID     uuid.UUID
		flowCtx     shared.FlowContext
	}

	flowToolsProvider struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
	}

	// FlowToolsProvider creates flow executor tools via DI
	FlowToolsProvider interface {
		CreateTool(agentID uuid.UUID, flowCtx shared.FlowContext, toolName shared.ToolName) (gollem.Tool, error)
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
func (p *flowToolsProvider) CreateTool(agentID uuid.UUID, flowCtx shared.FlowContext, toolName shared.ToolName) (gollem.Tool, error) {
	switch toolName {
	case shared.ToolNameSetOutputField:
		return &setOutputFieldTool{
			logService:  p.logService,
			hookManager: p.hookManager,
			agentID:     agentID,
			flowCtx:     flowCtx,
		}, nil
	case shared.ToolNameSetContextField:
		return &setContextFieldTool{
			logService:  p.logService,
			hookManager: p.hookManager,
			agentID:     agentID,
			flowCtx:     flowCtx,
		}, nil
	case shared.ToolNameGetContext:
		return &getContextTool{
			logService:  p.logService,
			hookManager: p.hookManager,
			agentID:     agentID,
			flowCtx:     flowCtx,
		}, nil
	case shared.ToolNameEmitLog:
		return &emitLogTool{
			logService:  p.logService,
			hookManager: p.hookManager,
			agentID:     agentID,
			flowCtx:     flowCtx,
		}, nil
	case shared.ToolNameTransitionTo:
		return &transitionToTool{
			logService:  p.logService,
			hookManager: p.hookManager,
			agentID:     agentID,
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
	return t.hookManager.WithToolHooks(ctx, uuid.Nil, t.agentID, shared.ToolNameSetOutputField, args,
		func() (map[string]any, error) {
			return t.runSetOutputField(ctx, args)
		})
}

func (t *setOutputFieldTool) runSetOutputField(ctx context.Context, args map[string]any) (map[string]any, error) {
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("field name is required")
	}

	value, ok := args["value"].(string)
	if !ok {
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
	return t.hookManager.WithToolHooks(ctx, uuid.Nil, t.agentID, shared.ToolNameSetContextField, args,
		func() (map[string]any, error) {
			return t.runSetContextField(ctx, args)
		})
}

func (t *setContextFieldTool) runSetContextField(ctx context.Context, args map[string]any) (map[string]any, error) {
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("field name is required")
	}

	value, ok := args["value"].(string)
	if !ok {
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
	return t.hookManager.WithToolHooks(ctx, uuid.Nil, t.agentID, shared.ToolNameGetContext, args,
		func() (map[string]any, error) {
			return t.runGetContext(ctx, args)
		})
}

func (t *getContextTool) runGetContext(ctx context.Context, args map[string]any) (map[string]any, error) {
	fields, _ := args["fields"].([]any)

	result := make(map[string]any)
	if len(fields) == 0 {
		// Return all context fields
		result = t.flowCtx.GetAllContextFields()
	} else {
		for _, f := range fields {
			if fieldName, ok := f.(string); ok {
				if val, err := t.flowCtx.GetContextField(fieldName); err == nil {
					result[fieldName] = val
				}
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
	return t.hookManager.WithToolHooks(ctx, uuid.Nil, t.agentID, shared.ToolNameEmitLog, args,
		func() (map[string]any, error) {
			return t.runEmitLog(ctx, args)
		})
}

func (t *emitLogTool) runEmitLog(ctx context.Context, args map[string]any) (map[string]any, error) {
	level, _ := args["level"].(string)
	message, _ := args["message"].(string)

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
			t.logService.Debugf(logMsg)
		case "info":
			t.logService.Infof(logMsg)
		case "warn":
			t.logService.Warnf(logMsg)
		case "error":
			t.logService.Errorf(logMsg)
		default:
			t.logService.Infof(logMsg)
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
	return t.hookManager.WithToolHooks(ctx, uuid.Nil, t.agentID, shared.ToolNameTransitionTo, args,
		func() (map[string]any, error) {
			return t.runTransitionTo(ctx, args)
		})
}

func (t *transitionToTool) runTransitionTo(ctx context.Context, args map[string]any) (map[string]any, error) {
	toState, ok := args["to"].(string)
	if !ok || toState == "" {
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
		t.logService.Debugf(msg)
	}

	return map[string]any{
		"success": true,
		"from":    fromState,
		"to":      toState,
	}, nil
}
