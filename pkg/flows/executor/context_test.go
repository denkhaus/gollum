package executor

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/errors"
	"github.com/stretchr/testify/assert"
)

func TestNewContext_InitializesWithDefaults(t *testing.T) {
	input := &flows.InputBlock{
		Strings: []flows.FieldDef{{Name: "repo", Required: true}},
	}
	inputVals := map[string]string{"repo": "gollum"}

	ctx := newContext(input, nil, nil, inputVals)

	assert.Equal(t, "gollum", ctx.GetInput("repo"))
	assert.NotNil(t, ctx.contextValues)
	assert.NotNil(t, ctx.computedVals)
}

func TestNewContext_AppliesInputDefaults(t *testing.T) {
	input := &flows.InputBlock{
		Strings: []flows.FieldDef{{Name: "owner", Default: "denkhaus"}},
	}

	ctx := NewContext(input, nil, nil, nil)

	assert.Equal(t, "denkhaus", ctx.GetInput("owner"))
}

func TestContext_EvaluateComputedFields(t *testing.T) {
	computedBlock := &flows.ComputedBlock{
		Bools: []flows.ComputedFieldDef{
			{Name: "is_open", Type: "bool", Eval: "EQ(context.status, 'open')"},
		},
	}
	contextBlock := &flows.ContextBlock{
		Strings: []flows.ContextField{{Name: "status"}},
	}

	ctx := newContext(&flows.InputBlock{}, nil, contextBlock, nil)
	ctx.SetComputedBlock(computedBlock)

	// Set the context field that the computed field depends on
	_ = ctx.SetContextField("status", "open")

	err := ctx.EvaluateComputed()

	assert.NoError(t, err)
	val, err := ctx.GetComputedField("is_open")
	assert.NoError(t, err)
	assert.Equal(t, true, val)
}

func TestContext_ComputedFieldsAreImmutable(t *testing.T) {
	computedBlock := &flows.ComputedBlock{
		Bools: []flows.ComputedFieldDef{
			{Name: "is_large", Type: "bool", Eval: "GT(context.count, 10)"},
		},
	}
	contextBlock := &flows.ContextBlock{
		Ints: []flows.ContextField{{Name: "count"}},
	}

	ctx := newContext(&flows.InputBlock{}, nil, contextBlock, nil)
	ctx.SetComputedBlock(computedBlock)
	_ = ctx.SetContextField("count", 5)

	err := ctx.EvaluateComputed()
	assert.NoError(t, err)

	// Computed fields are stored separately - context fields don't include them
	val, err := ctx.GetComputedField("is_large")
	assert.NoError(t, err)
	assert.Equal(t, false, val)

	// Context field is separate and mutable
	assert.NoError(t, ctx.SetContextField("count", 15))
}

func TestContextImpl_SetOutputField_DeclarativeReadonly(t *testing.T) {
	output := &flows.OutputBlock{
		Ints: []flows.FieldDef{
			{Name: "sum", From: "computed.sum", Type: flows.TypeInt},
			{Name: "count", Type: flows.TypeInt},
		},
	}

	ctx := newContext(nil, output, nil, nil)

	// Attempting to set a declarative field should fail
	err := ctx.SetOutputField("sum", 42)
	assert.Error(t, err)

	var readOnlyErr *errors.OutputFieldReadOnlyError
	assert.ErrorAs(t, err, &readOnlyErr)
	assert.Equal(t, "sum", readOnlyErr.FieldName)
	assert.Equal(t, "computed.sum", readOnlyErr.Source)

	// Setting an imperative field should succeed
	err = ctx.SetOutputField("count", 100)
	assert.NoError(t, err)
}
