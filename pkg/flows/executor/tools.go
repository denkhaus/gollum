package executor

import (
	"fmt"

	"github.com/denkhaus/gollum/pkg/flows"
)

// Tool is the interface for executor tools
type Tool interface {
	Execute(input map[string]any) (map[string]any, error)
}

// ToolSetOutputField implements set_output_field
type ToolSetOutputField struct {
	executor *Executor
	flow     *flows.Flow
}

// Execute sets an output field value
func (t *ToolSetOutputField) Execute(input map[string]any) (map[string]any, error) {
	name, _ := input["name"].(string)
	value := input["value"]

	if name == "" {
		return nil, fmt.Errorf("field name is required")
	}

	// Type validation
	if err := t.validateFieldType(name, value); err != nil {
		return nil, err
	}

	t.executor.ctx.SetOutputField(name, value)

	return map[string]any{"success": true}, nil
}

// validateFieldType checks if value matches field type
func (t *ToolSetOutputField) validateFieldType(name string, value any) error {
	if t.flow == nil || t.flow.Output == nil {
		return nil
	}

	for _, field := range t.flow.Output.GetAllFields() {
		if field.Name == name {
			return validateType(field.Type, value)
		}
	}

	// Field not found in output definition - that's okay, just set it
	return nil
}

// validateType checks value type
func validateType(typ string, value any) error {
	switch typ {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "int":
		switch value.(type) {
		case int, int64, float64:
			// Accept numeric types
		default:
			return fmt.Errorf("expected int, got %T", value)
		}
	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected bool, got %T", value)
		}
	}
	return nil
}

// ToolSetContextField implements set_context_field
type ToolSetContextField struct {
	executor *Executor
	flow     *flows.Flow
}

// Execute sets a context field value
func (t *ToolSetContextField) Execute(input map[string]any) (map[string]any, error) {
	name, _ := input["name"].(string)
	value := input["value"]

	if name == "" {
		return nil, fmt.Errorf("field name is required")
	}

	// Check if it's a computed field (immutable)
	if t.flow != nil && t.flow.Context != nil {
		for _, cf := range t.flow.Context.Computeds {
			if cf.Name == name {
				return nil, fmt.Errorf("cannot modify computed field '%s'", name)
			}
		}
	}

	t.executor.ctx.SetContextField(name, value)

	return map[string]any{"success": true}, nil
}

// ToolGetContext implements get_context
type ToolGetContext struct {
	executor *Executor
}

// Execute retrieves context fields
func (t *ToolGetContext) Execute(input map[string]any) (map[string]any, error) {
	fields, _ := input["fields"].([]any)

	result := make(map[string]any)
	if len(fields) == 0 {
		// Return all context fields
		for k, v := range t.executor.ctx.values {
			result[k] = v
		}
	} else {
		for _, f := range fields {
			if fieldName, ok := f.(string); ok {
				if val, ok := t.executor.ctx.GetContextField(fieldName); ok {
					result[fieldName] = val
				}
			}
		}
	}

	return result, nil
}

// ToolEmitLog implements emit_log
type ToolEmitLog struct {
	executor *Executor
}

// Execute logs a message (uses Langfuse hooks)
func (t *ToolEmitLog) Execute(input map[string]any) (map[string]any, error) {
	level, _ := input["level"].(string)
	message, _ := input["message"].(string)

	if level == "" {
		level = "info"
	}

	// TODO: Integrate with Langfuse hooks
	fmt.Printf("[%s] %s\n", level, message)

	return map[string]any{"success": true}, nil
}

// ToolTransitionTo implements transition_to
type ToolTransitionTo struct {
	executor *Executor
}

// Execute transitions to a new state
func (t *ToolTransitionTo) Execute(input map[string]any) (map[string]any, error) {
	toState, _ := input["to"].(string)

	if toState == "" {
		return nil, fmt.Errorf("target state is required")
	}

	// Validate transition is allowed
	currentState := t.executor.currentState
	allowed := false
	for _, s := range t.executor.flow.States {
		if s.Name == currentState {
			for _, trans := range s.Transitions {
				if trans.To == toState {
					allowed = true
					break
				}
			}
			break
		}
	}

	if !allowed {
		return nil, fmt.Errorf("transition from %s to %s is not allowed", currentState, toState)
	}

	return map[string]any{
		"success": true,
		"from":    currentState,
		"to":      toState,
	}, nil
}
