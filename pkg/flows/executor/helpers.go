package executor

import (
	"fmt"
	"strings"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/denkhaus/gollum/pkg/shared"
)

// parseAssignTarget parses an assignTo string in the format "scope.field".
// Returns the scope and field name, or an error if the format is invalid.
func (p *flowExecutorImpl) parseAssignTarget(assignTo string) (flows.FlowVariableScope, string, error) {
	// Split by first dot
	parts := strings.SplitN(assignTo, ".", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid assignTo format '%s': must be 'scope.field' (e.g., 'output.result' or 'context.temp')", assignTo)
	}
	scope := flows.FlowVariableScope(parts[0])
	if scope != flows.FlowVariableScopeOutput && scope != flows.FlowVariableScopeContext {
		return "", "", fmt.Errorf("invalid scope '%s' in assignTo: must be 'output' or 'context'", scope)
	}
	return scope, parts[1], nil
}

// resolveAssignFrom resolves a bare notation reference (e.g., "input.field") to its string value.
// Returns the string value of the referenced field, or empty string if not found.
func (p *flowExecutorImpl) resolveAssignFrom(assignFrom string) string {
	// Parse scope.field
	parts := strings.SplitN(assignFrom, ".", 2)
	if len(parts) != 2 {
		return "" // Invalid format, return empty string
	}

	scope := flows.FlowVariableScope(parts[0])
	field := parts[1]

	// Get value based on scope
	switch scope {
	case flows.FlowVariableScopeInput:
		if val := p.ctx.GetInput(field); val != nil {
			return shared.AnyToString(val)
		}
	case flows.FlowVariableScopeContext:
		if val, err := p.ctx.GetContextField(field); err == nil {
			return shared.AnyToString(val)
		}
	case flows.FlowVariableScopeOutput:
		if val, err := p.ctx.GetOutputField(field); err == nil {
			return shared.AnyToString(val)
		}
	case flows.FlowVariableScopeComputed:
		if val, err := p.ctx.GetComputedField(field); err == nil {
			return shared.AnyToString(val)
		}
	case flows.FlowVariableScopeSys:
		// System fields - handle specially if needed
		return ""
	}

	return ""
}
