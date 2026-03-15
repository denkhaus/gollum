package executor

import (
	"strconv"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/ast"
	"github.com/denkhaus/gollum/pkg/flows/errors"
	"github.com/denkhaus/gollum/pkg/flows/variables"
)

// Evaluator wraps the ast package for expression evaluation
type Evaluator struct {
	// Placeholder for future state/caching if needed
}

// NewEvaluator creates a new expression evaluator
func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

// EvaluateExpr parses and evaluates an expression string against a scope
func (e *Evaluator) EvaluateExpr(expr string, scope map[string]any) (any, error) {
	parsed, err := ast.ParseExpression(expr)
	if err != nil {
		return nil, err
	}
	return ast.Evaluate(parsed, scope)
}

// ExtractDependencies extracts field references from an expression
func (e *Evaluator) ExtractDependencies(expr string) []string {
	parsed, err := ast.ParseExpression(expr)
	if err != nil {
		return nil
	}

	var deps []string
	extractDeps(parsed, &deps)
	return deps
}

// extractDeps recursively extracts field references from AST
func extractDeps(expr ast.Expr, deps *[]string) {
	switch e := expr.(type) {
	case *ast.CallExpr:
		for _, arg := range e.Args {
			extractDeps(arg, deps)
		}
	case *ast.FieldRef:
		// For field references, we only care about the first path component
		// since that's the top-level context field being accessed
		if len(e.Path) > 0 {
			// Add the first path element as a dependency (e.g., "status" from "context.status")
			*deps = append(*deps, e.Path[0])
		}
	}
}

// Context manages execution context with input, output, and computed fields
type Context struct {
	inputBlock    *flows.InputBlock
	inputVals     *variables.InputValues
	outputBlock   *flows.OutputBlock
	outputValues  *variables.OutputValues
	contextBlock  *flows.ContextBlock
	contextValues *variables.ContextValues
	computedBlock *flows.ComputedBlock
	computedVals  *variables.ComputedValues
	eval          *Evaluator
	lastError     *ErrorContext
}

// NewContext creates a new execution context with optional schema definitions
func NewContext(
	inputBlock *flows.InputBlock,
	outputBlock *flows.OutputBlock,
	contextBlock *flows.ContextBlock,
	inputVals map[string]string,
) *Context {
	ctx := &Context{
		inputBlock:    inputBlock,
		inputVals:     variables.NewInputValues(inputBlock), // Use typed wrapper
		outputBlock:   outputBlock,
		outputValues:  variables.NewOutputValues(outputBlock),
		contextBlock:  contextBlock,
		contextValues: variables.NewContextValues(contextBlock),
		computedBlock: nil,                              // Will be set from flow.Computed if available
		computedVals:  variables.NewComputedValues(nil), // Empty initially, will be populated from flow
		eval:          NewEvaluator(),
	}

	// Initialize input values or defaults
	if inputBlock != nil {
		for _, field := range inputBlock.GetAllFields() {
			if val, ok := inputVals[field.Name]; ok {
				// Use typed setters based on field type
				switch variables.ValueType(field.Type) {
				case variables.TypeString:
					_ = ctx.inputVals.SetString(field.Name, val)
				case variables.TypeInt:
					if i, err := strconv.Atoi(val); err == nil {
						_ = ctx.inputVals.SetInt(field.Name, i)
					}
				case variables.TypeBool:
					if b, err := strconv.ParseBool(val); err == nil {
						_ = ctx.inputVals.SetBool(field.Name, b)
					}
				case variables.TypeFloat:
					if f, err := strconv.ParseFloat(val, 64); err == nil {
						_ = ctx.inputVals.SetFloat(field.Name, f)
					}
				}
			}
			// Defaults are already handled by NewInputValues
		}
	}

	return ctx
}

