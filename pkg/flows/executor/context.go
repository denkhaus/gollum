package executor

import (
	"strconv"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/ast"
	"github.com/denkhaus/gollum/pkg/flows/variables"
	"github.com/go-ap/errors"
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
	input        *flows.InputBlock
	output       *flows.OutputBlock
	contextBlock *flows.ContextBlock
	computedBlock *flows.ComputedBlock
	inputVals     map[string]any
	contextVals   map[string]any
	outputVals    map[string]any
	computedVals  *variables.ComputedValues
	eval          *Evaluator
}

// NewContext creates a new execution context with optional schema definitions
func NewContext(input *flows.InputBlock, output *flows.OutputBlock, contextBlock *flows.ContextBlock, inputVals map[string]string) *Context {
	ctx := &Context{
		input:         input,
		output:        output,
		contextBlock:  contextBlock,
		computedBlock: nil, // Will be set from flow.Computed if available
		inputVals:     make(map[string]any),
		contextVals:   make(map[string]any),
		outputVals:    make(map[string]any),
		computedVals:  variables.NewComputedValues(nil), // Empty initially, will be populated from flow
		eval:          NewEvaluator(),
	}

	// Initialize input values or defaults
	if input != nil {
		for _, field := range input.GetAllFields() {
			if val, ok := inputVals[field.Name]; ok {
				ctx.inputVals[field.Name] = val
			} else if field.Default != "" {
				ctx.inputVals[field.Name] = coerceType(field.Type, field.Default)
			}
		}
	}

	// Pre-initialize output fields with nil (so we can validate against schema)
	if output != nil {
		for _, field := range output.GetAllFields() {
			ctx.outputVals[field.Name] = nil
		}
	}

	// Pre-initialize context fields with nil (so we can validate against schema)
	// NOTE: This is ONLY for regular context fields, NOT computed fields
	if contextBlock != nil {
		for _, field := range contextBlock.GetAllFields() {
			ctx.contextVals[field.Name] = nil
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
		computedFields = append(computedFields, flows.ComputedField{
			Name: cf.Name,
			Type: cf.Type,
			Eval: cf.Eval,
		})
	}

	// Initialize ComputedValues with the fields
	c.computedVals = variables.NewComputedValues(computedFields)
}

// GetInput retrieves an input field value
func (c *Context) GetInput(name string) any {
	return c.inputVals[name]
}

// GetContextField retrieves a context field value
func (c *Context) GetContextField(name string) (any, error) {
	if val, ok := c.contextVals[name]; ok {
		return val, nil
	}

	return nil, errors.Errorf("context variable %s undefined", name)
}

// SetOutputField sets the output variable by name
// Returns error if the field is not defined in the output schema
func (c *Context) SetOutputField(name string, value any) error {
	if c.output != nil {
		// Check if field is defined in schema
		if _, exists := c.outputVals[name]; !exists {
			return errors.Errorf("output field '%s' not defined in flow schema", name)
		}
	}
	c.outputVals[name] = value
	return nil
}

// GetOutputField retrieves an output field value
func (c *Context) GetOutputField(name string) (any, error) {
	if val, ok := c.outputVals[name]; ok {
		return val, nil
	}

	return nil, errors.Errorf("output variable %s undefined", name)
}

// EvaluateComputed evaluates all computed fields using the ComputedEvaluator
func (c *Context) EvaluateComputed() error {
	if c.computedVals == nil {
		return nil
	}

	// Create input/context/output wrappers for evaluation
	inputVals := variables.NewInputValues(c.input)
	for k, v := range c.inputVals {
		inputVals.SetRaw(k, v)
	}

	contextVals := variables.NewContextValues(c.contextBlock)
	for k, v := range c.contextVals {
		contextVals.SetRaw(k, v)
	}

	outputVals := variables.NewOutputValues(c.output)
	for k, v := range c.outputVals {
		outputVals.SetRaw(k, v)
	}

	// Create evaluator
	eval := variables.NewComputedEvaluator(c.computedVals, inputVals, contextVals, outputVals)

	// Evaluate all computed fields
	return eval.ComputeDirty()
}

// GetComputedEvaluator returns a configured ComputedEvaluator for this context
func (c *Context) GetComputedEvaluator() *variables.ComputedEvaluator {
	// Create input/context/output wrappers for evaluation
	inputVals := variables.NewInputValues(c.input)
	for k, v := range c.inputVals {
		inputVals.SetRaw(k, v)
	}

	contextVals := variables.NewContextValues(c.contextBlock)
	for k, v := range c.contextVals {
		contextVals.SetRaw(k, v)
	}

	outputVals := variables.NewOutputValues(c.output)
	for k, v := range c.outputVals {
		outputVals.SetRaw(k, v)
	}

	return variables.NewComputedEvaluator(c.computedVals, inputVals, contextVals, outputVals)
}

// buildScope builds the evaluation scope for backward compatibility
// TODO: Remove this once all code uses GetComputedEvaluator
func (c *Context) buildScope() map[string]any {
	scope := make(map[string]any)

	// Build input scope
	inputScope := make(map[string]any)
	for k, v := range c.inputVals {
		inputScope[k] = v
	}
	if len(inputScope) > 0 {
		scope["input"] = inputScope
	}

	outputScope := make(map[string]any)
	for k, v := range c.outputVals {
		outputScope[k] = v
	}
	if len(outputScope) > 0 {
		scope["output"] = outputScope
	}

	contextScope := make(map[string]any)
	for k, v := range c.contextVals {
		contextScope[k] = v
	}
	if len(contextScope) > 0 {
		scope["context"] = contextScope
	}

	return scope
}

// SetContextField sets the context variable by name
// Returns error if the field is not defined in the schema
func (c *Context) SetContextField(name string, value any) error {
	// Check if field is defined in schema
	if c.contextBlock != nil {
		if _, exists := c.contextVals[name]; !exists {
			return errors.Errorf("context field '%s' not defined in flow schema", name)
		}
	}
	c.contextVals[name] = value
	return nil
}

// GetComputedField retrieves a computed field value
func (c *Context) GetComputedField(name string) (any, error) {
	if !c.computedVals.Has(name) {
		return nil, errors.Errorf("computed field '%s' not defined", name)
	}
	return c.computedVals.GetValue(name)
}

// coerceType converts string to appropriate type
func coerceType(typ, val string) any {
	switch typ {
	case "int":
		i, _ := strconv.Atoi(val)
		return i
	case "bool":
		b, _ := strconv.ParseBool(val)
		return b
	case "float":
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return val
	}
}
