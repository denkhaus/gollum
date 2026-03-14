package executor

import (
	"strconv"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/ast"
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
			// If there's a prefix (like "context.status"), add the first path element
			// If there's no prefix (bare identifier like "status"), add it directly
			if e.Prefix != "" {
				// For "context.status", we depend on "status" being in the context scope
				*deps = append(*deps, e.Path[0])
			} else {
				// For bare identifiers, use the first path element
				*deps = append(*deps, e.Path[0])
			}
		}
	}
}

// Context manages execution context with input, output, and computed fields
type Context struct {
	input       *flows.InputBlock
	inputVals   map[string]any
	contextVals map[string]any
	outputVals  map[string]any
	computed    map[string]string
	eval        *Evaluator
}

// NewContext creates a new execution context
func NewContext(input *flows.InputBlock, inputVals map[string]string) *Context {
	ctx := &Context{
		input:       input,
		inputVals:   make(map[string]any),
		contextVals: make(map[string]any),
		outputVals:  make(map[string]any),
		computed:    make(map[string]string),
		eval:        NewEvaluator(),
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

// GetContextField retrieves a context field value
func (c *Context) GetContextField(name string) (any, error) {
	if val, ok := c.contextVals[name]; ok {
		return val, nil
	}

	return nil, errors.Errorf("context variable %s undefined", name)
}

// SetOutputField sets the output variable by name
// If the variable doesn't exist an error is thrown
func (c *Context) SetOutputField(name string, value string) error {
	if _, ok := c.outputVals[name]; ok {
		c.outputVals[name] = value
	}

	return errors.Errorf("output variable %s undefined", name)
}

// GetOutputField retrieves an output field value
func (c *Context) GetOutputField(name string) (any, error) {
	if val, ok := c.outputVals[name]; ok {
		return val, nil
	}

	return nil, errors.Errorf("output variable %s undefined", name)
}

// EvaluateComputedFields evaluates all computed fields in dependency order
func (c *Context) EvaluateComputedFields(ctxBlock *flows.ContextBlock) error {
	if ctxBlock == nil {
		return nil
	}

	// Store computed expressions for immutability check
	for _, cf := range ctxBlock.Computeds {
		c.computed[cf.Name] = cf.When
	}

	// Build dependency graph and evaluate in topological order
	evaluated := make(map[string]bool)
	for len(evaluated) < len(ctxBlock.Computeds) {
		progress := false
		for _, cf := range ctxBlock.Computeds {
			if evaluated[cf.Name] {
				continue
			}

			// Check if all dependencies are evaluated
			deps := c.eval.ExtractDependencies(cf.When)
			ready := true
			for _, dep := range deps {
				if !c.isDependencyResolved(dep, evaluated) {
					ready = false
					break
				}
			}

			if ready {
				val, err := c.eval.EvaluateExpr(cf.When, c.buildScope())
				if err != nil {
					return err
				}
				// Set directly in values map, bypassing SetContextField
				// to avoid the immutability check during initial evaluation
				c.values[cf.Name] = val
				evaluated[cf.Name] = true
				progress = true
			}
		}

		if !progress {
			// Circular dependency detected
			return nil
		}
	}

	return nil
}

// isDependencyResolved checks if a dependency is available
func (c *Context) isDependencyResolved(dep string, evaluated map[string]bool) bool {
	// Check if it's an input field
	if _, ok := c.inputVals[dep]; ok {
		return true
	}
	// Check if it's a context field
	if _, ok := c.contextVals[dep]; ok {
		return true
	}
	// Check if it's a computed field that's been evaluated
	return evaluated[dep]
}

// buildScope builds the evaluation scope with nested structures
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
	for k, v := range c.outputVals {
		contextScope[k] = v
	}
	if len(contextScope) > 0 {
		scope["context"] = contextScope
	}

	return scope
}

// SetContextField sets the context variable by name
// If the variable doesn't exist an error is thrown
func (c *Context) SetContextField(name string, value string) error {
	if _, ok := c.contextVals[name]; ok {
		c.contextVals[name] = value
	}

	return errors.Errorf("context variable %s undefined", name)
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
