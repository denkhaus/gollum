package shared

import (
	"fmt"
	"reflect"
	"strconv"
)

// SetFieldValue sets a reflect.Value with type conversion
func SetFieldValue(field reflect.Value, value any) error {
	if !field.CanSet() {
		return fmt.Errorf("field is not settable")
	}

	val := reflect.ValueOf(value)
	if val.Type().ConvertibleTo(field.Type()) {
		field.Set(val.Convert(field.Type()))
		return nil
	}

	// Try specific conversions
	switch field.Kind() {
	case reflect.String:
		s, err := ConvertToString(value)
		if err != nil {
			return err
		}
		field.SetString(s)
	case reflect.Int, reflect.Int64:
		i, err := ConvertToInt(value)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(i).Convert(field.Type()))
	case reflect.Bool:
		b, err := ConvertToBool(value)
		if err != nil {
			return err
		}
		field.SetBool(b)
	case reflect.Float64, reflect.Float32:
		f, err := ConvertToFloat(value)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(f).Convert(field.Type()))
	default:
		return fmt.Errorf("unsupported type: %s", field.Kind())
	}

	return nil
}

// ConvertToString converts any value to string
func ConvertToString(v any) (string, error) {
	if v == nil {
		return "", nil
	}
	switch val := v.(type) {
	case string:
		return val, nil
	case int:
		return strconv.Itoa(val), nil
	case int64:
		return strconv.FormatInt(val, 10), nil
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(val), nil
	default:
		return fmt.Sprintf("%v", val), nil
	}
}

// ConvertToInt converts any value to int
func ConvertToInt(v any) (int, error) {
	if v == nil {
		return 0, fmt.Errorf("cannot convert nil to int")
	}
	switch val := v.(type) {
	case int:
		return val, nil
	case int64:
		return int(val), nil
	case float64:
		return int(val), nil
	case string:
		i, err := strconv.Atoi(val)
		if err != nil {
			return 0, fmt.Errorf("cannot convert string '%s' to int: %w", val, err)
		}
		return i, nil
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int", v)
	}
}

// ConvertToBool converts any value to bool
func ConvertToBool(v any) (bool, error) {
	if v == nil {
		return false, nil
	}
	switch val := v.(type) {
	case bool:
		return val, nil
	case string:
		return strconv.ParseBool(val)
	case int:
		return val != 0, nil
	case float64:
		return val != 0, nil
	default:
		return false, fmt.Errorf("cannot convert %T to bool", v)
	}
}

// ConvertToFloat converts any value to float64
func ConvertToFloat(v any) (float64, error) {
	if v == nil {
		return 0, nil
	}
	switch val := v.(type) {
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot convert string '%s' to float: %w", val, err)
		}
		return f, nil
	case bool:
		if val {
			return 1.0, nil
		}
		return 0.0, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float", v)
	}
}
