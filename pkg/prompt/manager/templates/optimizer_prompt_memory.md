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


Based on this, return an updated prompt

You should return the full prompt, so if there's anything from before that you want to include, make sure to do that. Feel free to override or change anything that seems irrelevant. You do not need to update the prompt - if you don't want to, just return update_prompt = False and an empty string for new prompt.
{{- end}}
