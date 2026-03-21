package executor

import (
	"fmt"
	"regexp"

	"github.com/denkhaus/gollum/pkg/flows"
)

// Substitution matches ${input.field}, ${context.field}, ${output.field}
var subRegex = regexp.MustCompile(`\$\{(input|context|output)\.([^}]+)\}`)

// SubstituteTemplate replaces variables in template strings
func SubstituteTemplate(ctx ExecutionContext, tmpl string) string {
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
		case flows.VarContainerTargetInput:
			value = ctx.GetInput(field)
		case flows.VarContainerTargetContext:
			value, err = ctx.GetContextField(field)
			if err != nil {
				value = nil
			}
		case flows.VarContainerTargetOutput:
			value, err = ctx.GetOutputField(field)
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
