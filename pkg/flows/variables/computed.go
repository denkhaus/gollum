package variables

import (
	"fmt"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/errors"
)

// Evaluator interface for reactive updates
type Evaluator interface {
	MarkDirty(changedField string)
	ComputeDirty() error
	GetValue(name string) (any, error)
}

// ComputedField represents a single computed field with its definition and cached value
type ComputedField struct {
	Name         string
	Type         flows.ValueType
	Expression   string
	Dependencies []FieldReference
	lastValue    *FieldValue
	dirty        bool
}

// ComputedValues stores computed field definitions and their cached values
type ComputedValues struct {
	fields map[string]*ComputedField
}

// NewComputedValues creates a new ComputedValues from a slice of ComputedField definitions
func NewComputedValues(computedFields []flows.ComputedFieldDef) *ComputedValues {
	cv := &ComputedValues{
		fields: make(map[string]*ComputedField),
	}

	if computedFields == nil {
		return cv
	}

	parser := NewExpressionParser()

	for _, field := range computedFields {
		deps := parser.ExtractDependencies(field.Eval)

		var valueType flows.ValueType
		switch string(field.Type) {
		case string(flows.TypeBool):
			valueType = flows.TypeBool
		case string(flows.TypeInt):
			valueType = flows.TypeInt
		case string(flows.TypeString):
			valueType = flows.TypeString
		case string(flows.TypeFloat):
			valueType = flows.TypeFloat
		default:
			valueType = flows.TypeString // default fallback
		}

		cv.fields[field.Name] = &ComputedField{
			Name:         field.Name,
			Type:         valueType,
			Expression:   field.Eval,
			Dependencies: deps,
			dirty:        true, // Needs initial evaluation
		}
	}

	return cv
}

// Has returns true if the computed field exists
func (cv *ComputedValues) Has(name string) bool {
	_, ok := cv.fields[name]
	return ok
}

// GetField returns the computed field definition
func (cv *ComputedValues) GetField(name string) (*ComputedField, error) {
	field, ok := cv.fields[name]
	if !ok {
		return nil, &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "computed field not defined",
				Field:   name,
			},
			Scope: flows.FlowVariableScopeComputed,
		}
	}
	return field, nil
}

// MarkDirty marks a computed field as needing re-evaluation
func (cv *ComputedValues) MarkDirty(name string) {
	if field, ok := cv.fields[name]; ok {
		field.dirty = true
	}
}

// IsDirty returns true if the field needs evaluation
func (cv *ComputedValues) IsDirty(name string) bool {
	if field, ok := cv.fields[name]; ok {
		return field.dirty
	}
	return false
}

// SetValue sets the cached value for a computed field
func (cv *ComputedValues) SetValue(name string, value FieldValue) {
	if field, ok := cv.fields[name]; ok {
		field.lastValue = &value
		field.dirty = false
	}
}

// GetBool returns a bool computed value
func (cv *ComputedValues) GetBool(name string) (bool, error) {
	field, err := cv.GetField(name)
	if err != nil {
		return false, err
	}
	if field.lastValue == nil || field.dirty {
		return false, fmt.Errorf("computed field not evaluated: %s", name)
	}
	return field.lastValue.Bool()
}

// GetInt returns an int computed value
func (cv *ComputedValues) GetInt(name string) (int, error) {
	field, err := cv.GetField(name)
	if err != nil {
		return 0, err
	}
	if field.lastValue == nil || field.dirty {
		return 0, fmt.Errorf("computed field not evaluated: %s", name)
	}
	return field.lastValue.Int()
}

// GetString returns a string computed value
func (cv *ComputedValues) GetString(name string) (string, error) {
	field, err := cv.GetField(name)
	if err != nil {
		return "", err
	}
	if field.lastValue == nil || field.dirty {
		return "", fmt.Errorf("computed field not evaluated: %s", name)
	}
	return field.lastValue.String()
}

