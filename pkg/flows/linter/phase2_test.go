package linter

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
)

func TestPhase2_InvalidExpression(t *testing.T) {
	flow := &flows.Flow{
		Name:   "test",
		Input:  &flows.InputBlock{},
		Output: &flows.OutputBlock{},
		Context: &flows.ContextBlock{
			Computeds: []flows.ComputedField{
				{Name: "is_open", When: "INVALID(context.pr.state,"},
			},
		},
	}

	result := Lint(flow)

	assert.False(t, result.Valid)
	// Should have an expression error
	hasExprError := false
	for _, err := range result.Errors {
		if err.Code == flows.ErrInvalidExpr {
			hasExprError = true
			break
		}
	}
	assert.True(t, hasExprError, "should have expression error")
}

func TestPhase2_FieldExists(t *testing.T) {
	flow := &flows.Flow{
		Name:   "test",
		Input:  &flows.InputBlock{},
		Output: &flows.OutputBlock{},
		Context: &flows.ContextBlock{
			Strings: []flows.ContextField{
				{Name: "status"},
			},
			Computeds: []flows.ComputedField{
				{Name: "is_open", When: "EQ(context.status, 'open')"},
			},
		},
	}

	result := Lint(flow)

	// Should validate successfully - field exists
	hasFieldNotFound := false
	for _, err := range result.Errors {
		if err.Code == flows.ErrFieldNotFound {
			hasFieldNotFound = true
			break
		}
	}
	assert.False(t, hasFieldNotFound, "should not have field not found error")
}
