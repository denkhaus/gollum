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
		// Set default value if available
		if field.Default != "" {
			switch ValueType(field.Type) {
			case TypeString:
				iv.fields[field.Name] = NewStringValue(field.Default)
			case TypeInt:
				if i, err := strconv.Atoi(field.Default); err == nil {
					iv.fields[field.Name] = NewIntValue(i)
				}
			case TypeBool:
				if b, err := strconv.ParseBool(field.Default); err == nil {
					iv.fields[field.Name] = NewBoolValue(b)
				}
			case TypeFloat:
				if f, err := strconv.ParseFloat(field.Default, 64); err == nil {
					iv.fields[field.Name] = NewFloatValue(f)
				}
			}
		} else {
			// Initialize with unset placeholder
			switch ValueType(field.Type) {
			case TypeString:
				iv.fields[field.Name] = NewUnsetStringValue()
			case TypeInt:
				iv.fields[field.Name] = NewUnsetIntValue()
			case TypeBool:
				iv.fields[field.Name] = NewUnsetBoolValue()
			case TypeFloat:
				iv.fields[field.Name] = NewUnsetFloatValue()
			}
		}
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
	if !fv.IsSet() {
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
	if !fv.IsSet() {
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
	if !fv.IsSet() {
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
	if !fv.IsSet() {
		return 0, &errors.UnknownFieldError{FlowError: errors.FlowError{Code: errors.ErrCodeUnknownField, Message: "input field not set", Field: name}, Scope: "input"}
	}
	return fv.Float()
}

// Has returns true if field has a value
func (iv *InputValues) Has(name string) bool {
	_, ok := iv.fields[name]
	return ok
}

// SetRaw sets a value without type validation (for computed field evaluation)
func (iv *InputValues) SetRaw(name string, value any) {
	iv.fields[name] = FieldValue{value: value}
}

// GetRaw gets a value without type checking
func (iv *InputValues) GetRaw(name string) (any, bool) {
	fv, ok := iv.fields[name]
	if !ok || !fv.IsSet() {
		return nil, false
	}
	// Return the raw value for set fields
	return fv.value, true
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
		field.Type = string(TypeString) // Set type explicitly
		cv.defs[field.Name] = field
		if field.Default != "" {
			cv.fields[field.Name] = NewStringValue(field.Default)
		} else {
			// Initialize with unset placeholder
			cv.fields[field.Name] = NewUnsetStringValue()
		}
	}

	// Register int fields
	for _, field := range block.Ints {
		field.Type = string(TypeInt) // Set type explicitly
		cv.defs[field.Name] = field
		if field.Default != "" {
			if i, err := strconv.Atoi(field.Default); err == nil {
				cv.fields[field.Name] = NewIntValue(i)
			}
		} else {
			// Initialize with unset placeholder
			cv.fields[field.Name] = NewUnsetIntValue()
		}
	}

	// Register bool fields
	for _, field := range block.Bools {
		field.Type = string(TypeBool) // Set type explicitly
		cv.defs[field.Name] = field
		if field.Default != "" {
			if b, err := strconv.ParseBool(field.Default); err == nil {
				cv.fields[field.Name] = NewBoolValue(b)
			}
		} else {
			// Initialize with unset placeholder
			cv.fields[field.Name] = NewUnsetBoolValue()
		}
	}

	// Register float fields
	for _, field := range block.Floats {
		field.Type = string(TypeFloat) // Set type explicitly
		cv.defs[field.Name] = field
		if field.Default != "" {
			if f, err := strconv.ParseFloat(field.Default, 64); err == nil {
				cv.fields[field.Name] = NewFloatValue(f)
			}
		} else {
			// Initialize with unset placeholder
			cv.fields[field.Name] = NewUnsetFloatValue()
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
				Message: "context field not defined",
				Field:   name,
			},
			Scope: "context",
		}
	}
	if !fv.IsSet() {
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
				Message: "context field not defined",
				Field:   name,
			},
			Scope: "context",
		}
	}
	if !fv.IsSet() {
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
				Message: "context field not defined",
				Field:   name,
			},
			Scope: "context",
		}
	}
	if !fv.IsSet() {
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
				Message: "context field not defined",
				Field:   name,
			},
			Scope: "context",
		}
	}
	if !fv.IsSet() {
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

// Has returns true if field has a value (is defined and set)
func (cv *ContextValues) Has(name string) bool {
	fv, ok := cv.fields[name]
	return ok && fv.IsSet()
}

// SetRaw sets a value without type validation (for computed field evaluation)
// Preserves the valueType from the schema definition
func (cv *ContextValues) SetRaw(name string, value any) {
	// Preserve the valueType from the existing field (schema definition)
	existing, ok := cv.fields[name]
	valueType := TypeString // default fallback
	if ok {
		valueType = existing.valueType
	}
	cv.fields[name] = FieldValue{value: value, valueType: valueType, isSet: true}
}

// GetRaw gets a value without type checking
func (cv *ContextValues) GetRaw(name string) (any, bool) {
	fv, ok := cv.fields[name]
	if !ok || !fv.IsSet() {
		return nil, false
	}
	// If valueType is not set, return the raw value
	if fv.valueType == "" {
		return fv.value, true
	}
	// Otherwise try to get the typed value
	switch fv.valueType {
	case TypeString:
		val, err := fv.String()
		return val, err == nil
	case TypeInt:
		val, err := fv.Int()
		return val, err == nil
	case TypeBool:
		val, err := fv.Bool()
		return val, err == nil
	case TypeFloat:
		val, err := fv.Float()
		return val, err == nil
	}
	return nil, false
}

// SetEvaluator sets the computed evaluator for reactive updates
func (cv *ContextValues) SetEvaluator(eval *ComputedEvaluator) {
	cv.evaluator = eval
}

// SetFromString sets a context field value from a string, automatically converting to the target type.
// Returns error if the field is not defined or if type conversion fails.
// This is used by flow tools that receive string values from LLMs.
func (cv *ContextValues) SetFromString(name string, value string) error {
	// Get the field definition to determine target type
	fieldDef, exists := cv.defs[name]
	if !exists {
		return &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "context field not defined",
				Field:   name,
			},
			Scope: "context",
		}
	}

	// Convert string value to the appropriate type based on field definition
	switch fieldDef.Type {
	case "string":
		return cv.SetString(name, value)
	case "int":
		// Try parsing as int
		i, err := strconv.Atoi(value)
		if err != nil {
			return &errors.TypeError{
				FlowError: errors.FlowError{
					Code:    errors.ErrCodeTypeMismatch,
					Message: fmt.Sprintf("cannot convert '%s' to int: %v", value, err),
					Field:   name,
				},
			}
		}
		return cv.SetInt(name, i)
	case "bool":
		// Try parsing as bool
		b, err := strconv.ParseBool(value)
		if err != nil {
			return &errors.TypeError{
				FlowError: errors.FlowError{
					Code:    errors.ErrCodeTypeMismatch,
					Message: fmt.Sprintf("cannot convert '%s' to bool: %v", value, err),
					Field:   name,
				},
			}
		}
		return cv.SetBool(name, b)
	case "float":
		// Try parsing as float
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return &errors.TypeError{
				FlowError: errors.FlowError{
					Code:    errors.ErrCodeTypeMismatch,
					Message: fmt.Sprintf("cannot convert '%s' to float: %v", value, err),
					Field:   name,
				},
			}
		}
		return cv.SetFloat(name, f)
	default:
		return fmt.Errorf("unknown field type '%s' for field '%s'", fieldDef.Type, name)
	}
}

