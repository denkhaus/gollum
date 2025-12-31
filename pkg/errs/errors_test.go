package errs

import (
	"errors"
	"testing"
)

func TestError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *Error
		wantMsg string
	}{
		{
			name:    "error without cause",
			err:     New(TypeValidation, "invalid input"),
			wantMsg: "[validation] invalid input",
		},
		{
			name:    "error with cause",
			err:     Wrap(errors.New("underlying error"), TypeInternal, "operation failed"),
			wantMsg: "[internal] operation failed: underlying error",
		},
		{
			name:    "error with context",
			err:     New(TypeNotFound, "agent not found").WithContext("agent_id", "123"),
			wantMsg: "[not_found] agent not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMsg := tt.err.Error()
			if gotMsg != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", gotMsg, tt.wantMsg)
			}
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := Wrap(underlying, TypeInternal, "wrapper")

	if got := err.Unwrap(); got != underlying {
		t.Errorf("Unwrap() = %v, want %v", got, underlying)
	}

	// Error without cause should return nil
	noCause := New(TypeValidation, "test")
	if got := noCause.Unwrap(); got != nil {
		t.Errorf("Unwrap() on error without cause = %v, want nil", got)
	}
}

func TestNew(t *testing.T) {
	err := New(TypeValidation, "test message")

	if err.Type != TypeValidation {
		t.Errorf("New() Type = %v, want %v", err.Type, TypeValidation)
	}

	if err.Message != "test message" {
		t.Errorf("New() Message = %q, want %q", err.Message, "test message")
	}

	if err.Cause != nil {
		t.Errorf("New() Cause = %v, want nil", err.Cause)
	}
}

func TestNewf(t *testing.T) {
	err := Newf(TypeNotFound, "agent %s not found", "123")

	if err.Type != TypeNotFound {
		t.Errorf("Newf() Type = %v, want %v", err.Type, TypeNotFound)
	}

	expectedMsg := "agent 123 not found"
	if err.Message != expectedMsg {
		t.Errorf("Newf() Message = %q, want %q", err.Message, expectedMsg)
	}
}

func TestWrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := Wrap(underlying, TypeInternal, "wrapped error")

	if err.Type != TypeInternal {
		t.Errorf("Wrap() Type = %v, want %v", err.Type, TypeInternal)
	}

	if err.Message != "wrapped error" {
		t.Errorf("Wrap() Message = %q, want %q", err.Message, "wrapped error")
	}

	if err.Cause != underlying {
		t.Errorf("Wrap() Cause = %v, want %v", err.Cause, underlying)
	}

	// Wrapping nil should return nil
	if Wrap(nil, TypeInternal, "test") != nil {
		t.Error("Wrap(nil, ...) should return nil")
	}
}

func TestWrapf(t *testing.T) {
	underlying := errors.New("underlying error")
	err := Wrapf(underlying, TypeValidation, "invalid %s: %v", "input", "test")

	expectedMsg := "invalid input: test"
	if err.Message != expectedMsg {
		t.Errorf("Wrapf() Message = %q, want %q", err.Message, expectedMsg)
	}
}

func TestError_WithContext(t *testing.T) {
	err := New(TypeValidation, "test").
		WithContext("field", "email").
		WithContext("value", "invalid")

	if len(err.Context) != 2 {
		t.Errorf("WithContext() context length = %d, want 2", len(err.Context))
	}

	if err.Context["field"] != "email" {
		t.Errorf("WithContext() field = %q, want %q", err.Context["field"], "email")
	}

	if err.Context["value"] != "invalid" {
		t.Errorf("WithContext() value = %q, want %q", err.Context["value"], "invalid")
	}
}

func TestError_WithContextMap(t *testing.T) {
	ctx := map[string]any{
		"agent_id": "123",
		"attempt":  3,
	}

	err := New(TypePermission, "access denied").WithContextMap(ctx)

	if len(err.Context) != 2 {
		t.Errorf("WithContextMap() context length = %d, want 2", len(err.Context))
	}

	if err.Context["agent_id"] != "123" {
		t.Errorf("WithContextMap() agent_id = %q, want %q", err.Context["agent_id"], "123")
	}
}

