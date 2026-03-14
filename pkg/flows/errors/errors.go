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
