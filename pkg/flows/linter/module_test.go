package linter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLintModule_SingleFile(t *testing.T) {
	// Create a temp file
	tmpDir := t.TempDir()
	flowPath := filepath.Join(tmpDir, "test.xml")

	content := `<?xml version="1.0" encoding="UTF-8"?>
<flow name="test-flow">
	<input>
		<string name="query" />
	</input>
	<output>
		<string name="result" />
	</output>
	<states>
		<state name="init" initial="true">
			<steps>
				<step>Process ${input.query}</step>
			</steps>
			<transitions>
				<transition to="done" />
			</transitions>
		</state>
		<state name="done" />
	</states>
</flow>
`
	err := os.WriteFile(flowPath, []byte(content), 0644)
	require.NoError(t, err)

	result := LintModule(flowPath)

	assert.True(t, result.Valid)
	assert.Len(t, result.Flows, 1)
	assert.Contains(t, result.Flows, flowPath)
}

func TestLintModule_DirectoryWithMainXML(t *testing.T) {
	// Create a module directory with main.xml
	tmpDir := t.TempDir()
	moduleDir := filepath.Join(tmpDir, "test-module")
	require.NoError(t, os.Mkdir(moduleDir, 0755))

	mainContent := `<?xml version="1.0" encoding="UTF-8"?>
<flow name="test-module">
	<input>
		<string name="query" />
	</input>
	<output>
		<string name="result" />
	</output>
	<states>
		<state name="init" initial="true">
			<steps>
				<step>Process ${input.query}</step>
			</steps>
			<transitions>
				<transition to="done" />
			</transitions>
		</state>
		<state name="done" />
	</states>
</flow>
`
	mainPath := filepath.Join(moduleDir, "main.xml")
	err := os.WriteFile(mainPath, []byte(mainContent), 0644)
	require.NoError(t, err)

	result := LintModule(moduleDir)

	assert.True(t, result.Valid)
	assert.Len(t, result.Flows, 1)
	assert.Contains(t, result.Flows, mainPath)
}

func TestLintModule_DirectoryWithoutMainXML(t *testing.T) {
	// Create a module directory without main.xml
	tmpDir := t.TempDir()
	moduleDir := filepath.Join(tmpDir, "test-module")
	require.NoError(t, os.Mkdir(moduleDir, 0755))

	result := LintModule(moduleDir)

	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0].Message, "module entry point not found")
}

func TestLintModule_NonexistentPath(t *testing.T) {
	result := LintModule("/nonexistent/path")

	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0].Message, "cannot access module path")
}

func TestLintModule_RecursiveDependencies(t *testing.T) {
	// Create a module with sub-flow dependencies
	tmpDir := t.TempDir()

	// Create main flow that calls a sub-flow
	mainContent := `<?xml version="1.0" encoding="UTF-8"?>
<flow name="caller">
	<input>
		<string name="query" />
	</input>
	<output>
		<string name="result" />
	</output>
	<states>
		<state name="init" initial="true">
			<steps>
				<call ref="callee">
					<param name="input_query">${input.query}</param>
					<output name="output_result" as="result" />
				</call>
			</steps>
			<transitions>
				<transition to="done" />
			</transitions>
		</state>
		<state name="done" />
	</states>
</flow>
`
	mainPath := filepath.Join(tmpDir, "main.xml")
	err := os.WriteFile(mainPath, []byte(mainContent), 0644)
	require.NoError(t, err)

	// Create callee flow
	calleeContent := `<?xml version="1.0" encoding="UTF-8"?>
<flow name="callee">
	<input>
		<string name="input_query" />
	</input>
	<output>
		<string name="output_result" />
	</output>
	<states>
		<state name="init" initial="true">
			<steps>
				<step>Process ${input.input_query}</step>
			</steps>
			<transitions>
				<transition to="done" />
			</transitions>
		</state>
		<state name="done" />
	</states>
</flow>
`
	calleePath := filepath.Join(tmpDir, "callee.xml")
	err = os.WriteFile(calleePath, []byte(calleeContent), 0644)
	require.NoError(t, err)

	result := LintModule(tmpDir)

	assert.True(t, result.Valid)
	assert.Len(t, result.Flows, 2, "should have linted both main and callee")
	assert.Contains(t, result.Flows, mainPath)
	assert.Contains(t, result.Flows, calleePath)
}