// GetAllFields returns all context field definitions
func (cv *ContextValues) GetAllFields() []flows.ContextField {
	fields := make([]flows.ContextField, 0, len(cv.defs))
	for _, field := range cv.defs {
		fields = append(fields, field)
	}
	return fields
}

// OutputValues stores output field values (write-once per field)
type OutputValues struct {
	fields  map[string]FieldValue
	defs    map[string]flows.FieldDef
	written map[string]bool
}

// NewOutputValues creates a new OutputValues from OutputBlock
func NewOutputValues(block *flows.OutputBlock) *OutputValues {
	ov := &OutputValues{
		fields:  make(map[string]FieldValue),
		defs:    make(map[string]flows.FieldDef),
		written: make(map[string]bool),
	}

	if block == nil {
		return ov
	}

	for _, field := range block.GetAllFields() {
		ov.defs[field.Name] = field
		// Initialize with unset placeholder based on type
		switch ValueType(field.Type) {
		case TypeString:
			ov.fields[field.Name] = NewUnsetStringValue()
		case TypeInt:
			ov.fields[field.Name] = NewUnsetIntValue()
		case TypeBool:
			ov.fields[field.Name] = NewUnsetBoolValue()
		case TypeFloat:
			ov.fields[field.Name] = NewUnsetFloatValue()
		default:
			// Default to string unset for unknown types
			ov.fields[field.Name] = NewUnsetStringValue()
		}
	}

	return ov
}

// SetString sets a string output value (write-once)
func (ov *OutputValues) SetString(name string, value string) error {
	if ov.written[name] {
		return &errors.ImmutableFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeImmutable,
				Message: "output field already written",
				Field:   name,
			},
			AttemptedOperation: "SetString",
		}
	}

	if _, ok := ov.defs[name]; !ok {
		return &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "output field not defined",
				Field:   name,
			},
			Scope: "output",
		}
	}

	if ov.defs[name].Type != string(TypeString) {
		return &errors.TypeError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeTypeMismatch,
				Message: "field type mismatch",
				Field:   name,
			},
			ExpectedType: ov.defs[name].Type,
			ActualType:   string(TypeString),
		}
	}

	ov.fields[name] = NewStringValue(value)
	ov.written[name] = true
	return nil
}

