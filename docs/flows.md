# Flow Documentation

This document describes the XML schema and features available in Gollum flow definitions.

## Step Attributes

### verbose (optional boolean)

Controls whether LLM response output is shown in execution logs.

- **Default:** `false` (LLM responses are suppressed)
- `true`: LLM responses are logged at INFO level

Example:
```xml
<step type="llm" agent="assistant" verbose="true">
    <prompt>Generate a summary</prompt>
</step>
```

**Note:** This attribute only applies to LLM steps. It does not affect other step types.
