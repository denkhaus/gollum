package flow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunCommand(t *testing.T) {
	runCmd := RunCommand()

	assert.Equal(t, "run", runCmd.Name)
	assert.Equal(t, "Execute a flow", runCmd.Usage)
	assert.NotNil(t, runCmd.Action)
}
