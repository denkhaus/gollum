package executor

import (
	"fmt"
	"regexp"
)

// Substitution matches ${input.field}, ${context.field}, ${output.field}
var subRegex = regexp.MustCompile(`\$\{(input|context|output)\.([^}]+)\}`)

// SubstituteTemplate replaces variables in template strings
func SubstituteTemplate(ctx *Context, tmpl string) string {
	return subRegex.ReplaceAllStringFunc(tmpl, func(match string) string {
		// Extract input.field from ${input.field}
		parts := subRegex.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}

		scope := parts[1]
		field := parts[2]

		var value any
		var ok bool

		switch scope {
		case "input":
			value = ctx.GetInput(field)
		case "context":
			value, ok = ctx.GetContextField(field)
			if !ok {
				value = nil
			}
		case "output":
			value, ok = ctx.GetOutputField(field)
			if !ok {
				value = nil
			}
		}

		if value == nil {
			return match
		}

		return fmt.Sprintf("%v", value)
	})
}
