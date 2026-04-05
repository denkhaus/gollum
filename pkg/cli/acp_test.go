package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestACPCommand(t *testing.T) {
	acpCmd := ACPCommand()

	assert.Equal(t, "acp", acpCmd.Name)
	assert.Equal(t, "Start Gollum ACP server (Agent Client Protocol)", acpCmd.Usage)
	assert.NotNil(t, acpCmd.Action)
}