func TestType_String(t *testing.T) {
	tests := []struct {
		t        Type
		expected string
	}{
		{TypeValidation, "validation"},
		{TypeNotFound, "not_found"},
		{TypePermission, "permission"},
		{TypeInternal, "internal"},
		{TypeConflict, "conflict"},
		{TypeTimeout, "timeout"},
		{TypeUnknown, "unknown"},
		{Type(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.t.String(); got != tt.expected {
				t.Errorf("Type.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestConvenienceFunctions(t *testing.T) {
	tests := []struct {
		name       string
		createFn   func(string) *Error
		wantType   Type
		wantPrefix string
	}{
		{
			name:       "Validation",
			createFn:   Validation,
			wantType:   TypeValidation,
			wantPrefix: "invalid",
		},
		{
			name:       "NotFound",
			createFn:   NotFound,
			wantType:   TypeNotFound,
			wantPrefix: "not found",
		},
		{
			name:       "Permission",
			createFn:   Permission,
			wantType:   TypePermission,
			wantPrefix: "denied",
		},
		{
			name:       "Internal",
			createFn:   Internal,
			wantType:   TypeInternal,
			wantPrefix: "failed",
		},
		{
			name:       "Conflict",
			createFn:   Conflict,
			wantType:   TypeConflict,
			wantPrefix: "conflict",
		},
		{
			name:       "Timeout",
			createFn:   Timeout,
			wantType:   TypeTimeout,
			wantPrefix: "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.createFn(tt.wantPrefix + " error")
			if err.Type != tt.wantType {
				t.Errorf("%s() Type = %v, want %v", tt.name, err.Type, tt.wantType)
			}
		})
	}
}

func TestValidationf(t *testing.T) {
	err := Validationf("invalid %s: %v", "email", "test@test")
	expected := "invalid email: test@test"
	if err.Message != expected {
		t.Errorf("Validationf() = %q, want %q", err.Message, expected)
	}
}

func TestAsError(t *testing.T) {
	// Test with our Error type
	myErr := New(TypeValidation, "test")
	if AsError(myErr) != myErr {
		t.Error("AsError() should return the same error for *Error type")
	}

	// Test with standard error
	stdErr := errors.New("standard error")
	if AsError(stdErr) != nil {
		t.Error("AsError() should return nil for non-*Error type")
	}

	// Test with nil
	if AsError(nil) != nil {
		t.Error("AsError(nil) should return nil")
	}

	// Test with wrapped error
	wrapped := Wrap(myErr, TypeInternal, "wrapped")
	if AsError(wrapped) == nil {
		t.Error("AsError() should be able to unwrap and find *Error")
	}
}

func TestError_Getters(t *testing.T) {
	underlying := errors.New("underlying")
	ctx := map[string]any{"key": "value"}

	err := Wrap(underlying, TypeValidation, "test").WithContextMap(ctx)

	if err.GetType() != TypeValidation {
		t.Errorf("GetType() = %v, want %v", err.GetType(), TypeValidation)
	}

	if err.GetCause() != underlying {
		t.Errorf("GetCause() = %v, want %v", err.GetCause(), underlying)
	}

	gotCtx := err.GetContext()
	if gotCtx["key"] != "value" {
		t.Errorf("GetContext() = %v, want %v", gotCtx, ctx)
	}
}

func TestIsType(t *testing.T) {
	validationErr := New(TypeValidation, "test")

	if !IsType(validationErr, TypeValidation) {
		t.Error("IsType() should return true for matching type")
	}

	if IsType(validationErr, TypeNotFound) {
		t.Error("IsType() should return false for non-matching type")
	}
}

func TestIsType_Wrapping(t *testing.T) {
	validationErr := New(TypeValidation, "test")
	wrapped := Wrap(validationErr, TypeInternal, "wrapped")

	if !IsType(wrapped, TypeInternal) {
		t.Error("IsType() should detect the outermost error type")
	}
}
