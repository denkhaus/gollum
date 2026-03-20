package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOutputFieldReadOnlyError(t *testing.T) {
	err := NewOutputFieldReadOnlyError("sum", "computed.sum")

	assert.Contains(t, err.Error(), "sum")
	assert.Contains(t, err.Error(), "computed.sum")
	assert.Contains(t, err.Error(), "readonly")
	assert.Equal(t, "sum", err.FieldName)
	assert.Equal(t, "computed.sum", err.Source)
}

func TestOutputBindingError(t *testing.T) {
	cause := assert.AnError
	err := NewOutputBindingError("result", "computed.value", cause)

	assert.Contains(t, err.Error(), "result")
	assert.Contains(t, err.Error(), "computed.value")
	assert.Equal(t, "result", err.OutputName)
	assert.Equal(t, "computed.value", err.Source)
	assert.ErrorIs(t, err, cause)
}
