{{- define "optimizergradientprompt"}}
You are reviewing the performance of an AI assistant in a given interaction.

## Instructions

The current prompt that was used for the session is provided below.

<current_prompt>
{{.Prompt}}
</current_prompt>

The developer provided the following instructions around when and how to update the prompt:

<update_instructions>
{{.UpdateInstructions}}
</update_instructions>

## Session data

Analyze the following trajectories (and any associated user feedback) (either conversations with a user or other work that was performed by the assistant):

<trajectories>
{{.Trajectories}}
</trajectories>

## Task

Analyze the conversation, including the user's request and the assistant's response, and evaluate:
1. How effectively the assistant fulfilled the user's intent.
2. Where the assistant might have deviated from user expectations or the desired outcome.
3. Specific areas (correctness, completeness, style, tone, alignment, etc.) that need improvement.

If the prompt seems to do well, then no further action is needed. We ONLY recommend updates if there is evidence of failures.
When failures occur, we want to recommend the minimal required changes to fix the problem.

Focus on actionable changes and be concrete.

1. Summarize the key successes and failures in the assistant's response.
2. Identify which failure mode(s) best describe the issues (examples: style mismatch, unclear or incomplete instructions, flawed logic or reasoning, hallucination, etc.).
3. Based on these failure modes, recommend the most suitable edit strategy. For example, consider::
   - Use synthetic few-shot examples for style or clarifying decision boundaries.
   - Use explicit instruction updates for conditionals, rules, or logic fixes.
   - Provide step-by-step reasoning guidelines for multi-step logic problems.
4. Provide detailed, concrete suggestions for how to update the prompt accordingly.

But remember, the final updated prompt should only be changed if there is evidence of poor performance, and our recommendations should be minimally invasive.
Do not recommend generic changes that aren't clearly linked to failure modes.

First think through the conversation and critique the current behavior.
If you believe the prompt needs to further adapt to the target context, provide precise recommendations.
Otherwise, mark warrants_adjustment as False and respond with 'No recommendations.'

## Response Format

You must respond with a JSON object containing:
- warrants_adjustment (boolean): true if the prompt needs adjustment, false otherwise
- hypotheses (string): your analysis of what problems exist
- recommendations (string): your specific recommendations for fixing the problems
- reasoning (string): your explanation of the analysis
{{- end}}
