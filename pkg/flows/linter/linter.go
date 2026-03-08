package linter

import (
	"github.com/denkhaus/gollum/pkg/flows"
)

// Lint runs all linter phases on a flow
func Lint(flow *flows.Flow) *flows.LinterResult {
	result := &flows.LinterResult{}

	// Run all phases
	phase1 := &SchemaChecker{}
	phase1.Check(flow, result)

	phase2 := &ExpressionChecker{}
	phase2.Check(flow, result)

	phase3 := &GraphChecker{}
	phase3.Check(flow, result)

	// Determine validity
	result.Valid = len(result.Errors) == 0

	return result
}
