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
		Fields: []ComputedFieldDef{
			{Name: "is_open", Type: "bool", Eval: "EQ(context.status, 'open')"},
			{Name: "is_large", Type: "bool", Eval: "GT(context.count, 10)"},
		},
	}

	assert.Len(t, computed.Fields, 2)
	assert.Equal(t, "is_open", computed.Fields[0].Name)
	assert.Equal(t, "bool", computed.Fields[0].Type)
	assert.Equal(t, "EQ(context.status, 'open')", computed.Fields[0].Eval)
}

func TestFlow_HasComputedBlock(t *testing.T) {
	flow := &Flow{
		Name: "test-flow",
		Computed: &ComputedBlock{
			Fields: []ComputedFieldDef{
				{Name: "result", Type: "string", Eval: "CONCAT(input.prefix, input.suffix)"},
			},
		},
	}

	assert.NotNil(t, flow.Computed)
	assert.Len(t, flow.Computed.Fields, 1)
	assert.Equal(t, "result", flow.Computed.Fields[0].Name)
}

func TestFlow_BackwardCompatibility_ContextComputeds(t *testing.T) {
	// Ensure backward compatibility with Context.Computeds
	flow := &Flow{
		Name: "test-flow",
		Context: &ContextBlock{
			Computeds: []ComputedField{
				{Name: "old_style", Type: "bool", When: "true"},
			},
		},
	}

	assert.NotNil(t, flow.Context)
	assert.Len(t, flow.Context.Computeds, 1)
	assert.Equal(t, "old_style", flow.Context.Computeds[0].Name)
}
