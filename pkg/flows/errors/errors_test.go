package errors_test

import (
	stderrors "errors"
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/errors"
	"github.com/stretchr/testify/assert"
)

func TestFlowError_Error(t *testing.T) {
	err := &errors.FlowError{
		Code:    "TEST_CODE",
		Message: "test message",
		Field:   "testField",
	}

	assert.Equal(t, "TEST_CODE: test message (field: testField)", err.Error())
}

func TestFlowError_Unwrap(t *testing.T) {
	cause := stderrors.New("underlying error")
	err := &errors.FlowError{
		Code:    "ERR_CODE",
		Message: "wrapper message",
		Cause:   cause,
	}

	assert.Equal(t, cause, stderrors.Unwrap(err))
}

func TestUnknownFieldError(t *testing.T) {
	err := &errors.UnknownFieldError{
		FlowError: errors.FlowError{
			Code:    errors.ErrCodeUnknownField,
			Message: "field not found",
			Field:   "missingField",
		},
		Scope: "context",
	}

	assert.Equal(t, "context", err.Scope)
	assert.Equal(t, errors.ErrCodeUnknownField, err.Code)
}

func TestTypeError(t *testing.T) {
	err := &errors.TypeError{
		FlowError: errors.FlowError{
			Code:    errors.ErrCodeTypeMismatch,
			Message: "type mismatch",
			Field:   "count",
		},
		ExpectedType: flows.TypeInt,
		ActualType:   flows.TypeString,
	}

	assert.Equal(t, flows.TypeInt, err.ExpectedType)
	assert.Equal(t, flows.TypeString, err.ActualType)
}

func TestCircularDependencyError(t *testing.T) {
	err := &errors.CircularDependencyError{
		FlowError: errors.FlowError{
			Code:    errors.ErrCodeCircularDep,
			Message: "circular dependency detected",
		},
		Cycle: []string{"a", "b", "a"},
	}

	assert.Equal(t, []string{"a", "b", "a"}, err.Cycle)
}

func TestErrorCodes(t *testing.T) {
	assert.Equal(t, "UNKNOWN_FIELD", errors.ErrCodeUnknownField)
	assert.Equal(t, "TYPE_MISMATCH", errors.ErrCodeTypeMismatch)
	assert.Equal(t, "CIRCULAR_DEPENDENCY", errors.ErrCodeCircularDep)
	assert.Equal(t, "EXPRESSION_ERROR", errors.ErrCodeExpression)
	assert.Equal(t, "IMMUTABLE_FIELD", errors.ErrCodeImmutable)
	assert.Equal(t, "VALIDATION_FAILED", errors.ErrCodeValidation)
}
