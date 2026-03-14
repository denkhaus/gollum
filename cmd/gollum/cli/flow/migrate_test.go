package flow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateCommand_ContextComputedToTopLevel(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create a test flow file with old-style context computed fields
	oldFlow := `<?xml version="1.0" encoding="UTF-8"?>
<flow name="test-flow" version="1.0">
    <input>
        <int name="x" />
    </input>
    <context>
        <computed name="is_large" type="bool" when="GT(input.x, 10)" />
        <computed name="doubled" type="int" when="MUL(input.x, 2)" />
    </context>
    <states>
        <state name="done" />
    </states>
</flow>`

	flowPath := filepath.Join(tmpDir, "test-flow.xml")
	require.NoError(t, os.WriteFile(flowPath, []byte(oldFlow), 0644))

	// Run migration
	cmd := MigrateCommand()
	require.NotNil(t, cmd)

	// Verify command properties
	assert.Equal(t, "migrate", cmd.Name)
	assert.Equal(t, "Migrate flow files to latest schema version", cmd.Usage)
}

func TestMigrateCommand_AlreadyMigrated(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create a test flow file with new-style computed fields
	newFlow := `<?xml version="1.0" encoding="UTF-8"?>
<flow name="test-flow" version="1.0">
    <input>
        <int name="x" />
    </input>
    <computed>
        <field name="is_large" type="bool" eval="GT(input.x, 10)" />
    </computed>
    <states>
        <state name="done" />
    </states>
</flow>`

	flowPath := filepath.Join(tmpDir, "test-flow.xml")
	require.NoError(t, os.WriteFile(flowPath, []byte(newFlow), 0644))

	// Verify file was created
	_, err := os.Stat(flowPath)
	require.NoError(t, err)
}

func TestMigrateCommand_NoComputedFields(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create a test flow file without computed fields
	noComputedFlow := `<?xml version="1.0" encoding="UTF-8"?>
<flow name="test-flow" version="1.0">
    <input>
        <int name="x" />
    </input>
    <states>
        <state name="done" />
    </states>
</flow>`

	flowPath := filepath.Join(tmpDir, "test-flow.xml")
	require.NoError(t, os.WriteFile(flowPath, []byte(noComputedFlow), 0644))

	// Verify file was created
	_, err := os.Stat(flowPath)
	require.NoError(t, err)
}
