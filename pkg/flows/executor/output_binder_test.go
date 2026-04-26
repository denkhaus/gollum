package executor

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
)

func TestOutputBinder_InitializeBindings(t *testing.T) {
	flow := &flows.Flow{
		Input: &flows.InputBlock{
			Ints: []flows.FieldDef{{Name: "a", Type: flows.TypeInt}},
		},
		Computed: &flows.ComputedBlock{
			Ints: []flows.ComputedFieldDef{
				{Name: "doubled", Type: flows.TypeInt, Eval: "MUL(input.a, 2)"},
			},
		},
		Output: &flows.OutputBlock{
			Ints: []flows.FieldDef{
				{Name: "result", AssignFrom: "computed.doubled", Type: flows.TypeInt},
			},
		},
	}

	ctx := newContext(flow.Input, flow.Output, flow.Context, map[string]string{"a": "10"})
	ctx.SetComputedBlock(flow.Computed)

	// Evaluate computed fields
	err := ctx.EvaluateComputed()
	assert.NoError(t, err)

	// Initialize output bindings
	binder := NewOutputBinder()
	err = binder.InitializeBindings(ctx, flow.Output)
	assert.NoError(t, err)

	// Verify output has the value
	result, err := ctx.GetOutputField("result")
	assert.NoError(t, err)
	assert.Equal(t, 20, result)
}

func TestOutputBinder_BindFromInput(t *testing.T) {
	flow := &flows.Flow{
		Input: &flows.InputBlock{
			Strings: []flows.FieldDef{{Name: "message", Type: flows.TypeString}},
		},
		Output: &flows.OutputBlock{
			Strings: []flows.FieldDef{
				{Name: "result", AssignFrom: "input.message", Type: flows.TypeString},
			},
		},
	}

	ctx := newContext(flow.Input, flow.Output, flow.Context, map[string]string{"message": "Hello World"})

	binder := NewOutputBinder()
	err := binder.InitializeBindings(ctx, flow.Output)
	assert.NoError(t, err)

	result, err := ctx.GetOutputField("result")
	assert.NoError(t, err)
	assert.Equal(t, "Hello World", result)
}

func TestOutputBinder_InvalidReference(t *testing.T) {
	flow := &flows.Flow{
		Output: &flows.OutputBlock{
			Ints: []flows.FieldDef{
				{Name: "result", AssignFrom: "invalidformat", Type: flows.TypeInt},
			},
		},
	}

	ctx := newContext(flow.Input, flow.Output, flow.Context, nil)

	binder := NewOutputBinder()
	err := binder.InitializeBindings(ctx, flow.Output)
	assert.Error(t, err)
}
