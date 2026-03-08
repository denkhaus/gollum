package linter

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
)

func TestPhase3_UnreachableState(t *testing.T) {
	flow := &flows.Flow{
		Name:   "test",
		Input:  &flows.InputBlock{},
		Output: &flows.OutputBlock{},
		States: []flows.State{
			{Name: "init", Initial: true, Transitions: []flows.Transition{{To: "done"}}},
			{Name: "orphan"}, // unreachable
			{Name: "done"},
		},
	}

	result := Lint(flow)

	hasUnreachable := false
	for _, err := range result.Errors {
		if err.Code == flows.ErrUnreachableState {
			hasUnreachable = true
			break
		}
	}
	assert.True(t, hasUnreachable, "should detect unreachable state")
}

func TestPhase3_ValidTransitionTarget(t *testing.T) {
	flow := &flows.Flow{
		Name:   "test",
		Input:  &flows.InputBlock{},
		Output: &flows.OutputBlock{},
		States: []flows.State{
			{Name: "init", Initial: true, Transitions: []flows.Transition{{To: "nonexistent"}}},
		},
	}

	result := Lint(flow)

	hasInvalidTransition := false
	for _, err := range result.Errors {
		if err.Code == flows.ErrInvalidTransition {
			hasInvalidTransition = true
			break
		}
	}
	assert.True(t, hasInvalidTransition, "should detect invalid transition target")
}
