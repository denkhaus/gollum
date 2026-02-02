// Package optimizer provides prompt templates for optimization strategies.
package optimizer

import (
	"github.com/m-mizutani/gollem"
)

// DEFAULT_GRADIENT_PROMPT is used for the reflection phase of gradient optimization.
// Placeholders: {prompt}, {trajectories}, {update_instructions}
const DEFAULT_GRADIENT_PROMPT = `You are reviewing the performance of an AI assistant in a given interaction.

## Instructions

The current prompt that was used for the session is provided below.

<current_prompt>
{prompt}
</current_prompt>

The developer provided the following instructions around when and how to update the prompt:

<update_instructions>
{update_instructions}
</update_instructions>

## Session data

Analyze the following trajectories (and any associated user feedback) (either conversations with a user or other work that was performed by the assistant):

<trajectories>
{trajectories}
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
3. Based on these failure modes, recommend the most suitable edit strategy. For example, consider:
   - Use synthetic few-shot examples for style or clarifying decision boundaries.
   - Use explicit instruction updates for conditionals, rules, or logic fixes.
   - Provide step-by-step reasoning guidelines for multi-step logic problems.
4. Provide detailed, concrete suggestions for how to update the prompt accordingly.

But remember, the final updated prompt should only be changed if there is evidence of poor performance, and our recommendations should be minimally invasive.
Do not recommend generic changes that aren't clearly linked to failure modes.

First think through the conversation and critique the current behavior.
If you believe the prompt needs to further adapt to the target context, provide precise recommendations.
Otherwise, mark warrants_adjustment as False and respond with 'No recommendations.'
`

// DEFAULT_GRADIENT_METAPROMPT is used for applying the recommended improvements.
// Placeholders: {current_prompt}, {hypotheses}, {recommendations}
const DEFAULT_GRADIENT_METAPROMPT = `You are optimizing a prompt to handle its target task more effectively.

<current_prompt>
{current_prompt}
</current_prompt>

We hypothesize the current prompt underperforms for these reasons:

<hypotheses>
{hypotheses}
</hypotheses>

Based on these hypotheses, we recommend the following adjustments:

<recommendations>
{recommendations}
</recommendations>

Respond with the updated prompt. Remember to ONLY make changes that are clearly necessary. Aim to be minimally invasive.`

// DEFAULT_METAPROMPT combines reflection and update in a single strategy.
// Placeholders: {prompt}, {trajectories}, {update_instructions}
const DEFAULT_METAPROMPT = `You are helping an AI assistant learn by optimizing its prompt.

## Background

Below is the current prompt:

<current_prompt>
{prompt}
</current_prompt>

The developer provided these instructions regarding when/how to update:

<update_instructions>
{update_instructions}
</update_instructions>

## Session Data
Analyze the session(s) (and any user feedback) below:

<trajectories>
{trajectories}
</trajectories>

## Instructions

1. Reflect on the agent's performance on the given session(s) and identify any real failure modes (e.g., style mismatch, unclear or incomplete instructions, flawed reasoning, etc.).
2. Recommend the minimal changes necessary to address any real failures. If the prompt performs perfectly, simply respond with the original prompt without making any changes.
3. Retain any f-string variables in the existing prompt exactly as they are (e.g. {{variable_name}}).

IFF changes are warranted, focus on actionable edits. Be concrete. Edits should be appropriate for the identified failure modes. For example, consider synthetic few-shot examples for style or clarifying decision boundaries, or adding or modifying explicit instructions for conditionals, rules, or logic fixes; or provide step-by-step reasoning guidelines for multi-step logic problems if the model is failing to reason appropriately.`

// DEFAULT_PROMPT_MEMORY is a simple single-shot prompt for basic optimization.
// Placeholders: {current_prompt}, {trajectory}, {feedback}, {instructions}
const DEFAULT_PROMPT_MEMORY = `You are helping an AI agent improve. You can do this by changing their system prompt.

This is their current prompt:
<current_prompt>
{current_prompt}
</current_prompt>

Here was the agent's trajectory:
<trajectory>
{trajectory}
</trajectory>

Here is the user's feedback:

<feedback>
{feedback}
</feedback>

Here are instructions for updating the agent's prompt:

<instructions>
{instructions}
</instructions>


Based on this, return an updated prompt

You should return the full prompt, so if there's anything from before that you want to include, make sure to do that. Feel free to override or change anything that seems irrelevant. You do not need to update the prompt - if you don't want to, just return 'warrants_adjustment = False' and the original prompt.`

// ThinkToolSpec defines the "think" tool for reflection in gradient strategy.
var ThinkToolSpec = gollem.ToolSpec{
	Name:        "think",
	Description: "A reflection tool for reasoning over complexities and hypothesizing fixes.",
	Parameters: map[string]*gollem.Parameter{
		"thought": {
			Type:        gollem.TypeString,
			Description: "The thought or reasoning to process",
			Required:    []string{"thought"},
		},
	},
	Required: []string{"thought"},
}

// CritiqueToolSpec defines the "critique" tool for diagnosing flaws in gradient strategy.
var CritiqueToolSpec = gollem.ToolSpec{
	Name:        "critique",
	Description: "A critique tool for diagnosing flaws in reasoning.",
	Parameters: map[string]*gollem.Parameter{
		"criticism": {
			Type:        gollem.TypeString,
			Description: "The critique or criticism of the current reasoning",
			Required:    []string{"criticism"},
		},
	},
	Required: []string{"criticism"},
}

// RecommendToolSpec defines the "recommend" tool for providing optimization recommendations.
var RecommendToolSpec = gollem.ToolSpec{
	Name:        "recommend",
	Description: "Decides whether a prompt should be adjusted and provides recommendations.",
	Parameters: map[string]*gollem.Parameter{
		"warrants_adjustment": {
			Type:        gollem.TypeBoolean,
			Description: "Whether the prompt warrants adjustment based on the analysis",
		},
		"hypotheses": {
			Type:        gollem.TypeString,
			Description: "Hypotheses about why the current prompt underperforms",
		},
		"full_recommendations": {
			Type:        gollem.TypeString,
			Description: "Detailed recommendations for prompt improvement",
		},
	},
	Required: []string{"warrants_adjustment"},
}
