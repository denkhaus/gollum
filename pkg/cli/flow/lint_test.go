package flow

import (
	"bytes"
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/flows/linter"
	"github.com/stretchr/testify/assert"
)

func TestLintCommand(t *testing.T) {
	lintCmd := LintCommand()

	assert.Equal(t, "lint", lintCmd.Name)
	assert.Equal(t, "Validate a flow file or module directory", lintCmd.Usage)
	assert.NotNil(t, lintCmd.Action)
}

func TestWriteLintResult_Valid(t *testing.T) {
	var buf bytes.Buffer
	result := &linter.ModuleLinterResult{
		Valid: true,
		Flows: map[string]*flows.LinterResult{
			"test.xml": {Valid: true},
		},
	}

	err := WriteLintResult(&buf, result)
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "test.xml")
}

func TestWriteLintResult_Invalid(t *testing.T) {
	var buf bytes.Buffer
	result := &linter.ModuleLinterResult{
		Valid: false,
		Flows: map[string]*flows.LinterResult{
			"test.xml": {Valid: false},
		},
	}

	err := WriteLintResult(&buf, result)
	assert.Error(t, err) // Should return exit code error
}
