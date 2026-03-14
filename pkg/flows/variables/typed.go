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
