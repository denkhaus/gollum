package executor

import (
	"fmt"
	"strconv"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/ast"
	"github.com/denkhaus/gollum/pkg/flows/errors"
	"github.com/denkhaus/gollum/pkg/flows/variables"
)

// EvaluationScope is a typed scope map for expression evaluation
type EvaluationScope map[flows.FlowVariableScope]map[string]any

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

// EvaluateExprTyped evaluates an expression against a typed EvaluationScope
func (e *Evaluator) EvaluateExprTyped(expr string, scope EvaluationScope) (any, error) {
	// Convert EvaluationScope to map[string]any for ast.Evaluate
	// This is needed because the ast package doesn't know about our typed scopes
	converted := make(map[string]any)
	for key, value := range scope {
		converted[string(key)] = value
	}
	return e.EvaluateExpr(expr, converted)
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

type ExecutionContext interface {
	SetContextField(name string, value any) error
	GetContextField(name string) (any, error)
	GetOutputField(name string) (any, error)
	SetOutputField(name string, value any) error
	GetComputedField(name string) (any, error)
	GetInput(name string) any
	EvaluateComputed() error
	SetError(ctx *ErrorContext)
	GetError() *ErrorContext
	SubstituteTemplate(tmpl string) string
}

// Context manages execution contextImpl with input, output, and computed fields
type contextImpl struct {
	inputBlock    *flows.InputBlock
	inputVals     *variables.FieldValues[flows.FieldDef]
	outputBlock   *flows.OutputBlock
	outputValues  *variables.FieldValues[flows.FieldDef]
	contextBlock  *flows.ContextBlock
	contextValues *variables.FieldValues[flows.ContextField]
	computedBlock *flows.ComputedBlock
	computedVals  *variables.ComputedValues
	eval          *Evaluator
	lastError     *ErrorContext
}

func NewContext(
	inputBlock *flows.InputBlock,
	outputBlock *flows.OutputBlock,
	contextBlock *flows.ContextBlock,
	inputVals map[string]string) ExecutionContext {
	return newContext(inputBlock, outputBlock, contextBlock, inputVals)
}

// NewContext creates a new execution context with optional schema definitions
func newContext(
	inputBlock *flows.InputBlock,
	outputBlock *flows.OutputBlock,
	contextBlock *flows.ContextBlock,
	inputVals map[string]string,
) *contextImpl {
	ctx := &contextImpl{
		inputBlock:    inputBlock,
		inputVals:     variables.NewFieldValues(getAllFieldsSafe(inputBlock)),
		outputBlock:   outputBlock,
		outputValues:  variables.NewFieldValues(getAllFieldsSafe(outputBlock)),
		contextBlock:  contextBlock,
		contextValues: variables.NewFieldValues(getAllContextFieldsSafe(contextBlock)),
		computedBlock: nil,                              // Will be set from flow.Computed if available
		computedVals:  variables.NewComputedValues(nil), // Empty initially, will be populated from flow
		eval:          NewEvaluator(),
	}

	// Initialize input values or defaults
	if inputBlock != nil {
		for _, field := range inputBlock.GetAllFields() {
			if val, ok := inputVals[field.Name]; ok {
				// Use typed setters based on field type
				switch flows.ValueType(field.Type) {
				case flows.TypeString:
					_ = ctx.inputVals.SetString(field.Name, val)
				case flows.TypeInt:
					if i, err := strconv.Atoi(val); err == nil {
						_ = ctx.inputVals.SetInt(field.Name, i)
					}
				case flows.TypeBool:
					if b, err := strconv.ParseBool(val); err == nil {
						_ = ctx.inputVals.SetBool(field.Name, b)
					}
				case flows.TypeFloat:
					if f, err := strconv.ParseFloat(val, 64); err == nil {
						_ = ctx.inputVals.SetFloat(field.Name, f)
					}
				}
			}
			// Defaults are already handled by NewFieldValues
		}
	}

	return ctx
}

// getAllFieldsSafe safely gets all fields from an InputBlock or OutputBlock
func getAllFieldsSafe(block interface{}) []flows.FieldDef {
	if b, ok := block.(*flows.InputBlock); ok && b != nil {
		return b.GetAllFields()
	}
	if b, ok := block.(*flows.OutputBlock); ok && b != nil {
		return b.GetAllFields()
	}
	return []flows.FieldDef{}
}

// getAllContextFieldsSafe safely gets all fields from a ContextBlock
func getAllContextFieldsSafe(block *flows.ContextBlock) []flows.ContextField {
	if block != nil {
		return block.GetAllFields()
	}
	return []flows.ContextField{}
}

// SetComputedBlock sets the computed field definitions and initializes the evaluator
func (c *contextImpl) SetComputedBlock(block *flows.ComputedBlock) {
	c.computedBlock = block

	if block == nil {
		return
	}

	// Convert ComputedFieldDef to ComputedField for the variables package
	computedFields := make([]flows.ComputedField, 0, len(block.GetAllFields()))
	for _, cf := range block.GetAllFields() {
		// Manually convert since ComputedFieldDef.Type is now ValueType
		computedFields = append(computedFields, flows.ComputedField{
			Name: cf.Name,
			Type: string(cf.Type),
			Eval: cf.Eval,
		})
	}

	// Initialize ComputedValues with the fields
	c.computedVals = variables.NewComputedValues(computedFields)
}

// GetInput retrieves an input field value
func (c *contextImpl) GetInput(name string) any {
	if c.inputBlock == nil {
		return nil
	}

	// Try to get based on type from schema
	for _, field := range c.inputBlock.GetAllFields() {
		if field.Name == name {
			switch flows.ValueType(field.Type) {
			case flows.TypeString:
				val, _ := c.inputVals.GetString(name)
				return val
			case flows.TypeInt:
				val, _ := c.inputVals.GetInt(name)
				return val
			case flows.TypeBool:
				val, _ := c.inputVals.GetBool(name)
				return val
			case flows.TypeFloat:
				val, _ := c.inputVals.GetFloat(name)
				return val
			}
		}
	}
	return nil
}

// GetContextField retrieves a context field value
// Returns error if field is not defined or not set
func (c *contextImpl) GetContextField(name string) (any, error) {
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
func (c *contextImpl) SetContextField(name string, value any) error {
	if c.contextValues == nil {
		return &errors.NoSchemaError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "context schema not defined",
			},
			Scope: "context",
		}
	}

	// Check if field exists
	if !c.contextValues.Has(name) {
		return &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "field not defined",
				Field:   name,
			},
			Scope: "context",
		}
	}

	// Convert value based on its actual type
	switch v := value.(type) {
	case string:
		return c.contextValues.SetFromString(name, v)
	case int:
		return c.contextValues.SetInt(name, v)
	case bool:
		return c.contextValues.SetBool(name, v)
	case float64:
		return c.contextValues.SetFloat(name, v)
	default:
		// Try to convert to string as fallback
		return c.contextValues.SetFromString(name, fmt.Sprintf("%v", v))
	}
}

