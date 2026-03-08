package extensions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateExtensionName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "my-extension", false},
		{"valid underscore", "test_ext", false},
		{"valid mixed", "Extension123", false},
		{"valid number start", "123extension", false},
		{"valid complex", "my_test-123", false},

		{"invalid empty", "", true},
		{"invalid path traversal", "../escape", true},
		{"invalid absolute path", "/absolute", true},
		{"invalid windows path", "\\windows", true},
		{"invalid space", "has space", true},
		{"invalid tab", "has\ttab", true},
		{"invalid special chars", "test@extension", true},
		{"invalid dot", "test.extension", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExtensionName(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateFuncName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "my-function", false},
		{"valid underscore", "test_func", false},
		{"valid camelCase", "MyFunction", false},
		{"valid pascal", "MyFunction123", false},

		{"invalid empty", "", true},
		{"invalid path traversal", "../escape", true},
		{"invalid space", "has space", true},
		{"invalid special chars", "test@func", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFuncName(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
