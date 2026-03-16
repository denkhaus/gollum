package linter

import (
	"strings"

	"github.com/denkhaus/gollum/pkg/flows"
)

// SyntaxChecker validates $() syntax rules:
// - Expressions (when, eval, computed): NO $() syntax
// - Templates (prompt, cmd, param value): MUST use $() syntax
type SyntaxChecker struct{}

// Check runs all syntax checks
func (s *SyntaxChecker) Check(flow *flows.Flow, result *flows.LinterResult) {
	s.checkExpressions(flow, result)
	s.checkTemplates(flow, result)
}

// checkExpressions validates that expressions don't use $() syntax
func (s *SyntaxChecker) checkExpressions(flow *flows.Flow, result *flows.LinterResult) {
	// Check transition conditions
	for _, state := range flow.States {
		for _, trans := range state.Transitions {
			if trans.When != "" && strings.Contains(trans.When, "${") {
				result.Warnings = append(result.Warnings, flows.LinterError{
					Code:    "W001",
					Message: "Expression contains ${} syntax - expressions should reference fields directly (e.g., 'computed.value' not '${computed.value}')",
					Context: trans.When,
				})
			}
		}

		// Check call conditions
		for _, call := range state.Calls {
			if call.When != "" && strings.Contains(call.When, "${") {
				result.Warnings = append(result.Warnings, flows.LinterError{
					Code:    "W001",
					Message: "Expression contains ${} syntax - expressions should reference fields directly (e.g., 'input.value' not '${input.value}')",
					Context: call.When,
				})
			}
		}

		// Check computed field expressions
		if flow.Computed != nil {
			for _, field := range flow.Computed.GetAllFields() {
				if strings.Contains(field.Eval, "${") {
					result.Warnings = append(result.Warnings, flows.LinterError{
						Code:    "W001",
						Message: "Computed expression contains ${} syntax - expressions should reference fields directly (e.g., 'input.value' not '${input.value}')",
						Context: field.Eval,
					})
				}
			}
		}
	}
}

// checkTemplates validates that templates use $() for variable substitution
func (s *SyntaxChecker) checkTemplates(flow *flows.Flow, result *flows.LinterResult) {
	// Define patterns that look like field references but are missing ${}
	fieldRefPattern := `\b(input\.|output\.|context\.|computed\.|sys\.)[a-zA-Z_][a-zA-Z0-9_]*\b`

	for _, state := range flow.States {
		for _, step := range state.Steps {
			// Check prompt templates
			if step.Prompt != "" {
				s.checkTemplateForFieldRefs(step.Prompt, "prompt", result, fieldRefPattern)
			}

			// Check command templates
			if step.Cmd != "" {
				s.checkTemplateForFieldRefs(step.Cmd, "command", result, fieldRefPattern)
			}

			// Check param value templates
			for _, param := range step.Params {
				if param.Value != "" {
					s.checkTemplateForFieldRefs(param.Value, "param", result, fieldRefPattern)
				}
			}
		}

		// Check call input/output values
		for _, call := range state.Calls {
			if call.Input != nil {
				for _, field := range call.Input.GetFields() {
					if value := s.getCallFieldValue(field); value != "" {
						s.checkTemplateForFieldRefs(value, "call input", result, fieldRefPattern)
					}
				}
			}
			if call.Output != nil {
				for _, field := range call.Output.GetFields() {
					if value := s.getCallOutputFieldValue(field); value != "" {
						s.checkTemplateForFieldRefs(value, "call output", result, fieldRefPattern)
					}
				}
			}
		}
	}
}

// checkTemplateForFieldRefs checks if a template contains field references without ${}
func (s *SyntaxChecker) checkTemplateForFieldRefs(template, context string, result *flows.LinterResult, pattern string) {
	// Skip if template already uses ${} syntax correctly
	if strings.Contains(template, "${") {
		return
	}

	// Simple check for field reference patterns without ${}
	// This is a basic heuristic - we look for patterns like "input.value" but not "${input.value}"
	if s.containsFieldRefWithoutBrackets(template, pattern) {
		result.Warnings = append(result.Warnings, flows.LinterError{
			Code:    "W002",
			Message: "Template may contain field references without ${} syntax - use ${input.field}, ${context.field}, etc.",
			Context: template,
		})
	}
}

// containsFieldRefWithoutBrackets is a simple heuristic to find field references
func (s *SyntaxChecker) containsFieldRefWithoutBrackets(template, pattern string) bool {
	// This is a basic implementation - we check if the template contains
	// known prefixes followed by field names, but not wrapped in ${}
	knownPrefixes := []string{"input.", "output.", "context.", "computed.", "sys."}

	for _, prefix := range knownPrefixes {
		// Look for prefix followed by identifier (but not as part of ${})
		// This is a simplified check - a full implementation would need proper parsing
		if strings.Contains(template, prefix) {
			// Found a prefix - check if it's followed by a field name pattern
			afterPrefix := strings.Split(template, prefix)[1]
			if len(afterPrefix) > 0 {
				// Simple heuristic: check if next word starts with lowercase letter
				firstWord := strings.Split(afterPrefix, " ")[0]
				firstWord = strings.Split(firstWord, "\n")[0]
				firstWord = strings.Split(firstWord, "\t")[0]
				firstWord = strings.Split(firstWord, ",")[0]
				firstWord = strings.Split(firstWord, ".")[0]
				firstWord = strings.Split(firstWord, ")")[0]
				firstWord = strings.Split(firstWord, "}")[0]
				firstWord = strings.Split(firstWord, ":")[0]
				firstWord = strings.Split(firstWord, "+")[0]
				firstWord = strings.Split(firstWord, "-")[0]
				firstWord = strings.Split(firstWord, "*")[0]
				firstWord = strings.Split(firstWord, "/")[0]

				if len(firstWord) > 0 && firstWord[0] >= 'a' && firstWord[0] <= 'z' {
					return true
				}
			}
		}
	}

	return false
}

// getCallFieldValue extracts the Value from a CallInputField (union type)
func (s *SyntaxChecker) getCallFieldValue(field flows.CallInputField) string {
	if field.String != nil {
		return field.String.Value
	}
	if field.Int != nil {
		return field.Int.Value
	}
	if field.Bool != nil {
		return field.Bool.Value
	}
	if field.Float != nil {
		return field.Float.Value
	}
	return ""
}

// getCallOutputFieldValue extracts the Value from a CallOutputField (union type)
func (s *SyntaxChecker) getCallOutputFieldValue(field flows.CallOutputField) string {
	if field.String != nil {
		return field.String.Value
	}
	if field.Int != nil {
		return field.Int.Value
	}
	if field.Bool != nil {
		return field.Bool.Value
	}
	if field.Float != nil {
		return field.Float.Value
	}
	return ""
}
