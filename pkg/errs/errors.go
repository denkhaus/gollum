// Package errs provides structured error types with categorization and context support.
package errs

import (
	"errors"
	"fmt"
)

// Type represents the category of error
type Type int

const (
	// TypeUnknown is for uncategorized errors
	TypeUnknown Type = iota

	// TypeValidation indicates input validation failures
	TypeValidation

	// TypeNotFound indicates requested resources were not found
	TypeNotFound

	// TypePermission indicates authorization/permission failures
	TypePermission

	// TypeInternal indicates internal system errors
	TypeInternal

	// TypeConflict indicates conflicts with current state (e.g., concurrent modification)
	TypeConflict

	// TypeTimeout indicates operation timeout errors
	TypeTimeout
)

// String returns the string representation of the error type
func (t Type) String() string {
	switch t {
	case TypeValidation:
		return "validation"
	case TypeNotFound:
		return "not_found"
	case TypePermission:
		return "permission"
	case TypeInternal:
		return "internal"
	case TypeConflict:
		return "conflict"
	case TypeTimeout:
		return "timeout"
	default:
		return "unknown"
	}
}

// Error represents a structured error with type, message, and optional context
type Error struct {
	Type    Type           // Error category
	Message string         // Human-readable error message
	Cause   error          // Optional underlying error
	Context map[string]any // Optional additional context
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Cause == nil {
		return fmt.Sprintf("[%s] %s", e.Type, e.Message)
	}
	return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Cause)
}

// Unwrap returns the underlying cause error for errors.Unwrap()
func (e *Error) Unwrap() error {
	return e.Cause
}

// New creates a new Error with the given type and message
func New(errorType Type, message string) *Error {
	return &Error{
		Type:    errorType,
		Message: message,
	}
}

// Newf creates a new Error with formatted message
func Newf(errorType Type, format string, args ...any) *Error {
	return &Error{
		Type:    errorType,
		Message: fmt.Sprintf(format, args...),
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, errorType Type, message string) *Error {
	if err == nil {
		return nil
	}
	return &Error{
		Type:    errorType,
		Message: message,
		Cause:   err,
	}
}

// Wrapf wraps an error with formatted message
func Wrapf(err error, errorType Type, format string, args ...any) *Error {
	if err == nil {
		return nil
	}
	return &Error{
		Type:    errorType,
		Message: fmt.Sprintf(format, args...),
		Cause:   err,
	}
}

// WithContext adds context key-value pairs to the error
func (e *Error) WithContext(key string, value any) *Error {
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	e.Context[key] = value
	return e
}

// WithContextMap adds multiple context values to the error
func (e *Error) WithContextMap(ctx map[string]any) *Error {
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	for k, v := range ctx {
		e.Context[k] = v
	}
	return e
}

// GetType returns the error type
func (e *Error) GetType() Type {
	return e.Type
}

// GetCause returns the underlying cause error
func (e *Error) GetCause() error {
	return e.Cause
}

// GetContext returns the error context map
func (e *Error) GetContext() map[string]any {
	return e.Context
}

// IsType checks if an error is of a specific type
// It handles both *Error and wrapped errors
func IsType(err error, errorType Type) bool {
	var e *Error
	if err == nil {
		return false
	}

	// Check if error is our Error type
	if stdErr, ok := err.(*Error); ok {
		return stdErr.Type == errorType
	}

	// Try to unwrap and check
	return errors.As(err, &e) && e.Type == errorType
}

// AsError attempts to convert an error to our Error type
// Returns nil if the error is not of type *Error
func AsError(err error) *Error {
	if err == nil {
		return nil
	}

	if e, ok := err.(*Error); ok {
		return e
	}

	// Try unwrapping
	var e *Error
	if errors.As(err, &e) {
		return e
	}

	return nil
}

// Convenience functions for common error types

// Validation creates a validation error
func Validation(message string) *Error {
	return New(TypeValidation, message)
}

// Validationf creates a validation error with formatted message
func Validationf(format string, args ...any) *Error {
	return Newf(TypeValidation, format, args...)
}

// NotFound creates a "not found" error
func NotFound(message string) *Error {
	return New(TypeNotFound, message)
}

// NotFoundf creates a "not found" error with formatted message
func NotFoundf(format string, args ...any) *Error {
	return Newf(TypeNotFound, format, args...)
}

// Permission creates a permission error
func Permission(message string) *Error {
	return New(TypePermission, message)
}

// Permissionf creates a permission error with formatted message
func Permissionf(format string, args ...any) *Error {
	return Newf(TypePermission, format, args...)
}

// Internal creates an internal error
func Internal(message string) *Error {
	return New(TypeInternal, message)
}

// Internalf creates an internal error with formatted message
func Internalf(format string, args ...any) *Error {
	return Newf(TypeInternal, format, args...)
}

// Conflict creates a conflict error
func Conflict(message string) *Error {
	return New(TypeConflict, message)
}

// Conflictf creates a conflict error with formatted message
func Conflictf(format string, args ...any) *Error {
	return Newf(TypeConflict, format, args...)
}

// Timeout creates a timeout error
func Timeout(message string) *Error {
	return New(TypeTimeout, message)
}

// Timeoutf creates a timeout error with formatted message
func Timeoutf(format string, args ...any) *Error {
	return Newf(TypeTimeout, format, args...)
}