// GetFloat returns a float computed value
func (cv *ComputedValues) GetFloat(name string) (float64, error) {
	field, err := cv.GetField(name)
	if err != nil {
		return 0, err
	}
	if field.lastValue == nil || field.dirty {
		return 0, fmt.Errorf("computed field not evaluated: %s", name)
	}
	return field.lastValue.Float()
}

// GetAll returns all computed field definitions
func (cv *ComputedValues) GetAll() map[string]*ComputedField {
	return cv.fields
}

// GetValue returns a computed value as any (evaluates if dirty)
func (cv *ComputedValues) GetValue(name string) (any, error) {
	field, err := cv.GetField(name)
	if err != nil {
		return nil, err
	}
	if field.lastValue == nil || field.dirty {
		return nil, fmt.Errorf("computed field not evaluated: %s", name)
	}
	switch field.Type {
	case flows.TypeBool:
		return field.lastValue.Bool()
	case flows.TypeInt:
		return field.lastValue.Int()
	case flows.TypeString:
		return field.lastValue.String()
	case flows.TypeFloat:
		return field.lastValue.Float()
	default:
		return nil, fmt.Errorf("unknown type: %s", field.Type)
	}
}

// GetDependents returns all computed fields that depend on the given field
func (cv *ComputedValues) GetDependents(fieldRef FieldReference) []string {
	var dependents []string

	for _, field := range cv.fields {
		for _, dep := range field.Dependencies {
			if dep.Scope == fieldRef.Scope && dep.Name == fieldRef.Name {
				dependents = append(dependents, field.Name)
				break
			}
		}
	}

	return dependents
}

// ComputedEvaluator manages reactive computed field evaluation
type ComputedEvaluator struct {
	computed   *ComputedValues
	input      *FieldValues[flows.FieldDef]
	context    *FieldValues[flows.ContextField]
	output     *FieldValues[flows.FieldDef]
	exprEval   *ExpressionEvaluator
	dependents map[string][]string // fieldRef -> computed fields that depend on it
	evaluating map[string]bool     // for cycle detection
}

// NewComputedEvaluator creates a new computed evaluator
func NewComputedEvaluator(computed *ComputedValues, input *FieldValues[flows.FieldDef],
	context *FieldValues[flows.ContextField], output *FieldValues[flows.FieldDef]) *ComputedEvaluator {

	evaluator := &ComputedEvaluator{
		computed:   computed,
		input:      input,
		context:    context,
		output:     output,
		exprEval:   NewExpressionEvaluator(),
		dependents: make(map[string][]string),
		evaluating: make(map[string]bool),
	}

	// Build dependency graph
	evaluator.buildDependentsMap()

	return evaluator
}

// buildDependentsMap builds the reverse dependency graph
func (ce *ComputedEvaluator) buildDependentsMap() {
	for _, field := range ce.computed.GetAll() {
		for _, dep := range field.Dependencies {
			key := dep.Key()
			ce.dependents[key] = append(ce.dependents[key], field.Name)
		}
	}
}

// MarkDirty marks a computed field and its dependents as dirty
func (ce *ComputedEvaluator) MarkDirty(changedField string) {
	// This is called when a context/input/output field changes
	key := "context." + changedField // Assuming context for now

	// Mark direct dependents
	for _, depName := range ce.dependents[key] {
		ce.computed.MarkDirty(depName)
		// Recursively mark dependents of dependents
		ce.markDependentsDirty(depName)
	}
}

// markDependentsDirty recursively marks dependents dirty
func (ce *ComputedEvaluator) markDependentsDirty(computedName string) {
	key := "computed." + computedName

	for _, depName := range ce.dependents[key] {
		ce.computed.MarkDirty(depName)
		ce.markDependentsDirty(depName)
	}
}

