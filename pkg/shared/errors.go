package shared

import (
	"github.com/denkhaus/gollum/pkg/errs"
)

// WrapError wraps an error with type and context using the errs package
func WrapError(err error, errorType errs.Type, message string) *errs.Error {
	if err == nil {
		return nil
	}
	return errs.Wrap(err, errorType, message)
}

// WrapErrorf wraps an error with type and formatted message
func WrapErrorf(err error, errorType errs.Type, format string, args ...interface{}) *errs.Error {
	if err == nil {
		return nil
	}
	return errs.Wrapf(err, errorType, format, args...)
}

// NewError creates a new error with the given type and message
func NewError(errorType errs.Type, message string) *errs.Error {
	return errs.New(errorType, message)
}

// NewErrorf creates a new error with formatted message
func NewErrorf(errorType errs.Type, format string, args ...interface{}) *errs.Error {
	return errs.Newf(errorType, format, args...)
}

// Convenience functions for common error types

// WrapValidation wraps an error as a validation error
func WrapValidation(err error, message string) *errs.Error {
	return WrapError(err, errs.TypeValidation, message)
}

// WrapValidationf wraps an error as a validation error with formatted message
func WrapValidationf(err error, format string, args ...interface{}) *errs.Error {
	return WrapErrorf(err, errs.TypeValidation, format, args...)
}

// WrapNotFound wraps an error as a "not found" error
func WrapNotFound(err error, message string) *errs.Error {
	return WrapError(err, errs.TypeNotFound, message)
}

// WrapInternal wraps an error as an internal error
func WrapInternal(err error, message string) *errs.Error {
	return WrapError(err, errs.TypeInternal, message)
}

// WrapTimeout wraps an error as a timeout error
func WrapTimeout(err error, message string) *errs.Error {
	return WrapError(err, errs.TypeTimeout, message)
}