func TestLintModule_ErrorInDependency(t *testing.T) {
	// Create a module where the main flow is valid but a sub-flow has errors
	tmpDir := t.TempDir()

	// Create valid main flow
	mainContent := `<?xml version="1.0" encoding="UTF-8"?>
<flow name="caller">
	<input>
		<string name="query" />
	</input>
	<output>
		<string name="result" />
	</output>
	<states>
		<state name="init" initial="true">
			<steps>
				<call ref="invalid-callee">
					<param name="input_query">${input.query}</param>
					<output name="output_result" as="result" />
				</call>
			</steps>
			<transitions>
				<transition to="done" />
			</transitions>
		</state>
		<state name="done" />
	</states>
</flow>
`
	mainPath := filepath.Join(tmpDir, "main.xml")
	err := os.WriteFile(mainPath, []byte(mainContent), 0644)
	require.NoError(t, err)

	// Create callee with error (missing initial state)
	invalidCalleeContent := `<?xml version="1.0" encoding="UTF-8"?>
<flow name="invalid-callee">
	<input>
		<string name="input_query" />
	</input>
	<output>
		<string name="output_result" />
	</output>
	<states>
		<state name="processing">
			<steps>
				<step>Process</step>
			</steps>
		</state>
	</states>
</flow>
`
	calleePath := filepath.Join(tmpDir, "invalid-callee.xml")
	err = os.WriteFile(calleePath, []byte(invalidCalleeContent), 0644)
	require.NoError(t, err)

	result := LintModule(tmpDir)

	assert.False(t, result.Valid, "should be invalid due to error in dependency")
	assert.Len(t, result.Flows, 2, "should have linted both flows")

	// The error should be in the callee
	calleeResult := result.Flows[calleePath]
	assert.False(t, calleeResult.Valid)
}

func TestLintModule_CircularDependency(t *testing.T) {
	// Create two flows that call each other
	tmpDir := t.TempDir()

	// Flow A calls flow B
	flowAContent := `<?xml version="1.0" encoding="UTF-8"?>
<flow name="flow-a">
	<input>
		<string name="query" />
	</input>
	<output>
		<string name="result" />
	</output>
	<states>
		<state name="init" initial="true">
			<steps>
				<call ref="flow-b">
					<param name="input_query">${input.query}</param>
					<output name="output_result" as="result" />
				</call>
			</steps>
			<transitions>
				<transition to="done" />
			</transitions>
		</state>
		<state name="done" />
	</states>
</flow>
`
	flowAPath := filepath.Join(tmpDir, "flow-a.xml")
	err := os.WriteFile(flowAPath, []byte(flowAContent), 0644)
	require.NoError(t, err)

	// Flow B calls flow A
	flowBContent := `<?xml version="1.0" encoding="UTF-8"?>
<flow name="flow-b">
	<input>
		<string name="input_query" />
	</input>
	<output>
		<string name="output_result" />
	</output>
	<states>
		<state name="init" initial="true">
			<steps>
				<call ref="flow-a">
					<param name="query">${input.input_query}</param>
					<output name="result" as="output_result" />
				</call>
			</steps>
			<transitions>
				<transition to="done" />
			</transitions>
		</state>
		<state name="done" />
	</states>
</flow>
`
	flowBPath := filepath.Join(tmpDir, "flow-b.xml")
	err = os.WriteFile(flowBPath, []byte(flowBContent), 0644)
	require.NoError(t, err)

	// Linting should not hang and should lint both flows exactly once
	result := LintModule(flowAPath)

	assert.True(t, result.Valid)
	assert.Len(t, result.Flows, 2, "should lint both flows exactly once despite circular dependency")
	assert.Contains(t, result.Flows, flowAPath)
	assert.Contains(t, result.Flows, flowBPath)
}

func TestModuleLinterResult_String(t *testing.T) {
	t.Run("valid result", func(t *testing.T) {
		result := &ModuleLinterResult{
			Valid: true,
			Flows: map[string]*flows.LinterResult{
				"/path/to/flow.xml": {Valid: true},
			},
		}

		// The String() method should work without panicking
		str := result.String()
		assert.NotEmpty(t, str)
	})

	t.Run("invalid result", func(t *testing.T) {
		result := &ModuleLinterResult{
			Valid: false,
			Flows: map[string]*flows.LinterResult{
				"/path/to/flow.xml": {
					Valid: false,
					Errors: []flows.LinterError{
						{Message: "test error"},
					},
				},
			},
		}

		str := result.String()
		assert.Contains(t, str, "invalid")
	})
}