// ComputeDirty evaluates all dirty computed fields
func (ce *ComputedEvaluator) ComputeDirty() error {
	evaluated := make(map[string]bool)

	for name, field := range ce.computed.GetAll() {
		if field.dirty && !evaluated[name] {
			if err := ce.evaluateField(name, evaluated); err != nil {
				return err
			}
		}
	}

	return nil
}

// evaluateField evaluates a single computed field
func (ce *ComputedEvaluator) evaluateField(name string, evaluated map[string]bool) error {
	// Check for cycles
	if ce.evaluating[name] {
		return &errors.CircularDependencyError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeCircularDep,
				Message: "circular dependency detected",
			},
			Cycle: ce.extractCycle(name),
		}
	}

	ce.evaluating[name] = true
	defer delete(ce.evaluating, name)

	field, err := ce.computed.GetField(name)
	if err != nil {
		return err
	}

	// First, evaluate all dependencies
	for _, dep := range field.Dependencies {
		if dep.Scope == flows.FlowVariableScopeComputed && !evaluated[dep.Name] {
			if err := ce.evaluateField(dep.Name, evaluated); err != nil {
				return err
			}
		}
	}

	// Build evaluation scope
	scope := ce.buildScope()

	// Evaluate the expression
	result, err := ce.exprEval.Evaluate(field.Expression, scope)
	if err != nil {
		return &errors.ExpressionError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeExpression,
				Message: "expression evaluation failed",
				Field:   name,
				Cause:   err,
			},
			Expression: field.Expression,
		}
	}

	// Store the result
	var fv FieldValue
	switch field.Type {
	case flows.TypeBool:
		boolVal, ok := result.(bool)
		if !ok {
			return &errors.TypeError{
				FlowError: errors.FlowError{
					Code:    errors.ErrCodeTypeMismatch,
					Message: "type mismatch",
					Field:   name,
				},
				ExpectedType: flows.TypeBool,
				ActualType:   flows.ValueType(fmt.Sprintf("%T", result)),
			}
		}
		fv = NewBoolValue(boolVal)
	case flows.TypeInt:
		intVal, ok := result.(int)
		if !ok {
			if floatVal, ok := result.(float64); ok {
				intVal = int(floatVal)
			} else {
				return &errors.TypeError{
					FlowError: errors.FlowError{
						Code:    errors.ErrCodeTypeMismatch,
						Message: "type mismatch",
						Field:   name,
					},
					ExpectedType: flows.TypeInt,
					ActualType:   flows.ValueType(fmt.Sprintf("%T", result)),
				}
			}
		}
		fv = NewIntValue(intVal)
	case flows.TypeString:
		strVal, ok := result.(string)
		if !ok {
			return &errors.TypeError{
				FlowError: errors.FlowError{
					Code:    errors.ErrCodeTypeMismatch,
					Message: "type mismatch",
					Field:   name,
				},
				ExpectedType: flows.TypeString,
				ActualType:   flows.ValueType(fmt.Sprintf("%T", result)),
			}
		}
		fv = NewStringValue(strVal)
	case flows.TypeFloat:
		floatVal, ok := result.(float64)
		if !ok {
			return &errors.TypeError{
				FlowError: errors.FlowError{
					Code:    errors.ErrCodeTypeMismatch,
					Message: "type mismatch",
					Field:   name,
				},
				ExpectedType: flows.TypeFloat,
				ActualType:   flows.ValueType(fmt.Sprintf("%T", result)),
			}
		}
		fv = NewFloatValue(floatVal)
	default:
		return fmt.Errorf("unknown type: %s", field.Type)
	}

	ce.computed.SetValue(name, fv)
	evaluated[name] = true

	return nil
}

