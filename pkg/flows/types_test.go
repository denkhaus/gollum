package flows

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlowStruct_BasicFields(t *testing.T) {
	flow := &Flow{
		Name:    "test-flow",
		Version: "1.0",
	}

	assert.Equal(t, "test-flow", flow.Name)
	assert.Equal(t, "1.0", flow.Version)
}

func TestInputBlock_HasRequiredAndType(t *testing.T) {
	input := &InputBlock{
		Ints: []FieldDef{
			{Name: "pr_number", Required: true},
		},
		Strings: []FieldDef{
			{Name: "repo_owner", Default: "denkhaus"},
		},
	}

	fields := input.GetAllFields()
	assert.Len(t, fields, 2)

	// Strings come first, then Ints (per implementation order)
	assert.Equal(t, "repo_owner", fields[0].Name)
	assert.Equal(t, "denkhaus", fields[0].Default)

	assert.Equal(t, "pr_number", fields[1].Name)
	assert.True(t, fields[1].Required)
}

func TestComputedBlock_HasComputedFields(t *testing.T) {
	computed := &ComputedBlock{
		Bools: []ComputedFieldDef{
			{Name: "is_open", Type: "bool", Eval: "EQ(context.status, 'open')"},
			{Name: "is_large", Type: "bool", Eval: "GT(context.count, 10)"},
		},
	}

	assert.Len(t, computed.GetAllFields(), 2)
	assert.Equal(t, "is_open", computed.GetAllFields()[0].Name)
	assert.Equal(t, TypeBool, computed.GetAllFields()[0].Type)
	assert.Equal(t, "EQ(context.status, 'open')", computed.GetAllFields()[0].Eval)
}

func TestFlow_HasComputedBlock(t *testing.T) {
	flow := &Flow{
		Name: "test-flow",
		Computed: &ComputedBlock{
			Strings: []ComputedFieldDef{
				{Name: "result", Type: "string", Eval: "CONCAT(input.prefix, input.suffix)"},
			},
		},
	}

	assert.NotNil(t, flow.Computed)
	assert.Len(t, flow.Computed.GetAllFields(), 1)
	assert.Equal(t, "result", flow.Computed.GetAllFields()[0].Name)
}
