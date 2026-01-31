# LangMEM Prompt Templates

**Source:** [langchain-ai/langmem](https://github.com/langchain-ai/langmem)
**Extracted:** 2025-01-31
**Purpose:** Reference for implementing Go-based Prompt Optimizer

---

## 1. Gradient Optimizer Prompts

### 1.1 DEFAULT_GRADIENT_PROMPT

Used for reflection and analysis phase - identifies what needs improvement.

```text
You are reviewing the performance of an AI assistant in a given interaction.

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
3. Based on these failure modes, recommend the most suitable edit strategy. For example, consider::
   - Use synthetic few-shot examples for style or clarifying decision boundaries.
   - Use explicit instruction updates for conditionals, rules, or logic fixes.
   - Provide step-by-step reasoning guidelines for multi-step logic problems.
4. Provide detailed, concrete suggestions for how to update the prompt accordingly.

But remember, the final updated prompt should only be changed if there is evidence of poor performance, and our recommendations should be minimally invasive.
Do not recommend generic changes that aren't clearly linked to failure modes.

First think through the conversation and critique the current behavior.
If you believe the prompt needs to further adapt to the target context, provide precise recommendations.
Otherwise, mark `warrants_adjustment` as False and respond with 'No recommendations.'
```

### 1.2 DEFAULT_GRADIENT_METAPROMPT

Used for applying the recommended improvements to the prompt.

```text
You are optimizing a prompt to handle its target task more effectively.

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

Respond with the updated prompt. Remember to ONLY make changes that are clearly necessary. Aim to be minimally invasive:
```

---

## 2. Meta-Prompt Optimizer Template

### 2.1 DEFAULT_METAPROMPT

Combined reflection and update in a single strategy.

```text
You are helping an AI assistant learn by optimizing its prompt.

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

IFF changes are warranted, focus on actionable edits. Be concrete. Edits should be appropriate for the identified failure modes. For example, consider synthetic few-shot examples for style or clarifying decision boundaries, or adding or modifying explicit instructions for conditionals, rules, or logic fixes; or provide step-by-step reasoning guidelines for multi-step logic problems if the model is failing to reason appropriately.
```

---

## 3. Prompt Memory Template

### 3.1 INSTRUCTION_REFLECTION_PROMPT

Simple single-shot prompt for basic optimization.

```text
You are helping an AI agent improve. You can do this by changing their system prompt.

These is their current prompt:
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

You should return the full prompt, so if there's anything from before that you want to include, make sure to do that. Feel free to override or change anything that seems irrelevant. You do not need to update the prompt - if you don't want to, just return `update_prompt = False` and an empty string for new prompt.
```

---

## 4. Key Variables for Template Rendering

| Variable | Description | Example |
|----------|-------------|---------|
| `{prompt}` / `{current_prompt}` | The prompt being optimized | "You are a helpful assistant..." |
| `{trajectories}` | Formatted conversation history | JSON or formatted text |
| `{update_instructions}` | Developer constraints on updates | "Keep the tone formal" |
| `{hypotheses}` | Identified problems | "Style mismatch detected" |
| `{recommendations}` | Suggested fixes | "Add explicit format instructions" |
| `{feedback}` | User feedback on performance | "Response was too verbose" |
| `{trajectory}` | Single conversation trajectory | Array of messages |
| `{instructions}` | Update guidelines | "Be minimal invasive" |

---

## 5. Go Template Adaptation Notes

When adapting to Go's `text/template` or `html/template`:

1. **Variable Syntax**: Python `{variable}` → Go `{{.Variable}}`
2. **Conditionals**: Use Go's template syntax for conditional rendering
3. **Loops**: For trajectory iteration, use `{{range .Trajectories}}`
4. **Escaping**: Use `{{template "content" .}}` for large text blocks

### Example Go Template Structure:

```go
type GradientPromptInput struct {
    Prompt              string
    Trajectories        string
    UpdateInstructions  string
}

const gradientPromptTemplate = `You are reviewing the performance of an AI assistant...

<current_prompt>
{{.Prompt}}
</current_prompt>

...
`
```

---

## 6. Reflection Tool Definitions

### Think Tool
```python
def think(thought: str) -> str:
    """A reflection tool, used to reason over complexities and hypothesize fixes."""
    return "Take your time thinking through problems."
```

### Critique Tool
```python
def critique(criticism: str) -> str:
    """A critique tool for diagnosing flaws in reasoning."""
    return "Reflect critically on the previous hypothesis."
```

### Recommend Tool
```python
def recommend(
    warrants_adjustment: bool,
    hypotheses: Optional[str] = None,
    full_recommendations: Optional[str] = None,
) -> str:
    """Decides whether a prompt should be adjusted."""
    return ""
```

---

## 7. Configuration Defaults

| Strategy | Max Reflection | Min Reflection | LLM Calls |
|----------|----------------|----------------|-----------|
| `gradient` | 5 | 1 | 2-10 |
| `metaprompt` | 5 | 1 | 1-5 |
| `prompt_memory` | N/A | N/A | 1 |

---

## See Also

- [LangMEM Optimizer Reference](./02-langmem-optimizer-reference.md)
- [LangMEM Architecture](./03-langmem-architecture.md)
- [Original Repository](https://github.com/langchain-ai/langmem)
