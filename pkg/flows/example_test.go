package flows_test

import (
	"path/filepath"
	"testing"

	"github.com/denkhaus/gollum/pkg/flows/linter"
	"github.com/denkhaus/gollum/pkg/flows/parser"
	"github.com/stretchr/testify/assert"
)

func TestParseAndLintAllExamples(t *testing.T) {
	// Get project root (2 levels up from pkg/flows directory)
	examplesDir := filepath.Join("..", "..", ".gollum", "flows", "examples")

	files := []string{
		"simple-flow.xml",
		"step-types-example.xml",
		"conditional-flow-calls.xml",
		"nested-context-example.xml",
	}

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			path := filepath.Join(examplesDir, file)

			// Parse
			flow, err := parser.Parse(path)
			assert.NoError(t, err, "should parse %s", file)
			assert.NotNil(t, flow, "flow should not be nil")

			// Lint
			result := linter.Lint(flow)

			// For now, just report - we expect some errors as we refine the spec
			t.Logf("Flow: %s, Valid: %v, Errors: %d, Warnings: %d",
				flow.Name, result.Valid, len(result.Errors), len(result.Warnings))

			if len(result.Errors) > 0 {
				for _, e := range result.Errors {
					t.Logf("  Error: %s - %s", e.Code, e.Message)
				}
			}
		})
	}
}
