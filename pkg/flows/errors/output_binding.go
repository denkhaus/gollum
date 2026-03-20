package errors

import "fmt"

// OutputFieldReadOnlyError is returned when attempting to set a declarative output field at runtime
type OutputFieldReadOnlyError struct {
	FlowError
	FieldName string
	Source    string // the 'from' source reference
}

func (e *OutputFieldReadOnlyError) Error() string {
	return e.FlowError.Error()
}

// NewOutputFieldReadOnlyError creates a new OutputFieldReadOnlyError
func NewOutputFieldReadOnlyError(fieldName, source string) *OutputFieldReadOnlyError {
	return &OutputFieldReadOnlyError{
		FlowError: FlowError{
			Code:    ErrCodeOutputFieldReadOnly,
			Message: fmt.Sprintf("output field '%s' is readonly (has 'from' attribute, value flows from: %s)", fieldName, source),
			Field:   fieldName,
		},
		FieldName: fieldName,
		Source:    source,
	}
}

// OutputBindingError is returned when output binding initialization fails
type OutputBindingError struct {
	FlowError
	OutputName string
	Source     string // the source that failed
	Cause      error  // underlying error
}

func (e *OutputBindingError) Error() string {
	return e.FlowError.Error()
}

func (e *OutputBindingError) Unwrap() error {
	return e.Cause
}

// NewOutputBindingError creates a new OutputBindingError
func NewOutputBindingError(outputName, source string, cause error) *OutputBindingError {
	return &OutputBindingError{
		FlowError: FlowError{
			Code:    ErrCodeOutputBinding,
			Message: fmt.Sprintf("failed to bind output '%s' from source '%s': %v", outputName, source, cause),
			Field:   outputName,
			Cause:   cause,
		},
		OutputName: outputName,
		Source:     source,
		Cause:      cause,
	}
}