// SetComputedBlock sets the computed field definitions and initializes the evaluator
func (c *Context) SetComputedBlock(block *flows.ComputedBlock) {
	c.computedBlock = block

	if block == nil {
		return
	}

	// Convert ComputedFieldDef to ComputedField for the variables package
	computedFields := make([]flows.ComputedField, 0, len(block.GetAllFields()))
	for _, cf := range block.GetAllFields() {
		// Use field directly - ComputedFieldDef has the same structure
		computedFields = append(computedFields, flows.ComputedField(cf))
	}

	// Initialize ComputedValues with the fields
	c.computedVals = variables.NewComputedValues(computedFields)
}

// GetInput retrieves an input field value
func (c *Context) GetInput(name string) any {
	if c.inputBlock == nil {
		return nil
	}

	// Try to get based on type from schema
	for _, field := range c.inputBlock.GetAllFields() {
		if field.Name == name {
			switch variables.ValueType(field.Type) {
			case variables.TypeString:
				val, _ := c.inputVals.GetString(name)
				return val
			case variables.TypeInt:
				val, _ := c.inputVals.GetInt(name)
				return val
			case variables.TypeBool:
				val, _ := c.inputVals.GetBool(name)
				return val
			case variables.TypeFloat:
				val, _ := c.inputVals.GetFloat(name)
				return val
			}
		}
	}
	return nil
}

// GetContextField retrieves a context field value
// Returns error if field is not defined or not set
func (c *Context) GetContextField(name string) (any, error) {
	if c.contextValues == nil {
		return nil, &errors.NoSchemaError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "context schema not defined",
			},
			Scope: "context",
		}
	}

	// Try to get based on type - we need to check the schema to know which type to use
	if c.contextBlock != nil {
		// Check string fields
		for _, field := range c.contextBlock.Strings {
			if field.Name == name {
				val, err := c.contextValues.GetString(name)
				if err != nil {
					return nil, err
				}
				return val, nil
			}
		}
		// Check int fields
		for _, field := range c.contextBlock.Ints {
			if field.Name == name {
				val, err := c.contextValues.GetInt(name)
				if err != nil {
					return nil, err
				}
				return val, nil
			}
		}
		// Check bool fields
		for _, field := range c.contextBlock.Bools {
			if field.Name == name {
				val, err := c.contextValues.GetBool(name)
				if err != nil {
					return nil, err
				}
				return val, nil
			}
		}
		// Check float fields
		for _, field := range c.contextBlock.Floats {
			if field.Name == name {
				val, err := c.contextValues.GetFloat(name)
				if err != nil {
					return nil, err
				}
				return val, nil
			}
		}
	}

	return nil, &errors.UnknownFieldError{
		FlowError: errors.FlowError{
			Code:    errors.ErrCodeUnknownField,
			Message: "context field not defined",
			Field:   name,
		},
		Scope: "context",
	}
}

// SetContextField sets a context field value
// Returns error if the field is not defined in the schema or if type conversion fails
func (c *Context) SetContextField(name string, value any) error {
	if c.contextValues == nil {
		return &errors.NoSchemaError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "context schema not defined",
			},
			Scope: "context",
		}
	}

	// Convert value to string for type conversion
	valueStr, ok := value.(string)
	if !ok {
		// Fallback to raw set for non-string values (should not happen from LLM)
		c.contextValues.SetRaw(name, value)
		return nil
	}

	// Use SetFromString for automatic type conversion
	return c.contextValues.SetFromString(name, valueStr)
}

// SetOutputField sets the output variable by name
// Returns error if the field is not defined in the output schema
func (c *Context) SetOutputField(name string, value any) error {
	if c.outputValues == nil {
		return &errors.NoSchemaError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "output schema not defined",
			},
			Scope: "output",
		}
	}

	// Use SetRaw to set the value (type coercion handled by caller)
	c.outputValues.SetRaw(name, value)
	return nil
}

