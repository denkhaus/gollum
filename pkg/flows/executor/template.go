package executor

import (
	"fmt"
	"regexp"

	"github.com/denkhaus/gollum/pkg/flows"
)

// Substitution matches ${input.field}, ${context.field}, ${output.field}, ${computed.field}
var subRegex = regexp.MustCompile(`\$\{(input|context|output|computed)\.([^}]+)\}`)

// substituteTemplate replaces variables in template strings (internal helper)
// Templates MUST use ${} notation (e.g., ${input.field}, ${context.field})
// Bare notation (input.field, context.field) is NOT supported in templates
func substituteTemplate(ctx ExecutionContext, tmpl string) string {
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
