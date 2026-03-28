package shared

import "testing"

func TestIsIdentStart(t *testing.T) {
	tests := []struct {
		ch       byte
		expected bool
	}{
		{'a', true},
		{'Z', true},
		{'_', true},
		{'0', false},
		{'-', false},
		{' ', false},
	}
	for _, tt := range tests {
		if got := IsIdentStart(tt.ch); got != tt.expected {
			t.Errorf("IsIdentStart(%q) = %v, want %v", tt.ch, got, tt.expected)
		}
	}
}

func TestIsIdentPart(t *testing.T) {
	tests := []struct {
		ch       byte
		expected bool
	}{
		{'a', true},
		{'Z', true},
		{'_', true},
		{'0', true},
		{'9', true},
		{'-', false},
		{' ', false},
	}
	for _, tt := range tests {
		if got := IsIdentPart(tt.ch); got != tt.expected {
			t.Errorf("IsIdentPart(%q) = %v, want %v", tt.ch, got, tt.expected)
		}
	}
}

func TestIsDigit(t *testing.T) {
	tests := []struct {
		ch       byte
		expected bool
	}{
		{'0', true},
		{'5', true},
		{'9', true},
		{'a', false},
		{'-', false},
	}
	for _, tt := range tests {
		if got := IsDigit(tt.ch); got != tt.expected {
			t.Errorf("IsDigit(%q) = %v, want %v", tt.ch, got, tt.expected)
		}
	}
}
