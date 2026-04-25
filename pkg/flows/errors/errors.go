package errors

import (
	"fmt"

	"github.com/denkhaus/gollum/pkg/flows"
)

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

// UnknownFieldError indicates a field reference that doesn't exist in the flow
type UnknownFieldError struct {
	FlowError
	Scope flows.FlowVariableScope // "input", "context", "output", "computed", "sys"
}

// TypeError indicates a type mismatch in value assignment or expression
type TypeError struct {
	FlowError
	ExpectedType flows.ValueType
	ActualType   flows.ValueType
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

// NoSchemaError indicates that no schema (context/output) was defined for the operation
type NoSchemaError struct {
	FlowError
	Scope flows.FlowVariableScope // "context", "output", "input"
}

// EnvVarNotFoundError indicates an environment variable referenced in flow was not found
type EnvVarNotFoundError struct {
	FlowError
	VarName string
}

// EnvVarEmptyError indicates an environment variable is set but empty
type EnvVarEmptyError struct {
	FlowError
	VarName string
}