// buildScope builds the evaluation scope
func (ce *ComputedEvaluator) buildScope() *EvaluationScope {
	scope := &EvaluationScope{
		Input:    make(map[string]any),
		Context:  make(map[string]any),
		Output:   make(map[string]any),
		Computed: make(map[string]any),
	}

	// Populate from input container
	if ce.input != nil {
		// Use Has to iterate over defined fields
		for name, fieldDef := range ce.input.defs {
			if val, err := ce.input.GetString(name); err == nil {
				scope.Input[name] = val
			} else if val, err := ce.input.GetInt(name); err == nil {
				scope.Input[name] = val
			} else if val, err := ce.input.GetBool(name); err == nil {
				scope.Input[name] = val
			} else if val, err := ce.input.GetFloat(name); err == nil {
				scope.Input[name] = val
			} else if val, ok := ce.input.GetRaw(name); ok {
				// Fallback for raw values (set without type)
				scope.Input[name] = val
			}
			_ = fieldDef // Use the variable to avoid unused warning
		}
	}

	// Populate from context container
	if ce.context != nil {
		// Use Has to iterate over defined fields
		for name, fieldDef := range ce.context.defs {
			if val, err := ce.context.GetString(name); err == nil {
				scope.Context[name] = val
			} else if val, err := ce.context.GetInt(name); err == nil {
				scope.Context[name] = val
			} else if val, err := ce.context.GetBool(name); err == nil {
				scope.Context[name] = val
			} else if val, err := ce.context.GetFloat(name); err == nil {
				scope.Context[name] = val
			} else if val, ok := ce.context.GetRaw(name); ok {
				// Fallback for raw values (set without type)
				scope.Context[name] = val
			} else {
				// Field is defined but not set - add nil to represent unset state
				scope.Context[name] = nil
			}
			_ = fieldDef // Use the variable to avoid unused warning
		}
	}

	// Populate from output container
	if ce.output != nil {
		// Use Has to iterate over defined fields
		for name, fieldDef := range ce.output.defs {
			if val, err := ce.output.GetString(name); err == nil {
				scope.Output[name] = val
			} else if val, err := ce.output.GetInt(name); err == nil {
				scope.Output[name] = val
			} else if val, err := ce.output.GetBool(name); err == nil {
				scope.Output[name] = val
			} else if val, err := ce.output.GetFloat(name); err == nil {
				scope.Output[name] = val
			} else if val, ok := ce.output.GetRaw(name); ok {
				// Fallback for raw values (set without type)
				scope.Output[name] = val
			} else {
				// Field is defined but not set - add nil to represent unset state
				scope.Output[name] = nil
			}
			_ = fieldDef // Use the variable to avoid unused warning
		}
	}

	// Populate from computed container
	if ce.computed != nil {
		for name, field := range ce.computed.GetAll() {
			if field.lastValue != nil && !field.dirty {
				if val, err := field.lastValue.Bool(); err == nil {
					scope.Computed[name] = val
				} else if val, err := field.lastValue.Int(); err == nil {
					scope.Computed[name] = val
				} else if val, err := field.lastValue.String(); err == nil {
					scope.Computed[name] = val
				} else if val, err := field.lastValue.Float(); err == nil {
					scope.Computed[name] = val
				}
			}
		}
	}

	return scope
}

// extractCycle extracts the cycle from the current evaluation state
func (ce *ComputedEvaluator) extractCycle(startNode string) []string {
	cycle := []string{startNode}
	for name := range ce.evaluating {
		if name != startNode {
			cycle = append([]string{name}, cycle...)
		}
	}
	return cycle
}

// GetValue returns a computed value (evaluates if dirty)
func (ce *ComputedEvaluator) GetValue(name string) (any, error) {
	field, err := ce.computed.GetField(name)
	if err != nil {
		return nil, err
	}

	if field.dirty {
		if err := ce.ComputeDirty(); err != nil {
			return nil, err
		}
	}

	switch field.Type {
	case flows.TypeBool:
		return ce.computed.GetBool(name)
	case flows.TypeInt:
		return ce.computed.GetInt(name)
	case flows.TypeString:
		return ce.computed.GetString(name)
	case flows.TypeFloat:
		return ce.computed.GetFloat(name)
	default:
		return nil, fmt.Errorf("unknown type: %s", field.Type)
	}
}
