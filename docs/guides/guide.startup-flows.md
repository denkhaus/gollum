# Startup Flows Guide

## Overview

Startup flows are special flows that execute when Gollum starts.
They can analyze your workspace and provide context that enriches
the main agent's understanding.

## How Startup Flows Work

```
┌─────────────────┐
│  Gollum Starts  │
└────────┬────────┘
         │
         ▼
┌─────────────────────┐
│  Startup Flow Runs  │
│  - Read files       │
│  - Analyze repo     │
│  - Compute context  │
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│  Context Stored     │
│  in StartupContext  │
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│  Main Agent Starts  │
│  with enriched      │
│  system prompt      │
└─────────────────────┘
```

## Creating a Startup Flow

1. Create the directory: `.gollum/flows/startup/`
2. Create your flow file: `.gollum/flows/startup/main.xml`
3. Output a `context` field with natural language text

## Example Flow

```xml
<flow name="startup">
    <output>
        <strings>
            <field name="context" type="string" />
        </strings>
    </output>

    <states>
        <state name="main" initial="true">
            <steps>
                <!-- Your flow logic here -->
                <func builtin="assign">
                    <params>
                        <param name="from" value="Your context text here" />
                        <param name="to" value="${output.context}" />
                    </params>
                </func>
            </steps>
        </state>
    </states>
</flow>
```

## What Startup Flows Can Do

- Read project documentation (README, CONTRIBUTING, etc.)
- Analyze project structure
- Load configuration files
- Compute project-specific context
- Set behavioral preferences

## Error Handling

- If the startup flow fails, Gollum continues normally
- If no startup flow exists, Gollum starts without enhanced context
- The agent's system prompt remains clean when no context is available

## See Also

- [Example Startup Flow](../../examples/startup-flow/)
- [Startup Context Design](../superpowers/specs/2026-04-25-startup-context-design.md)
