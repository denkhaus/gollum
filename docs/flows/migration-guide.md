# Flow Migration Guide

## Migrating from `assign` Function to Output Bindings

The `assign` function is deprecated. Use output field bindings instead.

### Step 1: Identify `assign` Usage

Find all flows using `assign`:

```bash
grep -r 'function="assign"' .gollum/flows/
```

### Step 2: Replace with Output Bindings

**Before:**
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
<state name="done" />
```

**After:**
```xml
<output>
    <int name="sum" from="computed.sum" />
</output>

<states>
    <state name="done" initial="true" />
</states>
```

### Step 3: Remove Empty States

If the state only contained assign steps, remove it and make the target state initial.

### Step 4: Verify with Linter

```bash
gollum flow lint your-flow.xml
```

### Allowed Source Scopes

| Scope | Example | Description |
|-------|---------|-------------|
| `input` | `from="input.value"` | Reference input field |
| `context` | `from="context.status"` | Reference context field |
| `computed` | `from="computed.sum"` | Reference computed field |
| `output` | `from="output.other"` | Reference another output field |

### Common Patterns

#### Pass Input to Output
```xml
<input>
    <string name="message" />
</input>
<output>
    <string name="result" from="input.message" />
</output>
```

#### Expose Computed Value
```xml
<computed>
    <int name="total" eval="ADD(input.a, input.b)" />
</computed>
<output>
    <int name="result" from="computed.total" />
</output>
```

#### Multiple Bindings
```xml
<computed>
    <int name="sum" eval="ADD(input.a, input.b)" />
    <int name="diff" eval="SUB(input.a, input.b)" />
</computed>
<output>
    <int name="sum" from="computed.sum" />
    <int name="difference" from="computed.diff" />
</output>
```
