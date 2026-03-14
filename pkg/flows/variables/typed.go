package variables

import (
	"fmt"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/errors"
)

// ValueType represents the type of a field value
type ValueType string

const (
	TypeString ValueType = "string"
	TypeInt    ValueType = "int"
	TypeBool   ValueType = "bool"
	TypeFloat  ValueType = "float"
)

// FieldValue is a type-safe wrapper for values
type FieldValue struct {
	valueType ValueType
	value     any
}

// NewStringValue creates a string FieldValue
func NewStringValue(v string) FieldValue {
	return FieldValue{valueType: TypeString, value: v}
}

// NewIntValue creates an int FieldValue
func NewIntValue(v int) FieldValue {
	return FieldValue{valueType: TypeInt, value: v}
}

// NewBoolValue creates a bool FieldValue
func NewBoolValue(v bool) FieldValue {
	return FieldValue{valueType: TypeBool, value: v}
}

// NewFloatValue creates a float FieldValue
func NewFloatValue(v float64) FieldValue {
	return FieldValue{valueType: TypeFloat, value: v}
}

// Type returns the value type
func (fv FieldValue) Type() ValueType {
	return fv.valueType
}

// String returns the string value
func (fv FieldValue) String() (string, error) {
	if fv.valueType != TypeString {
		return "", fmt.Errorf("type mismatch: expected %s, got %s", TypeString, fv.valueType)
	}
	return fv.value.(string), nil
}

// Int returns the int value
func (fv FieldValue) Int() (int, error) {
	if fv.valueType != TypeInt {
		return 0, fmt.Errorf("type mismatch: expected %s, got %s", TypeInt, fv.valueType)
	}
	return fv.value.(int), nil
}

// Bool returns the bool value
func (fv FieldValue) Bool() (bool, error) {
	if fv.valueType != TypeBool {
		return false, fmt.Errorf("type mismatch: expected %s, got %s", TypeBool, fv.valueType)
	}
	return fv.value.(bool), nil
}

// Float returns the float value
func (fv FieldValue) Float() (float64, error) {
	if fv.valueType != TypeFloat {
		return 0, fmt.Errorf("type mismatch: expected %s, got %s", TypeFloat, fv.valueType)
	}
	return fv.value.(float64), nil
}

// InputValues stores input field values (immutable after SetInput)
type InputValues struct {
	fields map[string]FieldValue
	defs   map[string]flows.FieldDef
}

// NewInputValues creates a new InputValues from InputBlock
func NewInputValues(block *flows.InputBlock) *InputValues {
	iv := &InputValues{
		fields: make(map[string]FieldValue),
		defs:   make(map[string]flows.FieldDef),
	}

	if block == nil {
		return iv
	}

	for _, field := range block.GetAllFields() {
		iv.defs[field.Name] = field
	}

	return iv
}

// SetString sets a string input value
func (iv *InputValues) SetString(name string, value string) error {
	if _, ok := iv.defs[name]; !ok {
		return &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "input field not defined",
				Field:   name,
			},
			Scope: "input",
		}
	}

	if iv.defs[name].Type != string(TypeString) {
		return &errors.TypeError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeTypeMismatch,
				Message: "field type mismatch",
				Field:   name,
			},
			ExpectedType: iv.defs[name].Type,
			ActualType:   string(TypeString),
		}
	}

	iv.fields[name] = NewStringValue(value)
	return nil
}

// SetInt sets an int input value
func (iv *InputValues) SetInt(name string, value int) error {
	if _, ok := iv.defs[name]; !ok {
		return &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "input field not defined",
				Field:   name,
			},
			Scope: "input",
		}
	}

	if iv.defs[name].Type != string(TypeInt) {
		return &errors.TypeError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeTypeMismatch,
				Message: "field type mismatch",
				Field:   name,
			},
			ExpectedType: iv.defs[name].Type,
			ActualType:   string(TypeInt),
		}
	}

	iv.fields[name] = NewIntValue(value)
	return nil
}

// SetBool sets a bool input value
func (iv *InputValues) SetBool(name string, value bool) error {
	if _, ok := iv.defs[name]; !ok {
		return &errors.UnknownFieldError{FlowError: errors.FlowError{Code: errors.ErrCodeUnknownField, Message: "input field not defined", Field: name}, Scope: "input"}
	}
	if iv.defs[name].Type != string(TypeBool) {
		return &errors.TypeError{FlowError: errors.FlowError{Code: errors.ErrCodeTypeMismatch, Message: "field type mismatch", Field: name}, ExpectedType: iv.defs[name].Type, ActualType: string(TypeBool)}
	}
	iv.fields[name] = NewBoolValue(value)
	return nil
}

// SetFloat sets a float input value
func (iv *InputValues) SetFloat(name string, value float64) error {
	if _, ok := iv.defs[name]; !ok {
		return &errors.UnknownFieldError{FlowError: errors.FlowError{Code: errors.ErrCodeUnknownField, Message: "input field not defined", Field: name}, Scope: "input"}
	}
	if iv.defs[name].Type != string(TypeFloat) {
		return &errors.TypeError{FlowError: errors.FlowError{Code: errors.ErrCodeTypeMismatch, Message: "field type mismatch", Field: name}, ExpectedType: iv.defs[name].Type, ActualType: string(TypeFloat)}
	}
	iv.fields[name] = NewFloatValue(value)
	return nil
}

// GetString returns a string input value
func (iv *InputValues) GetString(name string) (string, error) {
	fv, ok := iv.fields[name]
	if !ok {
		return "", &errors.UnknownFieldError{FlowError: errors.FlowError{Code: errors.ErrCodeUnknownField, Message: "input field not set", Field: name}, Scope: "input"}
	}
	return fv.String()
}

// GetInt returns an int input value
func (iv *InputValues) GetInt(name string) (int, error) {
	fv, ok := iv.fields[name]
	if !ok {
		return 0, &errors.UnknownFieldError{FlowError: errors.FlowError{Code: errors.ErrCodeUnknownField, Message: "input field not set", Field: name}, Scope: "input"}
	}
	return fv.Int()
}

// GetBool returns a bool input value
func (iv *InputValues) GetBool(name string) (bool, error) {
	fv, ok := iv.fields[name]
	if !ok {
		return false, &errors.UnknownFieldError{FlowError: errors.FlowError{Code: errors.ErrCodeUnknownField, Message: "input field not set", Field: name}, Scope: "input"}
	}
	return fv.Bool()
}

// GetFloat returns a float input value
func (iv *InputValues) GetFloat(name string) (float64, error) {
	fv, ok := iv.fields[name]
	if !ok {
		return 0, &errors.UnknownFieldError{FlowError: errors.FlowError{Code: errors.ErrCodeUnknownField, Message: "input field not set", Field: name}, Scope: "input"}
	}
	return fv.Float()
}

// Has returns true if field has a value
func (iv *InputValues) Has(name string) bool {
	_, ok := iv.fields[name]
	return ok
}
