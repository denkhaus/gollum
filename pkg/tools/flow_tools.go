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

	// FlowContext defines the interface for accessing flow execution state
	FlowContext interface {
		// SetOutputField sets an output field value
		SetOutputField(name string, value any) error
		// GetOutputField retrieves an output field value
		GetOutputField(name string) (any, error)
		// SetContextField sets a context field value
		SetContextField(name string, value any) error
		// GetContextField retrieves a context field value
		GetContextField(name string) (any, error)
		// GetCurrentState returns the current state name
		GetCurrentState() string
		// GetAllContextFields returns all context fields
		GetAllContextFields() map[string]any
		// ValidateTransition checks if a transition is allowed
		ValidateTransition(from, to string) error
		// RequestTransition signals that the flow should transition to the target state
		RequestTransition(to string) error
	}

	// setOutputFieldTool sets an output field value during flow execution
	setOutputFieldTool struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		agentID     uuid.UUID
		flowCtx     FlowContext
	}

	// setContextFieldTool sets a context field value during flow execution
	setContextFieldTool struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		agentID     uuid.UUID
		flowCtx     FlowContext
	}

	// getContextTool retrieves context fields during flow execution
	getContextTool struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		agentID     uuid.UUID
		flowCtx     FlowContext
	}

	// emitLogTool emits log messages during flow execution
	emitLogTool struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		agentID     uuid.UUID
		flowCtx     FlowContext
	}

	// transitionToTool transitions to a new state during flow execution
	transitionToTool struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
		agentID     uuid.UUID
		flowCtx     FlowContext
	}

	flowToolsProvider struct {
		logService  logger.LoggerService
		hookManager hooks.HookManager
	}

	// FlowToolsProvider creates flow executor tools via DI
	FlowToolsProvider interface {
		CreateTool(agentID uuid.UUID, flowCtx FlowContext, toolName shared.ToolName) (gollem.Tool, error)
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
func (p *flowToolsProvider) CreateTool(agentID uuid.UUID, flowCtx FlowContext, toolName shared.ToolName) (gollem.Tool, error) {
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

func (t *setOutputFieldTool) runSetOutputField(_ context.Context, args map[string]any) (map[string]any, error) {
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
	t.logService.Debugf("Set output field '%s' = %v", name, value)

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

func (t *setContextFieldTool) runSetContextField(_ context.Context, args map[string]any) (map[string]any, error) {
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
	t.logService.Infof("Set context field '%s' = %v", name, value)

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

func (t *getContextTool) runGetContext(_ context.Context, args map[string]any) (map[string]any, error) {
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

	t.logService.Debugf("Retrieved %d context fields", len(result))
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

func (t *emitLogTool) runEmitLog(_ context.Context, args map[string]any) (map[string]any, error) {
	level, _ := args["level"].(string)
	message, _ := args["message"].(string)

	if level == "" {
		level = "info"
	}

	switch level {
	case "debug":
		t.logService.Debugf(message)
	case "info":
		t.logService.Infof(message)
	case "warn":
		t.logService.Warnf(message)
	case "error":
		t.logService.Errorf(message)
	default:
		t.logService.Infof(message)
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

func (t *transitionToTool) runTransitionTo(_ context.Context, args map[string]any) (map[string]any, error) {
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

	t.logService.Debugf("Requested transition from '%s' to '%s'", fromState, toState)

	return map[string]any{
		"success": true,
		"from":    fromState,
		"to":      toState,
	}, nil
}
