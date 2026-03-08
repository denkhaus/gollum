package executor

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
)

func TestToolSetOutputField(t *testing.T) {
	exec := NewExecutor(&flows.Flow{})
	tool := &ToolSetOutputField{executor: exec}

	input := map[string]any{"name": "result", "value": "done"}
	result, err := tool.Execute(input)

	assert.NoError(t, err)
	assert.True(t, result["success"].(bool))

	val, ok := exec.ctx.GetOutputField("result")
	assert.True(t, ok)
	assert.Equal(t, "done", val)
}

func TestToolSetOutputField_ValidatesType(t *testing.T) {
	flow := &flows.Flow{
		Output: &flows.OutputBlock{
			Ints: []flows.FieldDef{{Name: "count"}},
		},
	}
	exec := NewExecutor(flow)
	tool := &ToolSetOutputField{executor: exec, flow: flow}

	// Should accept int value
	input := map[string]any{"name": "count", "value": 42}
	_, err := tool.Execute(input)
	assert.NoError(t, err)

	// Should reject string value for int field
	input = map[string]any{"name": "count", "value": "invalid"}
	_, err = tool.Execute(input)
	assert.Error(t, err)
}