// SetInt sets an int output value (write-once)
func (ov *OutputValues) SetInt(name string, value int) error {
	if ov.written[name] {
		return &errors.ImmutableFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeImmutable,
				Message: "output field already written",
				Field:   name,
			},
			AttemptedOperation: "SetInt",
		}
	}

	if _, ok := ov.defs[name]; !ok {
		return &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "output field not defined",
				Field:   name,
			},
			Scope: "output",
		}
	}

	if ov.defs[name].Type != string(TypeInt) {
		return &errors.TypeError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeTypeMismatch,
				Message: "field type mismatch",
				Field:   name,
			},
			ExpectedType: ov.defs[name].Type,
			ActualType:   string(TypeInt),
		}
	}

	ov.fields[name] = NewIntValue(value)
	ov.written[name] = true
	return nil
}

// SetBool sets a bool output value (write-once)
func (ov *OutputValues) SetBool(name string, value bool) error {
	if ov.written[name] {
		return &errors.ImmutableFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeImmutable,
				Message: "output field already written",
				Field:   name,
			},
			AttemptedOperation: "SetBool",
		}
	}

	if _, ok := ov.defs[name]; !ok {
		return &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "output field not defined",
				Field:   name,
			},
			Scope: "output",
		}
	}

	if ov.defs[name].Type != string(TypeBool) {
		return &errors.TypeError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeTypeMismatch,
				Message: "field type mismatch",
				Field:   name,
			},
			ExpectedType: ov.defs[name].Type,
			ActualType:   string(TypeBool),
		}
	}

	ov.fields[name] = NewBoolValue(value)
	ov.written[name] = true
	return nil
}

// SetFloat sets a float output value (write-once)
func (ov *OutputValues) SetFloat(name string, value float64) error {
	if ov.written[name] {
		return &errors.ImmutableFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeImmutable,
				Message: "output field already written",
				Field:   name,
			},
			AttemptedOperation: "SetFloat",
		}
	}

	if _, ok := ov.defs[name]; !ok {
		return &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "output field not defined",
				Field:   name,
			},
			Scope: "output",
		}
	}

	if ov.defs[name].Type != string(TypeFloat) {
		return &errors.TypeError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeTypeMismatch,
				Message: "field type mismatch",
				Field:   name,
			},
			ExpectedType: ov.defs[name].Type,
			ActualType:   string(TypeFloat),
		}
	}

	ov.fields[name] = NewFloatValue(value)
	ov.written[name] = true
	return nil
}

// GetString returns a string output value
func (ov *OutputValues) GetString(name string) (string, error) {
	fv, ok := ov.fields[name]
	if !ok {
		return "", &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "output field not defined",
				Field:   name,
			},
			Scope: "output",
		}
	}
	if !fv.IsSet() {
		return "", &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "output field not set",
				Field:   name,
			},
			Scope: "output",
		}
	}
	return fv.String()
}

// GetInt returns an int output value
func (ov *OutputValues) GetInt(name string) (int, error) {
	fv, ok := ov.fields[name]
	if !ok {
		return 0, &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "output field not defined",
				Field:   name,
			},
			Scope: "output",
		}
	}
	if !fv.IsSet() {
		return 0, &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "output field not set",
				Field:   name,
			},
			Scope: "output",
		}
	}
	return fv.Int()
}

// GetBool returns a bool output value
func (ov *OutputValues) GetBool(name string) (bool, error) {
	fv, ok := ov.fields[name]
	if !ok {
		return false, &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "output field not defined",
				Field:   name,
			},
			Scope: "output",
		}
	}
	if !fv.IsSet() {
		return false, &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "output field not set",
				Field:   name,
			},
			Scope: "output",
		}
	}
	return fv.Bool()
}

// GetFloat returns a float output value
func (ov *OutputValues) GetFloat(name string) (float64, error) {
	fv, ok := ov.fields[name]
	if !ok {
		return 0, &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "output field not defined",
				Field:   name,
			},
			Scope: "output",
		}
	}
	if !fv.IsSet() {
		return 0, &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "output field not set",
				Field:   name,
			},
			Scope: "output",
		}
	}
	return fv.Float()
}

// Has returns true if field has a value (is defined and set)
func (ov *OutputValues) Has(name string) bool {
	fv, ok := ov.fields[name]
	return ok && fv.IsSet()
}

// SetRaw sets a value without type validation (for computed field evaluation)
// Preserves the valueType from the schema definition
func (ov *OutputValues) SetRaw(name string, value any) {
	// Preserve the valueType from the existing field (schema definition)
	existing, ok := ov.fields[name]
	valueType := TypeString // default fallback
	if ok {
		valueType = existing.valueType
	}
	ov.fields[name] = FieldValue{value: value, valueType: valueType, isSet: true}
}

// GetRaw gets a value without type checking
func (ov *OutputValues) GetRaw(name string) (any, bool) {
	fv, ok := ov.fields[name]
	if !ok || !fv.IsSet() {
		return nil, false
	}
	// If valueType is not set, return the raw value
	if fv.valueType == "" {
		return fv.value, true
	}
	// Otherwise try to get the typed value
	switch fv.valueType {
	case TypeString:
		val, err := fv.String()
		return val, err == nil
	case TypeInt:
		val, err := fv.Int()
		return val, err == nil
	case TypeBool:
		val, err := fv.Bool()
		return val, err == nil
	case TypeFloat:
		val, err := fv.Float()
		return val, err == nil
	}
	return nil, false
}
