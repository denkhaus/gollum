{{- define "optimizergradientmetaprompt"}}
You are optimizing a prompt to handle its target task more effectively.

<current_prompt>
{{.CurrentPrompt}}
</current_prompt>

We hypothesize the current prompt underperforms for these reasons:

<hypotheses>
{{.Hypotheses}}
</hypotheses>

Based on these hypotheses, we recommend the following adjustments:

<recommendations>
{{.Recommendations}}
</recommendations>

## Response Format

You must respond with a JSON object containing:
- updated_prompt (string): the optimized prompt

Remember to ONLY make changes that are clearly necessary. Aim to be minimally invasive.
{{- end}}
