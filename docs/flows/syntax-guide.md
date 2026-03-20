# Flow Syntax Guide

## Field Reference Syntax

Gollum flows use two different syntax modes depending on the context:

### Expressions: NO `${}` Syntax

**Expression contexts** (no variable substitution):
- `when` attributes on transitions and calls
- `eval` attributes on computed fields

In expression contexts, reference fields **directly** without `${}`:

```xml
<!-- Correct: Expression without ${} -->
<transition to="next" when="EQ(computed.is_ready, true)" />

<call ref="other-flow" when="GT(input.count, 0)" />

<int name="doubled" eval="MUL(input.value, 2)" />
```

❌ **Wrong** - Using `${}` in expressions:
```xml
<transition to="next" when="${computed.is_ready}" />  <!-- Error: W001 -->
```

### Templates: MUST Use `${}` Syntax

**Template contexts** (variable substitution):
- `prompt` attributes on steps
- `cmd` attributes on shell steps
- `value` attributes on params

In template contexts, **wrap field references** in `${}`:

```xml
<!-- Correct: Template with ${} -->
<step type="llm">
    <prompt>Analyze ${input.target} and provide feedback</prompt>
</step>

<step type="shell">
    <cmd>gosec -fmt=json ${input.target_dir}</cmd>
</step>

<param name="format" value="${context.template}" />
```

❌ **Wrong** - Missing `${}` in templates:
```xml
<step type="llm">
    <prompt>Analyze input.target and provide feedback</prompt>  <!-- Error: W002 -->
</step>
```

### Field References: NO `${}` Syntax

**Field reference contexts** (pointing to fields, not values):
- `from` and `to` parameters in `assign` function
- `value` attributes in call input/output

These reference **where** to read/write, not the value itself:

```xml
<!-- Correct: Assign function -->
<step type="func" function="assign">
    <params>
        <param name="from" value="computed.sum" />  <!-- Plain prefix.field -->
        <param name="to" value="output.result" />
    </params>
</step>

<!-- Correct: Call input/output -->
<call ref="sub-flow">
    <input>
        <string name="source" value="input.target" />  <!-- Plain prefix.field -->
    </input>
    <output>
        <int name="score" value="context.analysis" />  <!-- Plain prefix.field -->
    </output>
</call>
```

❌ **Wrong** - Using `${}` in field references:
```xml
<param name="from" value="${computed.sum}" />  <!-- Wrong: not a template -->
```

## Field Scopes

When referencing fields, always use the absolute path syntax:

| Scope | Prefix | Example |
|-------|--------|---------|
| Input | `input.` | `${input.target}`, `input.value` |
| Context | `context.` | `${context.scan_output}`, `context.count` |
| Output | `output.` | `${output.result}`, `output.issues` |
| Computed | `computed.` | `${computed.is_ready}`, `computed.doubled` |
| System | `sys.` | `${sys.error.message}`, `${sys.error.step}` |

## Output Field Bindings

Output fields can optionally declare their source using the `from` attribute:

### Declarative Output (with `from`)

Output fields with `from` are automatically populated from the specified source:

```xml
<output>
    <int name="sum" from="computed.sum" />
    <string name="message" from="input.text" />
    <bool name="is_valid" from="computed.is_valid" />
</output>
```

**Key properties:**
- **Readonly at runtime** - Cannot be set by tools/steps via `set_output_value`
- **Auto-populated** - Value flows automatically after source evaluation
- **Type-safe** - Linter validates source exists and type matches

### Imperative Output (without `from`)

Output fields without `from` are set by tools/steps at runtime:

```xml
<output>
    <string name="result" />
    <int name="count" />
</output>
```

### Allowed Sources

| Source | Example | When to Use |
|--------|---------|-------------|
| `input.*` | `from="input.value"` | Pass input through to output |
| `context.*` | `from="context.status"` | Expose context value as output |
| `computed.*` | `from="computed.sum"` | Expose computed value as output |
| `output.*` | `from="output.other"` | Alias another output field |

### Migration from `assign` Function

**Before (verbose):**
```xml
<state name="assign" initial="true">
    <steps>
        <step type="func" function="assign">
            <params>
                <param name="from" value="computed.sum" />
                <param name="to" value="output.sum" />
            </params>
        </step>
    </steps>
    <transitions>
        <transition to="done" />
    </transitions>
</state>
```

**After (declarative):**
```xml
<output>
    <int name="sum" from="computed.sum" />
</output>

<states>
    <state name="done" initial="true" />
</states>
```

## Linter Warnings

The linter enforces these syntax rules:

- **W001**: Expression contains `${}` syntax - use direct field references
- **W002**: Template may contain field references without `${}` - wrap in `${}`
- **W003**: Multiple output fields map from the same source (warning)

## Examples

### Complete Flow Example

```xml
<flow name="example" version="1.0">
    <input>
        <string name="target" required="true" />
    </input>

    <output>
        <string name="result" />
    </output>

    <context>
        <string name="template" />
        <bool name="is_ready" />
    </context>

    <computed>
        <bool name="is_ready" eval="EQ(input.target, '')" />
    </computed>

    <states>
        <state name="process" initial="true">
            <steps>
                <!-- Template: uses ${} -->
                <step type="llm">
                    <prompt>Analyze ${input.target} when ready</prompt>
                </step>

                <!-- Expression: NO ${} -->
                <step type="func" function="assign">
                    <params>
                        <param name="value" value="${output.result}" />
                    </params>
                </step>
            </steps>

            <!-- Expression: NO ${} -->
            <transitions>
                <transition to="done" when="computed.is_ready" />
            </transitions>
        </state>

        <state name="done" />
    </states>
</flow>
```

## Quick Reference

| Context | Syntax | Example |
|---------|--------|---------|
| `when` attribute | Direct | `when="EQ(input.x, 5)"` |
| `eval` attribute | Direct | `eval="ADD(input.a, input.b)"` |
| `prompt` content | `${}` | `${input.target}` |
| `cmd` content | `${}` | `${input.target_dir}` |
| `param` value | `${}` | `${context.template}` |
