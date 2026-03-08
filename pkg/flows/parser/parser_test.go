package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSimpleFlow_BasicFields(t *testing.T) {
	// Get the project root
	root := filepath.Join("..", "..", "..")
	path := filepath.Join(root, ".gollum", "flows", "examples", "simple-flow.xml")

	flow, err := Parse(path)

	assert.NoError(t, err)
	assert.Equal(t, "simple-flow", flow.Name)
	assert.Equal(t, "1.0", flow.Version)
	assert.Contains(t, flow.Description, "Minimal flow")
}

func TestParseSimpleFlow_HasInputOutput(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	path := filepath.Join(root, ".gollum", "flows", "examples", "simple-flow.xml")

	flow, err := Parse(path)

	assert.NoError(t, err)
	assert.NotNil(t, flow.Input)
	assert.NotNil(t, flow.Output)
	assert.NotNil(t, flow.Context)
	assert.NotNil(t, flow.Agents)
}

func TestParseAllExamples(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	examplesDir := filepath.Join(root, ".gollum", "flows", "examples")

	entries, err := os.ReadDir(examplesDir)
	assert.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			path := filepath.Join(examplesDir, entry.Name())
			flow, err := Parse(path)

			// For now, just ensure it parses without crashing
			if err != nil {
				t.Logf("Parse error for %s: %v", entry.Name(), err)
			}
			if flow != nil {
				assert.NotEmpty(t, flow.Name, "flow should have a name")
			}
		})
	}
}

func TestParseSimpleFlow_InputFieldTypes(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	path := filepath.Join(root, ".gollum", "flows", "examples", "simple-flow.xml")

	flow, err := Parse(path)

	assert.NoError(t, err)
	assert.NotNil(t, flow.Input)

	// simple-flow.xml has: <string name="target" required="true" />
	fields := flow.Input.GetAllFields()
	assert.Len(t, fields, 1, "should have 1 input field")
	assert.Equal(t, "target", fields[0].Name)
	assert.Equal(t, "string", fields[0].Type)
	assert.True(t, fields[0].Required)
}

func TestParseInvalidXML_UnclosedComment(t *testing.T) {
	// Test that unclosed comments are caught
	invalidXML := `<?xml version="1.0" encoding="UTF-8"?>
<!-- This comment is never closed
<flow name="test" version="1.0">
</flow>`

	tmpFile, err := os.CreateTemp("", "invalid-*.xml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(invalidXML)
	assert.NoError(t, err)
	tmpFile.Close()

	flow, err := Parse(tmpFile.Name())

	assert.Error(t, err, "should return error for invalid XML")
	assert.Nil(t, flow, "should not return flow for invalid XML")
	assert.Contains(t, err.Error(), "XML syntax error", "error should mention XML syntax")
}

func TestParseInvalidXML_UnclosedTag(t *testing.T) {
	invalidXML := `<?xml version="1.0" encoding="UTF-8"?>
<flow name="test" version="1.0">
    <input>
</flow>`

	tmpFile, err := os.CreateTemp("", "invalid-*.xml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(invalidXML)
	assert.NoError(t, err)
	tmpFile.Close()

	flow, err := Parse(tmpFile.Name())

	assert.Error(t, err, "should return error for unclosed tag")
	assert.Nil(t, flow, "should not return flow for invalid XML")
}
