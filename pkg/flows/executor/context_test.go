package executor

import (
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
)

func TestNewContext_InitializesWithDefaults(t *testing.T) {
	input := &flows.InputBlock{
		Strings: []flows.FieldDef{{Name: "repo", Required: true}},
	}
	inputVals := map[string]any{"repo": "gollum"}

	ctx := NewContext(input, inputVals)

	assert.Equal(t, "gollum", ctx.GetInput("repo"))
	assert.NotNil(t, ctx.values)
	assert.NotNil(t, ctx.computed)
}

func TestNewContext_AppliesInputDefaults(t *testing.T) {
	input := &flows.InputBlock{
		Strings: []flows.FieldDef{{Name: "owner", Default: "denkhaus"}},
	}

	ctx := NewContext(input, nil)

	assert.Equal(t, "denkhaus", ctx.GetInput("owner"))
}
