package executor

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
)

func TestSubstituteTemplate_ReplacesVariables(t *testing.T) {
	// Create a simple context with input fields
	input := &flows.InputBlock{
		Strings: []flows.FieldDef{{Name: "pr_number"}},
	}
	contextBlock := &flows.ContextBlock{
		Strings: []flows.ContextField{{Name: "pr_title"}},
	}
	ctx := NewContext(input, nil, contextBlock, map[string]string{"pr_number": "123"})
	// Set context field directly
	_ = ctx.SetContextField("pr_title", "Fix bug")

	result := SubstituteTemplate(ctx, "Analyze PR #${input.pr_number}: ${context.pr_title}")

	assert.Equal(t, "Analyze PR #123: Fix bug", result)
}

func TestSubstituteTemplate_HandlesMissingFields(t *testing.T) {
	ctx := NewContext(nil, nil, nil, nil)

	result := SubstituteTemplate(ctx, "Value: ${input.missing}")

	assert.Equal(t, "Value: ${input.missing}", result)
}
