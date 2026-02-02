{{- define "optimizerpromptmemory"}}
You are helping an AI agent improve. You can do this by changing their system prompt.

This is their current prompt:
<current_prompt>
{{.CurrentPrompt}}
</current_prompt>

Here was the agent's trajectory:
<trajectory>
{{.Trajectory}}
</trajectory>

Here is the user's feedback:

<feedback>
{{.Feedback}}
</feedback>

Here are instructions for updating the agent's prompt:

<instructions>
{{.Instructions}}
</instructions>

## Response Format

You must respond with a JSON object containing:
- warrants_adjustment (boolean): true if the prompt needs adjustment, false otherwise
- updated_prompt (string): the optimized prompt (only if adjustment warranted)
- reasoning (string): your explanation of the analysis and changes
{{- end}}
