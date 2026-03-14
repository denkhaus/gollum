package executor

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
)

func TestNewContext_InitializesWithDefaults(t *testing.T) {
	input := &flows.InputBlock{
		Strings: []flows.FieldDef{{Name: "repo", Required: true}},
	}
	inputVals := map[string]string{"repo": "gollum"}

	ctx := NewContext(input, inputVals)

	assert.Equal(t, "gollum", ctx.GetInput("repo"))
	assert.NotNil(t, ctx.contextVals)
	assert.NotNil(t, ctx.computed)
}

func TestNewContext_AppliesInputDefaults(t *testing.T) {
	input := &flows.InputBlock{
		Strings: []flows.FieldDef{{Name: "owner", Default: "denkhaus"}},
	}

	ctx := NewContext(input, nil)

	assert.Equal(t, "denkhaus", ctx.GetInput("owner"))
}

func TestContext_EvaluateComputedFields(t *testing.T) {
	flow := &flows.Flow{
		Context: &flows.ContextBlock{
			Strings: []flows.ContextField{{Name: "status"}},
			Computeds: []flows.ComputedField{
				{Name: "is_open", Type: "bool", When: "EQ(context.status, 'open')"},
			},
		},
	}

	ctx := NewContext(&flows.InputBlock{}, nil)
	ctx.SetContextField("status", "open")

	err := ctx.EvaluateComputedFields(flow.Context)

	assert.NoError(t, err)
	val, err := ctx.GetContextField("is_open")
	assert.NoError(t, err)
	assert.Equal(t, true, val)
}

func TestContext_ComputedFieldsAreImmutable(t *testing.T) {
	flow := &flows.Flow{
		Context: &flows.ContextBlock{
			Strings: []flows.ContextField{{Name: "count"}},
			Computeds: []flows.ComputedField{
				{Name: "is_large", Type: "bool", When: "GT(context.count, 10)"},
			},
		},
	}

	ctx := NewContext(&flows.InputBlock{}, nil)
	ctx.SetContextField("count", "5")

	err := ctx.EvaluateComputedFields(flow.Context)
	assert.NoError(t, err)

	// Try to modify computed field
	ctx.SetContextField("is_large", "true")

	// Computed field should NOT be modified
	val, _ := ctx.GetContextField("is_large")
	assert.Equal(t, false, val)
}