// SetOutputField sets the output variable by name
// Returns error if the field is not defined or is a declarative (readonly) field
func (c *contextImpl) SetOutputField(name string, value any) error {
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

	// Check if field is declarative (readonly)
	if c.outputBlock != nil {
		for _, field := range c.outputBlock.GetAllFields() {
			if field.Name == name && field.From != "" {
				return errors.NewOutputFieldReadOnlyError(name, field.From)
			}
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
		// Try to convert to string as fallback
		return c.outputValues.SetFromString(name, fmt.Sprintf("%v", v))
	}
}

// GetOutputField retrieves an output field value
// Returns error if field is not defined or not set
func (c *contextImpl) GetOutputField(name string) (any, error) {
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
				switch flows.ValueType(field.Type) {
				case flows.TypeString:
					val, err := c.outputValues.GetString(name)
					if err != nil {
						return nil, err
					}
					return val, nil
				case flows.TypeInt:
					val, err := c.outputValues.GetInt(name)
					if err != nil {
						return nil, err
					}
					return val, nil
				case flows.TypeBool:
					val, err := c.outputValues.GetBool(name)
					if err != nil {
						return nil, err
					}
					return val, nil
				case flows.TypeFloat:
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
func (c *contextImpl) SetError(err *ErrorContext) {
	c.lastError = err
}

// GetError returns the last error context (may be nil)
func (c *contextImpl) GetError() *ErrorContext {
	return c.lastError
}

// EvaluateComputed evaluates all computed fields using the ComputedEvaluator
func (c *contextImpl) EvaluateComputed() error {
	if c.computedVals == nil {
		return nil
	}

	// Use existing input, context and output wrappers (source of truth)
	eval := variables.NewComputedEvaluator(c.computedVals, c.inputVals, c.contextValues, c.outputValues)

	// Evaluate all computed fields
	return eval.ComputeDirty()
}

// GetComputedEvaluator returns a configured ComputedEvaluator for this context
func (c *contextImpl) GetComputedEvaluator() *variables.ComputedEvaluator {
	// Use existing input, context and output wrappers (source of truth)
	return variables.NewComputedEvaluator(c.computedVals, c.inputVals, c.contextValues, c.outputValues)
}

// buildScope builds the evaluation scope for expression evaluation
// Used by executeTransition, executeCall, and substituteTemplate
func (c *contextImpl) buildScope() EvaluationScope {
	scope := make(EvaluationScope)

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
			scope[flows.FlowVariableScopeInput] = inputScope
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
			scope[flows.FlowVariableScopeOutput] = outputScope
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
			scope[flows.FlowVariableScopeContext] = contextScope
		}
	}

	// Build computed scope for expression evaluation
	if c.computedVals != nil && c.computedBlock != nil {
		computedScope := make(map[string]any)
		for _, field := range c.computedBlock.GetAllFields() {
			if val, err := c.computedVals.GetValue(field.Name); err == nil {
				computedScope[field.Name] = val
			}
		}
		if len(computedScope) > 0 {
			scope[flows.FlowVariableScopeComputed] = computedScope
		}
	}

	// Add system scope with error context if available
	if c.lastError != nil {
		scope[flows.FlowVariableScopeSys] = map[string]any{
			"error": c.lastError,
		}
	}

	return scope
}

// GetComputedField retrieves a computed field value
func (c *contextImpl) GetComputedField(name string) (any, error) {
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

// SubstituteTemplate replaces variables in template strings using context values
func (c *contextImpl) SubstituteTemplate(tmpl string) string {
	return substituteTemplate(c, tmpl)
}
