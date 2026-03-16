package linter

import (
	"strings"

	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
)

// TestSyntaxChecker_ExpressionWithDollarSign tests that $() in expressions generates a warning
func TestSyntaxChecker_ExpressionWithDollarSign(t *testing.T) {
	flow := &flows.Flow{
		Name: "test-flow",
		States: []flows.State{
			{
				Name: "test-state",
				Calls: []flows.Call{
					{
						Ref:  "some-flow",
						When: "${computed.value}", // Wrong: should be computed.value
					},
				},
			},
		},
	}

	result := Lint(flow)

	// Should have a warning about ${} in expression
	hasDollarWarning := false
	for _, w := range result.Warnings {
		if w.Code == "W001" && strings.Contains(w.Message, "${}") {
			hasDollarWarning = true
			break
		}
	}
	assert.True(t, hasDollarWarning, "Should warn about ${} in expression context")
}

// TestSyntaxChecker_TemplateWithoutDollarSign tests that missing $() in templates generates a warning
func TestSyntaxChecker_TemplateWithoutDollarSign(t *testing.T) {
	flow := &flows.Flow{
		Name: "test-flow",
		States: []flows.State{
			{
				Name: "test-state",
				Steps: []flows.Step{
					{
						Type:   "llm",
						Prompt: "Analyze input.target and provide feedback", // Wrong: should be ${input.target}
					},
				},
			},
		},
	}

	result := Lint(flow)

	// Should have a warning about missing $() in template
	hasTemplateWarning := false
	for _, w := range result.Warnings {
		if w.Code == "W002" && strings.Contains(w.Message, "field references") {
			hasTemplateWarning = true
			break
		}
	}
	assert.True(t, hasTemplateWarning, "Should warn about missing $() in template context")
}

// TestSyntaxChecker_ValidExpression tests that correct expression syntax passes
func TestSyntaxChecker_ValidExpression(t *testing.T) {
	flow := &flows.Flow{
		Name: "test-flow",
		States: []flows.State{
			{
				Name: "test-state",
				Calls: []flows.Call{
					{
						Ref:  "some-flow",
						When: "EQ(computed.value, true)", // Correct
					},
				},
			},
		},
	}

	result := Lint(flow)

	// Should not have syntax warnings about ${}
	for _, w := range result.Warnings {
		if w.Code == "W001" && strings.Contains(w.Message, "${}") {
			t.Errorf("Should not warn about ${} in valid expression: %s", w.Message)
		}
	}
}

// TestSyntaxChecker_ValidTemplate tests that correct template syntax passes
func TestSyntaxChecker_ValidTemplate(t *testing.T) {
	flow := &flows.Flow{
		Name: "test-flow",
		States: []flows.State{
			{
				Name: "test-state",
				Steps: []flows.Step{
					{
						Type:   "llm",
						Prompt: "Analyze ${input.target} and provide feedback", // Correct
					},
				},
			},
		},
	}

	result := Lint(flow)

	// Should not have syntax warnings about templates
	for _, w := range result.Warnings {
		if w.Code == "W002" && strings.Contains(w.Message, "field references") {
			t.Errorf("Should not warn about $() in valid template: %s", w.Message)
		}
	}
}
