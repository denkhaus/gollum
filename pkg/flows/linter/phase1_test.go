package linter

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
)

func TestPhase1_MissingInput(t *testing.T) {
	flow := &flows.Flow{
		Name:   "test",
		Output: &flows.OutputBlock{},
	}

	result := Lint(flow)

	assert.False(t, result.Valid)

	// Check for missing input error
	hasMissingInput := false
	for _, err := range result.Errors {
		if err.Code == flows.ErrMissingInput {
			hasMissingInput = true
			break
		}
	}
	assert.True(t, hasMissingInput, "should have missing input error")
}

func TestPhase1_MissingOutput(t *testing.T) {
	flow := &flows.Flow{
		Name:  "test",
		Input: &flows.InputBlock{},
	}

	result := Lint(flow)

	assert.False(t, result.Valid)

	// Check for missing output error
	hasMissingOutput := false
	for _, err := range result.Errors {
		if err.Code == flows.ErrMissingOutput {
			hasMissingOutput = true
			break
		}
	}
	assert.True(t, hasMissingOutput, "should have missing output error")
}

func TestPhase1_ValidMinimalFlow(t *testing.T) {
	flow := &flows.Flow{
		Name:   "minimal",
		Input:  &flows.InputBlock{},
		Output: &flows.OutputBlock{},
		States: []flows.State{
			{Name: "init", Initial: true},
			{Name: "done"},
		},
	}

	result := Lint(flow)

	// Should pass phase 1 (input/output present)
	// May have other phase errors
	assert.NotContains(t, result.Errors, flows.LinterError{Code: flows.ErrMissingInput})
	assert.NotContains(t, result.Errors, flows.LinterError{Code: flows.ErrMissingOutput})
}
