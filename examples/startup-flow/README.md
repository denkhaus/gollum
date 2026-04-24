# Startup Flow Example

This example demonstrates how to create a startup flow that
provides context to the main agent.

## What is a Startup Flow?

A startup flow is a special flow that executes when Gollum starts.
It can analyze your workspace, read files, and provide context
that enriches the main agent's understanding.

## How It Works

1. Place your flow at `.gollum/flows/startup/main.xml`
2. The flow runs when Gollum starts
3. Flow output is captured as `StartupContext`
4. Context is injected into the main agent's system prompt

## This Example

This example flow:
1. Checks for a `README.md` in your workspace
2. If found, reads it and creates context
3. If not found, provides default context

The output is a natural language description that helps
the agent understand your project better.

## Creating Your Own

Modify the flow to:
- Read specific documentation files
- Analyze project structure
- Load configuration
- Compute project-specific context

The only requirement is that your flow outputs a `context` field
(type: string) with natural language text.

## Output Format

The startup flow should output a `context` field:

```xml
<output>
    <strings>
        <field name="context" type="string" />
    </strings>
</output>
```

This context will be automatically injected into the agent's
system prompt as:

```
<StartupContext>
[your context text here]
</StartupContext>
```
