package variables

import (
	"fmt"

	"github.com/denkhaus/gollum/pkg/flows"
)

// FieldValue is a type-safe wrapper for values
type FieldValue struct {
	valueType flows.ValueType
	value     any
	isSet     bool // true if value was explicitly set, false if uninitialized
}

// NewStringValue creates a string FieldValue
func NewStringValue(v string) FieldValue {
	return FieldValue{valueType: flows.TypeString, value: v, isSet: true}
}

// NewIntValue creates an int FieldValue
func NewIntValue(v int) FieldValue {
	return FieldValue{valueType: flows.TypeInt, value: v, isSet: true}
}

// NewBoolValue creates a bool FieldValue
func NewBoolValue(v bool) FieldValue {
	return FieldValue{valueType: flows.TypeBool, value: v, isSet: true}
}

// NewFloatValue creates a float FieldValue
func NewFloatValue(v float64) FieldValue {
	return FieldValue{valueType: flows.TypeFloat, value: v, isSet: true}
}

// NewUnsetStringValue creates an unset string FieldValue (placeholder for uninitialized fields)
func NewUnsetStringValue() FieldValue {
	return FieldValue{valueType: flows.TypeString, isSet: false}
}

// NewUnsetIntValue creates an unset int FieldValue (placeholder for uninitialized fields)
func NewUnsetIntValue() FieldValue {
	return FieldValue{valueType: flows.TypeInt, isSet: false}
}

// NewUnsetBoolValue creates an unset bool FieldValue (placeholder for uninitialized fields)
func NewUnsetBoolValue() FieldValue {
	return FieldValue{valueType: flows.TypeBool, isSet: false}
}

// NewUnsetFloatValue creates an unset float FieldValue (placeholder for uninitialized fields)
func NewUnsetFloatValue() FieldValue {
	return FieldValue{valueType: flows.TypeFloat, isSet: false}
}

// Type returns the value type
func (fv FieldValue) Type() flows.ValueType {
	return fv.valueType
}

// IsSet returns true if the value was explicitly set (not just initialized)
func (fv FieldValue) IsSet() bool {
	return fv.isSet
}

// String returns the string value
func (fv FieldValue) String() (string, error) {
	if fv.valueType != flows.TypeString {
		return "", fmt.Errorf("type mismatch: expected %s, got %s", flows.TypeString, fv.valueType)
	}
	return fv.value.(string), nil
}

// Int returns the int value
func (fv FieldValue) Int() (int, error) {
	if fv.valueType != flows.TypeInt {
		return 0, fmt.Errorf("type mismatch: expected %s, got %s", flows.TypeInt, fv.valueType)
	}
	return fv.value.(int), nil
}

// Bool returns the bool value
func (fv FieldValue) Bool() (bool, error) {
	if fv.valueType != flows.TypeBool {
		return false, fmt.Errorf("type mismatch: expected %s, got %s", flows.TypeBool, fv.valueType)
	}
	return fv.value.(bool), nil
}

// Float returns the float value
func (fv FieldValue) Float() (float64, error) {
	if fv.valueType != flows.TypeFloat {
		return 0, fmt.Errorf("type mismatch: expected %s, got %s", flows.TypeFloat, fv.valueType)
	}
	return fv.value.(float64), nil
}
