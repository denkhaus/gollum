package variables

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/denkhaus/gollum/pkg/flows"
)

func TestOutputBinding(t *testing.T) {
	binding := OutputBinding{
		TargetName:  "result",
		SourceScope: "computed",
		SourceName:  "sum",
		ValueType:   flows.TypeInt,
	}

	assert.Equal(t, "result", binding.TargetName)
	assert.Equal(t, "computed", binding.SourceScope)
	assert.Equal(t, "sum", binding.SourceName)
	assert.Equal(t, flows.TypeInt, binding.ValueType)
}

func TestParseFieldReference(t *testing.T) {
	scope, name, err := ParseFieldReference("computed.sum")
	assert.NoError(t, err)
	assert.Equal(t, "computed", scope)
	assert.Equal(t, "sum", name)

	_, _, err = ParseFieldReference("invalid")
	assert.Error(t, err)

	_, _, err = ParseFieldReference(".missing")
	assert.Error(t, err)

	_, _, err = ParseFieldReference("missing.")
	assert.Error(t, err)
}

func TestIsValidSourceScope(t *testing.T) {
	assert.True(t, IsValidSourceScope("input"))
	assert.True(t, IsValidSourceScope("context"))
	assert.True(t, IsValidSourceScope("computed"))
	assert.True(t, IsValidSourceScope("output"))
	assert.False(t, IsValidSourceScope("invalid"))
	assert.False(t, IsValidSourceScope(""))
}
