package shared

import "testing"

func TestIsInt(t *testing.T) {
	tests := []struct {
		input    any
		expected bool
	}{
		{int(5), true},
		{int64(5), true},
		{int32(5), true},
		{uint(5), true},
		{float64(5.0), true},  // whole number
		{float64(5.5), false}, // not whole
		{float32(5.0), true},
		{"5", false},
		{nil, false},
	}
	for _, tt := range tests {
		if got := IsInt(tt.input); got != tt.expected {
			t.Errorf("IsInt(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestIsActualIntType(t *testing.T) {
	tests := []struct {
		input    any
		expected bool
	}{
		{int(5), true},
		{int64(5), true},
		{uint(5), true},
		{float64(5.0), false}, // float, even if whole
		{"5", false},
	}
	for _, tt := range tests {
		if got := IsActualIntType(tt.input); got != tt.expected {
			t.Errorf("IsActualIntType(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestIsWholeNumber(t *testing.T) {
	tests := []struct {
		input    float64
		expected bool
	}{
		{5.0, true},
		{5.5, false},
		{0.0, true},
		{-3.0, true},
		{-3.14, false},
	}
	for _, tt := range tests {
		if got := IsWholeNumber(tt.input); got != tt.expected {
			t.Errorf("IsWholeNumber(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}
