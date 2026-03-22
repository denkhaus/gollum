package linter

import (
	"github.com/denkhaus/gollum/pkg/flows"
)

func init() {
	// Register this package as the linter implementation
	flows.RegisterLintRunner(lintRunnerImpl{})
}

// lintRunnerImpl implements flows.LintRunner interface
type lintRunnerImpl struct{}

func (l lintRunnerImpl) LintFlow(flow *flows.Flow, flowPath, xmlContent string) *flows.LinterResult {
	return LintWithContent(flowPath, xmlContent, flow)
}

// Lint runs all linter phases on a flow (backward compatibility wrapper)
func Lint(flow *flows.Flow) *flows.LinterResult {
	return flow.Lint()
}

// LintPath runs all linter phases on a flow with a known file path (backward compatibility wrapper)
func LintPath(flowPath string, flow *flows.Flow) *flows.LinterResult {
	return flow.LintPath(flowPath)
}

// LintWithContent runs all linter phases on a flow with raw XML content for position tracking
// Note: This is the actual implementation used by Flow.LintWithContent
func LintWithContent(flowPath, xmlContent string, flow *flows.Flow) *flows.LinterResult {
	result := &flows.LinterResult{}

	// Create position tracker if XML content is provided
	var posTracker *PositionTracker
	if xmlContent != "" {
		posTracker = NewPositionTracker(xmlContent)
	}

	// Phase 1: Schema validation
	phase1 := &SchemaChecker{PosTracker: posTracker}
	phase1.Check(flow, result)

	// Phase 2: Expression validation
	phase2 := &ExpressionChecker{PosTracker: posTracker}
	phase2.Check(flow, result)

	// Phase 3: Graph validation
	phase3 := &GraphChecker{PosTracker: posTracker}
	phase3.Check(flow, result)

	// Phase 4: Call reference validation
	phase4 := NewCallChecker()
	phase4.PosTracker = posTracker
	phase4.Check(flowPath, flow, result)

	// Phase 5: Timeout validation
	phase5 := NewTimeoutChecker()
	phase5.PosTracker = posTracker
	phase5.Check(flowPath, flow, result)

	// Phase 6: Computed field validation
	phase6 := NewComputedChecker()
	phase6.PosTracker = posTracker
	phase6.Check(flow, result)

	// Phase 7: Syntax validation ($() rules)
	phase7 := &SyntaxChecker{PosTracker: posTracker}
	phase7.Check(flow, result)

	// Phase 8: Output binding validation
	phase8 := NewOutputBindingsChecker()
	phase8.PosTracker = posTracker
	phase8.Check(flow, result)

	// Determine validity
	result.Valid = len(result.Errors) == 0

	return result
}
