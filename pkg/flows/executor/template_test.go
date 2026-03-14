package executor

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
)

func TestSubstituteTemplate_ReplacesVariables(t *testing.T) {
	ctx := NewContext(&flows.InputBlock{}, nil)
	ctx.inputVals = map[string]string{"pr_number": 123}
	ctx.SetContextField("pr_title", "Fix bug")

	result := SubstituteTemplate(ctx, "Analyze PR #${input.pr_number}: ${context.pr_title}")

	assert.Equal(t, "Analyze PR #123: Fix bug", result)
}

func TestSubstituteTemplate_HandlesMissingFields(t *testing.T) {
	ctx := NewContext(&flows.InputBlock{}, nil)

	result := SubstituteTemplate(ctx, "Value: ${input.missing}")

	assert.Equal(t, "Value: ${input.missing}", result)
}
