package executor

import (
	"fmt"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/errors"
	"github.com/denkhaus/gollum/pkg/flows/variables"
)

// OutputBinder handles declarative output field binding initialization
type OutputBinder struct{}

// NewOutputBinder creates a new output binder
func NewOutputBinder() *OutputBinder {
	return &OutputBinder{}
}

// InitializeBindings populates declarative output fields from their sources
// This is called after computed fields are evaluated
func (b *OutputBinder) InitializeBindings(ctx ExecutionContext, outputBlock *flows.OutputBlock) error {
	if outputBlock == nil {
		return nil
	}

	// Get declarative output fields (those with 'from' attribute)
	declarative := outputBlock.GetDeclarative()

	for _, field := range declarative {
		if err := b.initializeBinding(ctx, &field); err != nil {
			return err
		}
	}

	return nil
}

// initializeBinding populates a single declarative output field
func (b *OutputBinder) initializeBinding(ctx ExecutionContext, field *flows.FieldDef) error {
	// Parse the 'from' reference
	sourceScope, sourceName, err := variables.ParseFieldReference(field.From)
	if err != nil {
		return errors.NewOutputBindingError(field.Name, field.From, err)
	}

	// Validate scope
	if !variables.IsValidSourceScope(sourceScope) {
		return errors.NewOutputBindingError(field.Name, field.From,
			fmt.Errorf("invalid scope '%s' (allowed: input, context, computed, output)", sourceScope))
	}

	// Get the value from the source
	var value any
	switch sourceScope {
	case "computed":
		value, err = ctx.GetComputedField(sourceName)
	case "input":
		value = ctx.GetInput(sourceName)
		if value == nil {
			return errors.NewOutputBindingError(field.Name, field.From,
				fmt.Errorf("input field not set: %s", sourceName))
		}
	case "context":
		value, err = ctx.GetContextField(sourceName)
	case "output":
		value, err = ctx.GetOutputField(sourceName)
	default:
		return errors.NewOutputBindingError(field.Name, field.From,
			fmt.Errorf("unsupported scope: %s", sourceScope))
	}

	if err != nil {
		return errors.NewOutputBindingError(field.Name, field.From, err)
	}

	// Set the output value using direct assignment to bypass readonly check
	// during initialization
	return b.setOutputValueDirectly(ctx, field.Name, value)
}

// setOutputValueDirectly sets an output field value bypassing readonly checks
// This is used during initialization of declarative output bindings
func (b *OutputBinder) setOutputValueDirectly(ctx ExecutionContext, name string, value any) error {
	// Cast to contextImpl to access internal setter
	// This bypasses the readonly check that will be added in Task 7
	if c, ok := ctx.(*contextImpl); ok {
		return b.setInternalOutputField(c, name, value)
	}
	// Fallback to regular SetOutputField for other implementations
	return ctx.SetOutputField(name, value)
}

// setInternalOutputField sets output directly on contextImpl, bypassing readonly checks
func (b *OutputBinder) setInternalOutputField(c *contextImpl, name string, value any) error {
	if c.outputValues == nil {
		return &errors.NoSchemaError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "output schema not defined",
			},
			Scope: "output",
		}
	}

	// Check if field exists
	if !c.outputValues.Has(name) {
		return &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "field not defined",
				Field:   name,
			},
			Scope: "output",
		}
	}

	// Convert value based on its actual type
	switch v := value.(type) {
	case string:
		return c.outputValues.SetFromString(name, v)
	case int:
		return c.outputValues.SetInt(name, v)
	case bool:
		return c.outputValues.SetBool(name, v)
	case float64:
		return c.outputValues.SetFloat(name, v)
	default:
		return c.outputValues.SetFromString(name, fmt.Sprintf("%v", v))
	}
}
