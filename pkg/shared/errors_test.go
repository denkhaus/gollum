package shared

import (
	"errors"
	"testing"

	"github.com/denkhaus/gollum/pkg/errs"
	"github.com/stretchr/testify/assert"
)

func TestWrapError_WithValidationType(t *testing.T) {
	baseErr := errors.New("invalid input")
	result := WrapError(baseErr, errs.TypeValidation, "user input failed")

	assert.NotNil(t, result)
	assert.True(t, errs.IsType(result, errs.TypeValidation))
	assert.Contains(t, result.Error(), "user input failed")
}

func TestWrapError_WithNotFoundType(t *testing.T) {
	baseErr := errors.New("not found")
	result := WrapError(baseErr, errs.TypeNotFound, "session not found")

	assert.True(t, errs.IsType(result, errs.TypeNotFound))
}

func TestWrapValidation(t *testing.T) {
	baseErr := errors.New("invalid email")
	result := WrapValidation(baseErr, "email validation failed")

	assert.True(t, errs.IsType(result, errs.TypeValidation))
}

func TestWrapNotFound(t *testing.T) {
	baseErr := errors.New("db: no rows")
	result := WrapNotFound(baseErr, "user not found")

	assert.True(t, errs.IsType(result, errs.TypeNotFound))
}

func TestWrapInternal(t *testing.T) {
	baseErr := errors.New("connection failed")
	result := WrapInternal(baseErr, "database connection failed")

	assert.True(t, errs.IsType(result, errs.TypeInternal))
}

func TestWrapError_NilError(t *testing.T) {
	result := WrapError(nil, errs.TypeValidation, "should not panic")
	assert.Nil(t, result)
}

func TestWrapErrorf_WithFormatting(t *testing.T) {
	baseErr := errors.New("base error")
	result := WrapErrorf(baseErr, errs.TypeValidation, "user %s failed validation", "john")

	assert.True(t, errs.IsType(result, errs.TypeValidation))
	assert.Contains(t, result.Error(), "user john failed validation")
}

func TestWrapValidationf(t *testing.T) {
	baseErr := errors.New("invalid")
	result := WrapValidationf(baseErr, "field %s is required", "email")

	assert.True(t, errs.IsType(result, errs.TypeValidation))
	assert.Contains(t, result.Error(), "field email is required")
}

func TestNewError(t *testing.T) {
	result := NewError(errs.TypeValidation, "direct error")

	assert.True(t, errs.IsType(result, errs.TypeValidation))
	assert.Contains(t, result.Error(), "direct error")
}

func TestNewErrorf(t *testing.T) {
	result := NewErrorf(errs.TypeNotFound, "resource %s not found", "user")

	assert.True(t, errs.IsType(result, errs.TypeNotFound))
	assert.Contains(t, result.Error(), "resource user not found")
}

func TestWrapTimeout(t *testing.T) {
	baseErr := errors.New("deadline exceeded")
	result := WrapTimeout(baseErr, "request timed out")

	assert.True(t, errs.IsType(result, errs.TypeTimeout))
}
