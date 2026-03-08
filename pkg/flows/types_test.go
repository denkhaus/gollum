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
		Fields: []FieldDef{
			{Name: "pr_number", Type: "int", Required: true},
			{Name: "repo_owner", Type: "string", Default: "denkhaus"},
		},
	}

	assert.Len(t, input.Fields, 2)
	assert.True(t, input.Fields[0].Required)
	assert.Equal(t, "denkhaus", input.Fields[1].Default)
}
