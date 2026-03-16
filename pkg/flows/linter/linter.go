package linter

import (
	"github.com/denkhaus/gollum/pkg/flows"
)

// Lint runs all linter phases on a flow
func Lint(flow *flows.Flow) *flows.LinterResult {
	return LintPath("", flow)
}

// LintPath runs all linter phases on a flow with a known file path
func LintPath(flowPath string, flow *flows.Flow) *flows.LinterResult {
	result := &flows.LinterResult{}

	// Phase 1: Schema validation
	phase1 := &SchemaChecker{}
	phase1.Check(flow, result)

	// Phase 2: Expression validation
	phase2 := &ExpressionChecker{}
	phase2.Check(flow, result)

	// Phase 3: Graph validation
	phase3 := &GraphChecker{}
	phase3.Check(flow, result)

	// Phase 4: Call reference validation
	phase4 := NewCallChecker()
	phase4.Check(flowPath, flow, result)

	// Phase 5: Timeout validation
	phase5 := NewTimeoutChecker()
	phase5.Check(flowPath, flow, result)

	// Phase 6: Computed field validation
	phase6 := NewComputedChecker()
	phase6.Check(flow, result)

	// Phase 7: Syntax validation ($() rules)
	phase7 := &SyntaxChecker{}
	phase7.Check(flow, result)

	// Determine validity
	result.Valid = len(result.Errors) == 0

	return result
}
