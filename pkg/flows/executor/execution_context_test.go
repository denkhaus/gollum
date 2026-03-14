package executor

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
)

func TestNewExecutionContext_CreatesEmptyContext(t *testing.T) {
	input := &flows.InputBlock{
		Strings: []flows.FieldDef{{Name: "repo", Required: true}},
	}

	ctx := NewExecutionContext(input)

	assert.NotNil(t, ctx)
	assert.NotNil(t, ctx.inputVals)
	assert.NotNil(t, ctx.contextVals)
	assert.NotNil(t, ctx.outputVals)
}

func TestExecutionContext_SetInput_SetsValues(t *testing.T) {
	input := &flows.InputBlock{
		Strings: []flows.FieldDef{{Name: "repo", Required: true}},
	}
	ctx := NewExecutionContext(input)

	ctx.SetInput(map[string]string{"repo": "gollum"})

	val, err := ctx.GetInputField("repo")
	assert.NoError(t, err)
	assert.Equal(t, "gollum", val)
}

func TestExecutionContext_SetInput_AppliesDefaults(t *testing.T) {
	input := &flows.InputBlock{
		Strings: []flows.FieldDef{{Name: "owner", Default: "denkhaus"}},
	}
	ctx := NewExecutionContext(input)

	ctx.SetInput(map[string]string{})

	val, err := ctx.GetInputField("owner")
	assert.NoError(t, err)
	assert.Equal(t, "denkhaus", val)
}

func TestExecutionContext_SetContextField_SetsValue(t *testing.T) {
	ctx := NewExecutionContext(&flows.InputBlock{})

	err := ctx.SetContextField("status", "open")
	assert.NoError(t, err)

	val, err := ctx.GetContextField("status")
	assert.NoError(t, err)
	assert.Equal(t, "open", val)
}

func TestExecutionContext_SetOutputField_SetsValue(t *testing.T) {
	ctx := NewExecutionContext(&flows.InputBlock{})

	err := ctx.SetOutputField("result", "done")
	assert.NoError(t, err)

	val, err := ctx.GetOutputField("result")
	assert.NoError(t, err)
	assert.Equal(t, "done", val)
}

func TestExecutionContext_BuildInputScope_ReturnsNestedMap(t *testing.T) {
	ctx := NewExecutionContext(&flows.InputBlock{})
	ctx.SetInput(map[string]string{"key": "value"})

	scope := ctx.BuildInputScope()
	assert.NotNil(t, scope)
	assert.Equal(t, "value", scope["key"])
}

func TestExecutionContext_BuildContextScope_ReturnsNestedMap(t *testing.T) {
	ctx := NewExecutionContext(&flows.InputBlock{})
	ctx.SetContextField("status", "open")

	scope := ctx.BuildContextScope()
	assert.NotNil(t, scope)
	assert.Equal(t, "open", scope["status"])
}

func TestExecutionContext_BuildOutputScope_ReturnsNestedMap(t *testing.T) {
	ctx := NewExecutionContext(&flows.InputBlock{})
	ctx.SetOutputField("result", "done")

	scope := ctx.BuildOutputScope()
	assert.NotNil(t, scope)
	assert.Equal(t, "done", scope["result"])
}

func TestExecutionContext_BuildFullScope_ReturnsAllScopes(t *testing.T) {
	ctx := NewExecutionContext(&flows.InputBlock{})
	ctx.SetInput(map[string]string{"key": "value"})
	ctx.SetContextField("status", "open")
	ctx.SetOutputField("result", "done")

	scope := ctx.BuildFullScope()
	assert.NotNil(t, scope)

	inputScope, ok := scope["input"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "value", inputScope["key"])

	contextScope, ok := scope["context"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "open", contextScope["status"])

	outputScope, ok := scope["output"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "done", outputScope["result"])
}
