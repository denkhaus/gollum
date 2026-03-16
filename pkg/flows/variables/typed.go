package variables

import (
	"fmt"
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
	isSet     bool // true if value was explicitly set, false if uninitialized
}

// NewStringValue creates a string FieldValue
func NewStringValue(v string) FieldValue {
	return FieldValue{valueType: TypeString, value: v, isSet: true}
}

// NewIntValue creates an int FieldValue
func NewIntValue(v int) FieldValue {
	return FieldValue{valueType: TypeInt, value: v, isSet: true}
}

// NewBoolValue creates a bool FieldValue
func NewBoolValue(v bool) FieldValue {
	return FieldValue{valueType: TypeBool, value: v, isSet: true}
}

// NewFloatValue creates a float FieldValue
func NewFloatValue(v float64) FieldValue {
	return FieldValue{valueType: TypeFloat, value: v, isSet: true}
}

// NewUnsetStringValue creates an unset string FieldValue (placeholder for uninitialized fields)
func NewUnsetStringValue() FieldValue {
	return FieldValue{valueType: TypeString, isSet: false}
}

// NewUnsetIntValue creates an unset int FieldValue (placeholder for uninitialized fields)
func NewUnsetIntValue() FieldValue {
	return FieldValue{valueType: TypeInt, isSet: false}
}

// NewUnsetBoolValue creates an unset bool FieldValue (placeholder for uninitialized fields)
func NewUnsetBoolValue() FieldValue {
	return FieldValue{valueType: TypeBool, isSet: false}
}

// NewUnsetFloatValue creates an unset float FieldValue (placeholder for uninitialized fields)
func NewUnsetFloatValue() FieldValue {
	return FieldValue{valueType: TypeFloat, isSet: false}
}

// Type returns the value type
func (fv FieldValue) Type() ValueType {
	return fv.valueType
}

// IsSet returns true if the value was explicitly set (not just initialized)
func (fv FieldValue) IsSet() bool {
	return fv.isSet
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

