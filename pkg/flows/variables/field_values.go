package variables

import (
	"fmt"
	"strconv"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/errors"
)

// FieldValues is a generic type-safe container for field values.
// T must be a type that implements FieldDefinition (e.g., flows.FieldDef, flows.ContextField).
type FieldValues[T FieldDefinition] struct {
	fields map[string]FieldValue
	defs   map[string]T
}

// NewFieldValues creates a new FieldValues from a slice of field definitions.
// It initializes all fields with their default values (if provided) or unset placeholders.
func NewFieldValues[T FieldDefinition](defs []T) *FieldValues[T] {
	fv := &FieldValues[T]{
		fields: make(map[string]FieldValue),
		defs:   make(map[string]T),
	}

	for _, def := range defs {
		name := def.GetName()
		fv.defs[name] = def

		// Initialize field based on default value and type
		fv.initializeField(name, def)
	}

	return fv
}

// initializeField initializes a single field with its default value or unset placeholder
func (fv *FieldValues[T]) initializeField(name string, def T) {
	fieldType := def.GetType()
	defaultValue := def.GetDefault()

	// If default value is provided, try to parse and set it
	if defaultValue != "" {
		switch fieldType {
		case flows.TypeString:
			fv.fields[name] = NewStringValue(defaultValue)
		case flows.TypeInt:
			if i, err := strconv.Atoi(defaultValue); err == nil {
				fv.fields[name] = NewIntValue(i)
			} else {
				fv.fields[name] = NewUnsetIntValue()
			}
		case flows.TypeBool:
			if b, err := strconv.ParseBool(defaultValue); err == nil {
				fv.fields[name] = NewBoolValue(b)
			} else {
				fv.fields[name] = NewUnsetBoolValue()
			}
		case flows.TypeFloat:
			if f, err := strconv.ParseFloat(defaultValue, 64); err == nil {
				fv.fields[name] = NewFloatValue(f)
			} else {
				fv.fields[name] = NewUnsetFloatValue()
			}
		}
		return
	}

	// No default value, initialize with unset placeholder
	switch fieldType {
	case flows.TypeString:
		fv.fields[name] = NewUnsetStringValue()
	case flows.TypeInt:
		fv.fields[name] = NewUnsetIntValue()
	case flows.TypeBool:
		fv.fields[name] = NewUnsetBoolValue()
	case flows.TypeFloat:
		fv.fields[name] = NewUnsetFloatValue()
	}
}

// Has returns true if the field is defined (exists in the schema)
func (fv *FieldValues[T]) Has(name string) bool {
	_, ok := fv.defs[name]
	return ok
}

// validateField checks if a field exists and matches the expected type
func (fv *FieldValues[T]) validateField(name string, expectedType flows.ValueType) error {
	if _, ok := fv.defs[name]; !ok {
		return &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "field not defined",
				Field:   name,
			},
			Scope: "values",
		}
	}
	if string(fv.defs[name].GetType()) != string(expectedType) {
		return &errors.TypeError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeTypeMismatch,
				Message: fmt.Sprintf("field '%s' is not of type %s", name, expectedType),
				Field:   name,
			},
			ExpectedType: expectedType,
			ActualType:   fv.defs[name].GetType(),
		}
	}
	return nil
}

// SetString sets a string field value
func (fv *FieldValues[T]) SetString(name string, value string) error {
	if err := fv.validateField(name, flows.TypeString); err != nil {
		return err
	}
	fv.fields[name] = NewStringValue(value)
	return nil
}

// GetString gets a string field value
func (fv *FieldValues[T]) GetString(name string) (string, error) {
	fieldValue, ok := fv.fields[name]
	if !ok {
		return "", &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "field not defined",
				Field:   name,
			},
		}
	}
	if !fieldValue.IsSet() {
		return "", &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "field not set",
				Field:   name,
			},
		}
	}
	return fieldValue.String()
}

// SetInt sets an int field value
func (fv *FieldValues[T]) SetInt(name string, value int) error {
	if err := fv.validateField(name, flows.TypeInt); err != nil {
		return err
	}
	fv.fields[name] = NewIntValue(value)
	return nil
}

// GetInt gets an int field value
func (fv *FieldValues[T]) GetInt(name string) (int, error) {
	fieldValue, ok := fv.fields[name]
	if !ok {
		return 0, &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "field not defined",
				Field:   name,
			},
		}
	}
	if !fieldValue.IsSet() {
		return 0, &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "field not set",
				Field:   name,
			},
		}
	}
	return fieldValue.Int()
}

// SetBool sets a bool field value
func (fv *FieldValues[T]) SetBool(name string, value bool) error {
	if err := fv.validateField(name, flows.TypeBool); err != nil {
		return err
	}
	fv.fields[name] = NewBoolValue(value)
	return nil
}

// GetBool gets a bool field value
func (fv *FieldValues[T]) GetBool(name string) (bool, error) {
	fieldValue, ok := fv.fields[name]
	if !ok {
		return false, &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "field not defined",
				Field:   name,
			},
		}
	}
	if !fieldValue.IsSet() {
		return false, &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "field not set",
				Field:   name,
			},
		}
	}
	return fieldValue.Bool()
}

// SetFloat sets a float field value
func (fv *FieldValues[T]) SetFloat(name string, value float64) error {
	if err := fv.validateField(name, flows.TypeFloat); err != nil {
		return err
	}
	fv.fields[name] = NewFloatValue(value)
	return nil
}

// GetFloat gets a float field value
func (fv *FieldValues[T]) GetFloat(name string) (float64, error) {
	fieldValue, ok := fv.fields[name]
	if !ok {
		return 0, &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "field not defined",
				Field:   name,
			},
		}
	}
	if !fieldValue.IsSet() {
		return 0, &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "field not set",
				Field:   name,
			},
		}
	}
	return fieldValue.Float()
}

// SetFromString sets a field value from a string, automatically converting to the target type.
// Returns error if the field is not defined or if type conversion fails.
func (fv *FieldValues[T]) SetFromString(name string, value string) error {
	fieldDef, ok := fv.defs[name]
	if !ok {
		return &errors.UnknownFieldError{
			FlowError: errors.FlowError{
				Code:    errors.ErrCodeUnknownField,
				Message: "field not defined",
				Field:   name,
			},
		}
	}

	// Convert string value to the appropriate type based on field definition
	switch fieldDef.GetType() {
	case flows.TypeString:
		return fv.SetString(name, value)
	case flows.TypeInt:
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
		return fv.SetInt(name, i)
	case flows.TypeBool:
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
		return fv.SetBool(name, b)
	case flows.TypeFloat:
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
		return fv.SetFloat(name, f)
	default:
		return fmt.Errorf("unknown field type '%s' for field '%s'", fieldDef.GetType(), name)
	}
}

// GetRaw gets a value without type checking.
// Returns the value and true if the field is set, (nil, false) otherwise.
func (fv *FieldValues[T]) GetRaw(name string) (any, bool) {
	fieldValue, ok := fv.fields[name]
	if !ok || !fieldValue.IsSet() {
		return nil, false
	}

	// Try to get the typed value
	switch fieldValue.Type() {
	case flows.TypeString:
		val, err := fieldValue.String()
		return val, err == nil
	case flows.TypeInt:
		val, err := fieldValue.Int()
		return val, err == nil
	case flows.TypeBool:
		val, err := fieldValue.Bool()
		return val, err == nil
	case flows.TypeFloat:
		val, err := fieldValue.Float()
		return val, err == nil
	}
	return nil, false
}
