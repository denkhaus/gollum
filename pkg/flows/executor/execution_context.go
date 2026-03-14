package executor

import (
	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/go-ap/errors"
)

// ExecutionContext manages execution context with input, output, and computed fields
type ExecutionContext struct {
	input       *flows.InputBlock
	inputVals   map[string]any
	contextVals map[string]any
	outputVals  map[string]any
}

// NewExecutionContext creates a new execution context
func NewExecutionContext(input *flows.InputBlock) *ExecutionContext {
	return &ExecutionContext{
		input:       input,
		inputVals:   make(map[string]any),
		contextVals: make(map[string]any),
		outputVals:  make(map[string]any),
	}
}

// SetInput sets input field values and applies defaults
func (c *ExecutionContext) SetInput(vals map[string]string) {
	// Apply input values or defaults
	if c.input != nil {
		for _, field := range c.input.GetAllFields() {
			if val, ok := vals[field.Name]; ok {
				c.inputVals[field.Name] = val
			} else if field.Default != "" {
				c.inputVals[field.Name] = coerceType(field.Type, field.Default)
			}
		}
	}
}

// GetInputField retrieves an input field value
func (c *ExecutionContext) GetInputField(name string) (any, error) {
	if val, ok := c.inputVals[name]; ok {
		return val, nil
	}
	return nil, errors.Errorf("input variable %s undefined", name)
}

// SetContextField sets a context field value
func (c *ExecutionContext) SetContextField(name string, value any) error {
	c.contextVals[name] = value
	return nil
}

// GetContextField retrieves a context field value
func (c *ExecutionContext) GetContextField(name string) (any, error) {
	if val, ok := c.contextVals[name]; ok {
		return val, nil
	}
	return nil, errors.Errorf("context variable %s undefined", name)
}

// SetOutputField sets an output field value
func (c *ExecutionContext) SetOutputField(name string, value any) error {
	c.outputVals[name] = value
	return nil
}

// GetOutputField retrieves an output field value
func (c *ExecutionContext) GetOutputField(name string) (any, error) {
	if val, ok := c.outputVals[name]; ok {
		return val, nil
	}
	return nil, errors.Errorf("output variable %s undefined", name)
}

// BuildInputScope builds the input scope for template substitution
func (c *ExecutionContext) BuildInputScope() map[string]any {
	scope := make(map[string]any)
	for k, v := range c.inputVals {
		scope[k] = v
	}
	return scope
}

// BuildContextScope builds the context scope for template substitution
func (c *ExecutionContext) BuildContextScope() map[string]any {
	scope := make(map[string]any)
	for k, v := range c.contextVals {
		scope[k] = v
	}
	return scope
}

// BuildOutputScope builds the output scope for template substitution
func (c *ExecutionContext) BuildOutputScope() map[string]any {
	scope := make(map[string]any)
	for k, v := range c.outputVals {
		scope[k] = v
	}
	return scope
}

// BuildFullScope builds the full evaluation scope with nested structures
func (c *ExecutionContext) BuildFullScope() map[string]any {
	scope := make(map[string]any)

	inputScope := c.BuildInputScope()
	if len(inputScope) > 0 {
		scope["input"] = inputScope
	}

	contextScope := c.BuildContextScope()
	if len(contextScope) > 0 {
		scope["context"] = contextScope
	}

	outputScope := c.BuildOutputScope()
	if len(outputScope) > 0 {
		scope["output"] = outputScope
	}

	return scope
}


