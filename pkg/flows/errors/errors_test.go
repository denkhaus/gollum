package errors_test

import (
	stderrors "errors"
	"testing"

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
