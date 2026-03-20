package shared

import "cmp"

// Contains checks if a string contains a substring
func Contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Clamp returns value constrained between min and max.
// Works with any type that satisfies cmp.Ordered (int, int8, int16, int32, int64,
// uint, uint8, uint16, uint32, uint64, uintptr, float32, float64).
func Clamp[T cmp.Ordered](value, minVal, maxVal T) T {
	if value < minVal {
		return minVal
	}
	if value > maxVal {
		return maxVal
	}
	return value
}