// GetOutputField retrieves an output field value
// Returns error if field is not defined or not set
func (c *Context) GetOutputField(name string) (any, error) {
	if c.outputValues == nil {
		return nil, &errors.NoSchemaError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "output schema not defined",
			},
			Scope: "output",
		}
	}

	if c.outputBlock != nil {
		// Try to get based on type from schema
		for _, field := range c.outputBlock.GetAllFields() {
			if field.Name == name {
				switch variables.ValueType(field.Type) {
				case variables.TypeString:
					val, err := c.outputValues.GetString(name)
					if err != nil {
						return nil, err
					}
					return val, nil
				case variables.TypeInt:
					val, err := c.outputValues.GetInt(name)
					if err != nil {
						return nil, err
					}
					return val, nil
				case variables.TypeBool:
					val, err := c.outputValues.GetBool(name)
					if err != nil {
						return nil, err
					}
					return val, nil
				case variables.TypeFloat:
					val, err := c.outputValues.GetFloat(name)
					if err != nil {
						return nil, err
					}
					return val, nil
				}
			}
		}
	}

	return nil, &errors.UnknownFieldError{
		FlowError: errors.FlowError{
			Code:    errors.ErrCodeUnknownField,
			Message: "output field not defined",
			Field:   name,
		},
		Scope: "output",
	}
}


// SetError sets the last error context
func (c *Context) SetError(err *ErrorContext) {
	c.lastError = err
}

// GetError returns the last error context (may be nil)
func (c *Context) GetError() *ErrorContext {
	return c.lastError
}

// EvaluateComputed evaluates all computed fields using the ComputedEvaluator
func (c *Context) EvaluateComputed() error {
	if c.computedVals == nil {
		return nil
	}

	// Use existing input, context and output wrappers (source of truth)
	eval := variables.NewComputedEvaluator(c.computedVals, c.inputVals, c.contextValues, c.outputValues)

	// Evaluate all computed fields
	return eval.ComputeDirty()
}

// GetComputedEvaluator returns a configured ComputedEvaluator for this context
func (c *Context) GetComputedEvaluator() *variables.ComputedEvaluator {
	// Use existing input, context and output wrappers (source of truth)
	return variables.NewComputedEvaluator(c.computedVals, c.inputVals, c.contextValues, c.outputValues)
}

// buildScope builds the evaluation scope for backward compatibility
// TODO: Remove this once all code uses GetComputedEvaluator
func (c *Context) buildScope() map[string]any {
	scope := make(map[string]any)

	// Build input scope from typed wrapper
	if c.inputBlock != nil && c.inputVals != nil {
		inputScope := make(map[string]any)
		for _, field := range c.inputBlock.GetAllFields() {
			if val, ok := c.inputVals.GetRaw(field.Name); ok {
				inputScope[field.Name] = val
			}
			// Unset fields are not added to scope
		}
		if len(inputScope) > 0 {
			scope["input"] = inputScope
		}
	}

	// Build output scope from typed wrapper
	if c.outputValues != nil && c.outputBlock != nil {
		outputScope := make(map[string]any)
		for _, field := range c.outputBlock.GetAllFields() {
			if val, ok := c.outputValues.GetRaw(field.Name); ok {
				outputScope[field.Name] = val
			}
			// Unset fields are not added to scope
		}
		if len(outputScope) > 0 {
			scope["output"] = outputScope
		}
	}

	// Build context scope from typed wrapper
	if c.contextValues != nil && c.contextBlock != nil {
		contextScope := make(map[string]any)
		for _, field := range c.contextBlock.GetAllFields() {
			if val, ok := c.contextValues.GetRaw(field.Name); ok {
				contextScope[field.Name] = val
			}
			// Unset fields are not added to scope
		}
		if len(contextScope) > 0 {
			scope["context"] = contextScope
		}
	}

	// Add system scope with error context if available
	if c.lastError != nil {
		scope["sys"] = map[string]any{
			"error": c.lastError,
		}
	}

	return scope
}

// GetComputedField retrieves a computed field value
func (c *Context) GetComputedField(name string) (any, error) {
	if !c.computedVals.Has(name) {
		return nil, &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "computed field not defined",
				Field:   name,
			},
			Scope: "computed",
		}
	}
	return c.computedVals.GetValue(name)
}
