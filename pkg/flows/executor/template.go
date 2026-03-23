package executor

import (
	"fmt"
	"regexp"

	"github.com/denkhaus/gollum/pkg/flows"
)

// Substitution matches ${input.field}, ${context.field}, ${output.field}
// Also handles bare notation: input.field, context.field, output.field (but only when it's the whole value)
var subRegex = regexp.MustCompile(`\$\{(input|context|output|computed)\.([^}]+)\}`)
var bareRefRegex = regexp.MustCompile(`^(input|context|output|computed)\.([a-zA-Z0-9_]+)$`)

// substituteTemplate replaces variables in template strings (internal helper)
// Supports both ${scope.field} and bare scope.field notation
func substituteTemplate(ctx ExecutionContext, tmpl string) string {
	// First, check if the entire template is a bare reference (e.g., "input.text")
	if bareRefRegex.MatchString(tmpl) {
		parts := bareRefRegex.FindStringSubmatch(tmpl)
		if len(parts) == 3 {
			scope := flows.FlowVariableScope(parts[1])
			field := parts[2]

			var value any
			var err error

			switch scope {
			case flows.FlowVariableScopeInput:
				value = ctx.GetInput(field)
			case flows.FlowVariableScopeContext:
				value, err = ctx.GetContextField(field)
				if err != nil {
					value = nil
				}
			case flows.FlowVariableScopeOutput:
				value, err = ctx.GetOutputField(field)
				if err != nil {
					value = nil
				}
			case flows.FlowVariableScopeComputed:
				// computed values are evaluated via context
				value, err = ctx.GetContextField(field)
				if err != nil {
					value = nil
				}
			}

			if value != nil {
				return fmt.Sprintf("%v", value)
			}
			// If bare reference fails, return as-is (will be handled by error checking)
			return tmpl
		}
	}

	// Handle ${scope.field} notation in templates
	return subRegex.ReplaceAllStringFunc(tmpl, func(match string) string {
		// Extract input.field from ${input.field}
		parts := subRegex.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}

		scope := flows.FlowVariableScope(parts[1])
		field := parts[2]

		var value any
		var err error

		switch scope {
		case flows.FlowVariableScopeInput:
			value = ctx.GetInput(field)
		case flows.FlowVariableScopeContext:
			value, err = ctx.GetContextField(field)
			if err != nil {
				value = nil
			}
		case flows.FlowVariableScopeOutput:
			value, err = ctx.GetOutputField(field)
			if err != nil {
				value = nil
			}
		case flows.FlowVariableScopeComputed:
			// computed values are evaluated via context
			value, err = ctx.GetContextField(field)
			if err != nil {
				value = nil
			}
		}

		if value == nil {
			return match
		}

		return fmt.Sprintf("%v", value)
	})
}
