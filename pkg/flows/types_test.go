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
	assert.True(t, fields[0].Required)
	assert.Equal(t, "denkhaus", fields[1].Default)
}
