package variables

import (
	"fmt"
	"strconv"

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

// ComputedEvaluator is a placeholder for reactive evaluation (will be implemented later)
type ComputedEvaluator struct {
	// Will be implemented in a later task
}

// MarkDirty marks a computed field as needing re-evaluation
func (ce *ComputedEvaluator) MarkDirty(field string) {
	// Will be implemented in a later task
}

// ContextValues stores context field values (mutable by tools)
type ContextValues struct {
	fields    map[string]FieldValue
	defs      map[string]flows.ContextField
	evaluator *ComputedEvaluator // Will be set later for reactive updates
}

// NewContextValues creates a new ContextValues from ContextBlock
func NewContextValues(block *flows.ContextBlock) *ContextValues {
	cv := &ContextValues{
		fields: make(map[string]FieldValue),
		defs:   make(map[string]flows.ContextField),
	}

	if block == nil {
		return cv
	}

	// Register string fields
	for _, field := range block.Strings {
		cv.defs[field.Name] = field
		if field.Default != "" {
			cv.fields[field.Name] = NewStringValue(field.Default)
		}
	}

	// Register int fields
	for _, field := range block.Ints {
		cv.defs[field.Name] = field
		if field.Default != "" {
			if i, err := strconv.Atoi(field.Default); err == nil {
				cv.fields[field.Name] = NewIntValue(i)
			}
		}
	}

	// Register bool fields
	for _, field := range block.Bools {
		cv.defs[field.Name] = field
		if field.Default != "" {
			if b, err := strconv.ParseBool(field.Default); err == nil {
				cv.fields[field.Name] = NewBoolValue(b)
			}
		}
	}

	// Register float fields
	for _, field := range block.Floats {
		cv.defs[field.Name] = field
		if field.Default != "" {
			if f, err := strconv.ParseFloat(field.Default, 64); err == nil {
				cv.fields[field.Name] = NewFloatValue(f)
			}
		}
	}

	return cv
}

// SetString sets a string context value
func (cv *ContextValues) SetString(name string, value string) error {
	if _, ok := cv.defs[name]; !ok {
		return &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "context field not defined",
				Field:   name,
			},
			Scope: "context",
		}
	}

	def, ok := cv.defs[name]
	if !ok || def.Type != string(TypeString) {
		return &errors.TypeError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeTypeMismatch,
				Message: "field type mismatch",
				Field:   name,
			},
			ExpectedType: def.Type,
			ActualType:   string(TypeString),
		}
	}

	// Check if value changed
	oldValue, hadOld := cv.fields[name]
	cv.fields[name] = NewStringValue(value)

	// Notify evaluator of change
	if cv.evaluator != nil {
		if !hadOld || oldValue.value != value {
			cv.evaluator.MarkDirty(name)
		}
	}

	return nil
}

// SetInt sets an int context value
func (cv *ContextValues) SetInt(name string, value int) error {
	if _, ok := cv.defs[name]; !ok {
		return &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "context field not defined",
				Field:   name,
			},
			Scope: "context",
		}
	}

	def, ok := cv.defs[name]
	if !ok || def.Type != string(TypeInt) {
		return &errors.TypeError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeTypeMismatch,
				Message: "field type mismatch",
				Field:   name,
			},
			ExpectedType: def.Type,
			ActualType:   string(TypeInt),
		}
	}

	// Check if value changed
	oldValue, hadOld := cv.fields[name]
	cv.fields[name] = NewIntValue(value)

	// Notify evaluator of change
	if cv.evaluator != nil {
		if !hadOld || oldValue.value != value {
			cv.evaluator.MarkDirty(name)
		}
	}

	return nil
}

// SetBool sets a bool context value
func (cv *ContextValues) SetBool(name string, value bool) error {
	if _, ok := cv.defs[name]; !ok {
		return &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "context field not defined",
				Field:   name,
			},
			Scope: "context",
		}
	}

	def, ok := cv.defs[name]
	if !ok || def.Type != string(TypeBool) {
		return &errors.TypeError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeTypeMismatch,
				Message: "field type mismatch",
				Field:   name,
			},
			ExpectedType: def.Type,
			ActualType:   string(TypeBool),
		}
	}

	// Check if value changed
	oldValue, hadOld := cv.fields[name]
	cv.fields[name] = NewBoolValue(value)

	// Notify evaluator of change
	if cv.evaluator != nil {
		if !hadOld || oldValue.value != value {
			cv.evaluator.MarkDirty(name)
		}
	}

	return nil
}

// SetFloat sets a float context value
func (cv *ContextValues) SetFloat(name string, value float64) error {
	if _, ok := cv.defs[name]; !ok {
		return &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "context field not defined",
				Field:   name,
			},
			Scope: "context",
		}
	}

	def, ok := cv.defs[name]
	if !ok || def.Type != string(TypeFloat) {
		return &errors.TypeError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeTypeMismatch,
				Message: "field type mismatch",
				Field:   name,
			},
			ExpectedType: def.Type,
			ActualType:   string(TypeFloat),
		}
	}

	// Check if value changed
	oldValue, hadOld := cv.fields[name]
	cv.fields[name] = NewFloatValue(value)

	// Notify evaluator of change
	if cv.evaluator != nil {
		if !hadOld || oldValue.value != value {
			cv.evaluator.MarkDirty(name)
		}
	}

	return nil
}

// GetString returns a string context value
func (cv *ContextValues) GetString(name string) (string, error) {
	fv, ok := cv.fields[name]
	if !ok {
		return "", &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "context field not set",
				Field:   name,
			},
			Scope: "context",
		}
	}
	return fv.String()
}

// GetInt returns an int context value
func (cv *ContextValues) GetInt(name string) (int, error) {
	fv, ok := cv.fields[name]
	if !ok {
		return 0, &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "context field not set",
				Field:   name,
			},
			Scope: "context",
		}
	}
	return fv.Int()
}

// GetBool returns a bool context value
func (cv *ContextValues) GetBool(name string) (bool, error) {
	fv, ok := cv.fields[name]
	if !ok {
		return false, &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "context field not set",
				Field:   name,
			},
			Scope: "context",
		}
	}
	return fv.Bool()
}

// GetFloat returns a float context value
func (cv *ContextValues) GetFloat(name string) (float64, error) {
	fv, ok := cv.fields[name]
	if !ok {
		return 0, &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "context field not set",
				Field:   name,
			},
			Scope: "context",
		}
	}
	return fv.Float()
}

// Has returns true if field has a value
func (cv *ContextValues) Has(name string) bool {
	_, ok := cv.fields[name]
	return ok
}

// SetEvaluator sets the computed evaluator for reactive updates
func (cv *ContextValues) SetEvaluator(eval *ComputedEvaluator) {
	cv.evaluator = eval
}
