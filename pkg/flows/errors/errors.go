package errors

import "fmt"

// FlowError is the base error type for all flow-related errors
type FlowError struct {
	Code    string
	Message string
	Field   string
	Cause   error
}

func (fe *FlowError) Error() string {
	if fe.Field != "" {
		return fmt.Sprintf("%s: %s (field: %s)", fe.Code, fe.Message, fe.Field)
	}
	return fmt.Sprintf("%s: %s", fe.Code, fe.Message)
}

func (fe *FlowError) Unwrap() error {
	return fe.Cause
}

// Error code constants
const (
	ErrCodeUnknownField   = "ERR_UNKNOWN_FIELD"
	ErrCodeTypeMismatch   = "ERR_TYPE_MISMATCH"
	ErrCodeCircularDep    = "ERR_CIRCULAR_DEP"
	ErrCodeExpression     = "ERR_EXPRESSION"
	ErrCodeImmutableField = "ERR_IMMUTABLE_FIELD"
	ErrCodeValidation     = "ERR_VALIDATION"
)

// UnknownFieldError indicates a field reference that doesn't exist in the flow
type UnknownFieldError struct {
	FlowError
	Scope string // "input", "context", "output", "computed"
}

// TypeError indicates a type mismatch in value assignment or expression
type TypeError struct {
	FlowError
	ExpectedType string
	ActualType   string
}

// CircularDependencyError indicates computed fields depend on each other
type CircularDependencyError struct {
	FlowError
	Cycle []string
}

// ExpressionError indicates an invalid expression syntax
type ExpressionError struct {
	FlowError
	Expression string
	Position   int
}

// ImmutableFieldError indicates an attempt to modify an immutable field
type ImmutableFieldError struct {
	FlowError
	AttemptedOperation string
}

// ValidationError aggregates multiple validation errors
type ValidationError struct {
	FlowError
	Errors []error
}

func (ve *ValidationError) Error() string {
	return fmt.Sprintf("%s: %d validation errors", ve.Code, len(ve.Errors))
}
