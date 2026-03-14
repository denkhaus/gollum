package variables

import (
	"fmt"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/errors"
)

// ComputedField represents a single computed field with its definition and cached value
type ComputedField struct {
	Name         string
	Type         ValueType
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
func NewComputedValues(computedFields []flows.ComputedField) *ComputedValues {
	cv := &ComputedValues{
		fields: make(map[string]*ComputedField),
	}

	if computedFields == nil {
		return cv
	}

	parser := NewExpressionParser()

	for _, field := range computedFields {
		deps := parser.ExtractDependencies(field.When)

		var valueType ValueType
		switch field.Type {
		case "bool":
			valueType = TypeBool
		case "int":
			valueType = TypeInt
		case "string":
			valueType = TypeString
		case "float":
			valueType = TypeFloat
		default:
			valueType = TypeString // default fallback
		}

		cv.fields[field.Name] = &ComputedField{
			Name:         field.Name,
			Type:         valueType,
			Expression:   field.When,
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
			Scope: "computed",
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
