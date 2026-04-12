package acp

import (
	"testing"
)

func TestTransportTypeValues(t *testing.T) {
	tests := []struct {
		name     string
		transport TransportType
		expected int
	}{
		{"stdio", TransportStdio, 0},
		{"http", TransportHTTP, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.transport) != tt.expected {
				t.Errorf("TransportType %s = %d, want %d", tt.name, tt.transport, tt.expected)
			}
		})
	}
}

func TestParseTransportType(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  TransportType
		wantError bool
	}{
		{"valid stdio", "stdio", TransportStdio, false},
		{"valid http", "http", TransportHTTP, false},
		{"invalid transport", "tcp", TransportStdio, true},
		{"empty string", "", TransportStdio, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseTransportType(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ParseTransportType(%q) error = %v, wantError %v", tt.input, err, tt.wantError)
				return
			}
			if !tt.wantError && result != tt.expected {
				t.Errorf("ParseTransportType(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTransportTypeString(t *testing.T) {
	tests := []struct {
		name     string
		transport TransportType
		expected string
	}{
		{"stdio", TransportStdio, "stdio"},
		{"http", TransportHTTP, "http"},
		{"unknown", TransportType(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := tt.transport.String(); result != tt.expected {
				t.Errorf("TransportType.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}
