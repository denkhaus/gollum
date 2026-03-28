package shared

// IsWholeNumber checks if a float64 is a whole number
func IsWholeNumber(f float64) bool {
	return f == float64(int64(f))
}

// IsInt checks if a value is an integer type or a float representing a whole number
func IsInt(v any) bool {
	switch val := v.(type) {
	case int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8:
		return true
	case float64:
		return IsWholeNumber(val)
	case float32:
		return IsWholeNumber(float64(val))
	default:
		return false
	}
}

// IsActualIntType checks if the value's actual type is an integer (not a float that happens to be whole)
func IsActualIntType(v any) bool {
	switch v.(type) {
	case int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8:
		return true
	default:
		return false
	}
}
