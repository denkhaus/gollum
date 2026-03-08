package executor

import (
	"strconv"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/ast"
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
		// Add the full reference as a dependency
		if e.Prefix != "" {
			*deps = append(*deps, e.Prefix)
		}
		for _, part := range e.Path {
			*deps = append(*deps, part)
		}
	}
}

// Context manages execution context with input, output, and computed fields
type Context struct {
	input    *flows.InputBlock
	inputVals map[string]any
	values    map[string]any  // context and output fields
	computed map[string]string // computed field expressions
	eval     *Evaluator
}

// NewContext creates a new execution context
func NewContext(input *flows.InputBlock, inputVals map[string]any) *Context {
	ctx := &Context{
		input:    input,
		inputVals: make(map[string]any),
		values:   make(map[string]any),
		computed: make(map[string]string),
		eval:     NewEvaluator(),
	}

	// Apply input values or defaults
	if input != nil {
		for _, field := range input.GetAllFields() {
			if val, ok := inputVals[field.Name]; ok {
				ctx.inputVals[field.Name] = val
			} else if field.Default != "" {
				ctx.inputVals[field.Name] = coerceType(field.Type, field.Default)
			}
		}
	}

	return ctx
}

// GetInput retrieves an input field value
func (c *Context) GetInput(name string) any {
	return c.inputVals[name]
}

// SetContextField sets a context field value
func (c *Context) SetContextField(name string, value any) {
	c.values[name] = value
}

// GetContextField retrieves a context field value
func (c *Context) GetContextField(name string) (any, bool) {
	val, ok := c.values[name]
	return val, ok
}

// SetOutputField sets an output field value
func (c *Context) SetOutputField(name string, value any) {
	c.values["output."+name] = value
}

// GetOutputField retrieves an output field value
func (c *Context) GetOutputField(name string) (any, bool) {
	val, ok := c.values["output."+name]
	return val, ok
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
