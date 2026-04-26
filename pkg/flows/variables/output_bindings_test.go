package variables

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
)

func TestOutputBinding(t *testing.T) {
	binding := OutputBinding{
		TargetName:  "result",
		SourceScope: "computed",
		SourceName:  "sum",
		ValueType:   flows.TypeInt,
	}

	assert.Equal(t, "result", binding.TargetName)
	assert.Equal(t, flows.FlowVariableScopeComputed, binding.SourceScope)
	assert.Equal(t, "sum", binding.SourceName)
	assert.Equal(t, flows.TypeInt, binding.ValueType)
}

func TestParseFieldReference(t *testing.T) {
	scope, name, err := ParseFieldReference("computed.sum")
	assert.NoError(t, err)
	assert.Equal(t, flows.FlowVariableScopeComputed, scope)
	assert.Equal(t, "sum", name)

	_, _, err = ParseFieldReference("invalid")
	assert.Error(t, err)

	_, _, err = ParseFieldReference(".missing")
	assert.Error(t, err)

	_, _, err = ParseFieldReference("missing.")
	assert.Error(t, err)
}

func TestFlowVariableScope_Validate(t *testing.T) {
	assert.NoError(t, flows.FlowVariableScopeInput.Validate())
	assert.NoError(t, flows.FlowVariableScopeContext.Validate())
	assert.NoError(t, flows.FlowVariableScopeComputed.Validate())
	assert.NoError(t, flows.FlowVariableScopeOutput.Validate())
	assert.NoError(t, flows.FlowVariableScopeSys.Validate())
	assert.Error(t, flows.FlowVariableScope("invalid").Validate())
	assert.Error(t, flows.FlowVariableScope("").Validate())
}
