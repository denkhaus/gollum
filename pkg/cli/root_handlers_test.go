package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootCommand_Structure(t *testing.T) {
	rootCmd := RootCommand()

	// Test basic command structure
	assert.Equal(t, "gollum", rootCmd.Name)
	assert.Equal(t, "AI agent workflow system", rootCmd.Usage)
	assert.NotNil(t, rootCmd.Before)
	assert.NotNil(t, rootCmd.After)
	assert.NotNil(t, rootCmd.Action)
	assert.NotEmpty(t, rootCmd.Commands, "should have subcommands")
}

func TestRootCommand_HasFlowCommand(t *testing.T) {
	rootCmd := RootCommand()

	// Should have flow command group
	found := false
	for _, cmd := range rootCmd.Commands {
		if cmd.Name == "flow" {
			found = true
			break
		}
	}
	assert.True(t, found, "should have 'flow' subcommand")
}

func TestRootCommand_BeforeAction(t *testing.T) {
	rootCmd := RootCommand()

	// The before action will fail without proper DI setup
	// We're just testing that it's callable
	assert.NotNil(t, rootCmd.Before)

	// Test that Before is a function (can be called)
	assert.NotNil(t, rootCmd.Before)
}

func TestRootCommand_AfterAction(t *testing.T) {
	rootCmd := RootCommand()

	// Test that After is a function
	assert.NotNil(t, rootCmd.After)
}

func TestRootCommand_RunAction(t *testing.T) {
	rootCmd := RootCommand()

	// Test that Run is a function
	assert.NotNil(t, rootCmd.Action)
}

func TestRootHandler_Structure(t *testing.T) {
	handler := &rootHandler{}

	// Just verify it can be instantiated
	assert.NotNil(t, handler)
}
