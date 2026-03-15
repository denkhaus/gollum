package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootCommand(t *testing.T) {
	rootCmd := RootCommand()

	assert.Equal(t, "gollum", rootCmd.Name)
	assert.Equal(t, "AI agent workflow system", rootCmd.Usage)
	assert.NotNil(t, rootCmd.Action)
}
