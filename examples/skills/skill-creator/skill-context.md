# Context Modes

Context modes control how skill execution relates to the parent agent's conversation history and state.

## Overview

When a skill is invoked, it can either:
- **Inherit** the parent agent's context (shared history)
- **Isolate** itself with a fresh context (clean slate)

This is controlled by the `context_mode` parameter passed during invocation, or can have a default configured in the skill.

## Inherited Context

With inherited context, the skill subagent shares the parent's message history.

### When to Use

- The skill needs context from previous messages
- Continuity is important for the task
- The skill builds on ongoing work

### Invocation

```json
{
  "name": "my-skill",
  "input": "Continue the analysis",
  "context_mode": "inherited"
}
```

### Behavior

```
Parent Agent History:
[User: "Analyze the auth module"]
[Agent: "I've found several issues..."]
[User: "Now check the database layer"]
[Agent: "Looking at the database..."]

Skill Invocation (inherited):
[Skill receives full history above]
[Skill can reference previous findings]
```

### Example Skill

```yaml
---
name: follow-up-analyzer
description: Performs follow-up analysis based on previous context
type: agent
---

# Follow-up Analyzer

Based on the previous analysis in our conversation, perform additional investigation...

Review what was already discovered and extend the analysis to cover:
- Related components
- Deeper issues
- Recommendations
```

## Isolated Context

With isolated context, the skill starts with no prior conversation history.

### When to Use

- The skill should be unbiased
- Fresh perspective is needed
- Previous context would be distracting
- Security or isolation requirements

### Invocation

```json
{
  "name": "my-skill",
  "input": "Review this code for issues",
  "context_mode": "isolated"
}
```

### Behavior

```
Parent Agent History:
[User: "Analyze the auth module"]
[Agent: "I've found several issues..."]
[User: "Now get a fresh review"]

Skill Invocation (isolated):
[Skill receives NO history]
[Skill starts with only the input prompt]
[Skill provides unbiased analysis]
```

### Example Skill

```yaml
---
name: fresh-code-review
description: Provides unbiased code review with fresh perspective
type: agent
---

# Fresh Code Review

Review the provided code with no preconceptions...

Analyze objectively without being influenced by previous discussions.
```

## Default Behavior

If no `context_mode` is specified, the default is `inherited`.

```go
// Default context mode
contextMode := ContextModeInherited
```

## Overriding at Invocation

The invoker can override any default:

```json
// Force isolated context for a skill that normally inherits
{
  "name": "follow-up-analyzer",
  "input": "Review independently",
  "context_mode": "isolated"
}
```

## Context Mode in Tool Response

The response includes the context mode used:

```json
{
  "status": "success",
  "skill_name": "my-skill",
  "skill_type": "agent",
  "context_mode": "inherited",
  "output": "..."
}
```

## Technical Implementation

### Context Inheritance Flow

```go
// Get parent agent for context inheritance
parentAgent, hasParent := t.registry.GetAgent(t.senderID)

var history *gollem.History
if contextMode == ContextModeInherited && hasParent {
    history, err = parentAgent.GetMessageHistory(ctx)
    if err != nil {
        // Non-fatal: continue without history
    }
}

// Create subagent with history
subagentConfig := &shared.AgentConfig{
    History: history,
    // ...
}
```

## Comparison

| Aspect | Inherited | Isolated |
|--------|-----------|----------|
| Message history | Shared | Empty |
| Context awareness | Full | None |
| Bias | May be influenced | Unbiased |
| Token usage | Higher | Lower |
| Continuity | Maintained | Reset |
| Security | Lower | Higher |

## Best Practices

### Use Inherited When:

1. The task requires continuity
2. Previous findings inform the skill's work
3. The user expects context awareness
4. Multi-step processes are involved

### Use Isolated When:

1. Fresh perspective is valuable
2. Previous context might bias results
3. Security requires isolation
4. The skill is self-contained

## Examples

### Multi-Step Analysis (Inherited)

```json
// Step 1: Initial analysis
{
  "name": "code-analyzer",
  "input": "Analyze the auth module",
  "context_mode": "inherited"
}

// Step 2: Follow-up (has context from step 1)
{
  "name": "code-analyzer",
  "input": "Now check the related database code",
  "context_mode": "inherited"
}
```

### Independent Reviews (Isolated)

```json
// First reviewer
{
  "name": "code-reviewer",
  "input": "Review src/auth/login.go",
  "context_mode": "isolated"
}

// Second reviewer (independent)
{
  "name": "code-reviewer",
  "input": "Review src/auth/login.go",
  "context_mode": "isolated"
}
```
