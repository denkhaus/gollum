# Computed Fields

Computed fields provide reactive, declarative computations that automatically update when their dependencies change.

## Syntax

Computed fields are defined at the flow level using the `<computed>` section:

```xml
<flow name="example" version="1.0">
    <input>
        <int name="value" />
    </input>

    <context>
        <int name="threshold" />
    </context>

    <computed>
        <bool name="is_large" eval="GT(input.value, context.threshold)" />
        <string name="status" eval="IF(input.value, GT(input.value, 10), 'large', 'small')" />
        <int name="doubled" eval="MUL(input.value, 2)" />
    </computed>

    <!-- ... states ... -->
</flow>
```

## Field Types

Computed fields support these types:
- `bool` - Boolean values
- `int` - Integer values
- `float` - Floating-point values
- `string` - String values

## Operators

### Comparison Operators
- `GT(a, b)` - Greater than
- `LT(a, b)` - Less than
- `GTE(a, b)` - Greater than or equal
- `LTE(a, b)` - Less than or equal
- `EQ(a, b)` - Equal
- `NEQ(a, b)` - Not equal

### Logical Operators
- `AND(a, b)` - Logical AND
- `OR(a, b)` - Logical OR
- `NOT(a)` - Logical NOT

### Arithmetic Operators
- `ADD(a, b)` - Addition
- `SUB(a, b)` - Subtraction
- `MUL(a, b)` - Multiplication
- `DIV(a, b)` - Division

### Conditional Operator
- `IF(condition, true_value, false_value)` - Ternary conditional

## Field References

Computed fields can reference fields from these scopes:

### Input Fields
```xml
<input>
    <int name="count" />
</input>

<computed>
    <bool name="has_count" eval="GT(input.count, 0)" />
</computed>
```

### Context Fields
```xml
<context>
    <int name="total" />
</context>

<computed>
    <bool name="is_complete" eval="EQ(context.total, 100)" />
</computed>
```

### Output Fields
```xml
<output>
    <int name="result" />
</output>

<computed>
    <bool name="has_result" eval="GT(output.result, 0)" />
</computed>
```

### Other Computed Fields
```xml
<computed>
    <int name="double" eval="MUL(input.value, 2)" />
    <int name="quadruple" eval="MUL(computed.double, 2)" />
</computed>
```

## Reactivity

Computed fields are **reactive** - they automatically re-evaluate when any dependency changes:

1. **Dependency Tracking**: The system tracks all field references in each expression
2. **Automatic Updates**: When a dependency changes, dependent computed fields are marked dirty
3. **Efficient Evaluation**: Dirty fields are re-evaluated on-demand when accessed
4. **No Cycles**: Circular dependencies are detected during flow validation

### Example: Reactive Chain

```xml
<computed>
    <int name="base" eval="input.value" />
    <int name="doubled" eval="MUL(computed.base, 2)" />
    <int name="quadrupled" eval="MUL(computed.doubled, 2)" />
</computed>
```

When `input.value` changes:
- `base` is marked dirty (depends on input.value)
- `doubled` is marked dirty (depends on computed.base)
- `quadrupled` is marked dirty (depends on computed.doubled)

All three fields re-evaluate in correct dependency order when next accessed.

## Validation Rules

The linter enforces these rules for computed fields:

### Required Attributes
- `name` - Field name (required, unique within computed section)
- `type` - Field type: bool, int, float, string (required)
- `eval` - Expression to evaluate (required)

### Field References
- All referenced fields must exist in input, context, output, or computed
- Self-references are not allowed
- Circular dependencies are not allowed

### Expression Syntax
- Must use valid operator format: `OPERATOR(arg1, arg2, ...)`
- Operator names must be uppercase
- Arguments must be valid field references or literals

## Examples

### Threshold Checking

```xml
<computed>
    <bool name="is_high_priority" eval="GT(context.priority, 5)" />
    <bool name="is_urgent" eval="AND(GT(context.priority, 8), input.flag)" />
</computed>
```

### String Selection

```xml
<computed>
    <string name="level" eval="IF(context.score, GT(context.score, 80), 'high', 'low')" />
</computed>
```

### Arithmetic

```xml
<computed>
    <int name="total" eval="ADD(input.a, input.b)" />
    <int name="average" eval="DIV(computed.total, 2)" />
    <int name="scaled" eval="MUL(computed.average, 100)" />
</computed>
```

### Complex Conditions

```xml
<computed>
    <bool name="should_process" eval="AND(
        GT(input.count, 0),
        LT(input.count, 100),
        EQ(context.status, 'ready')
    )" />
</computed>
```

## Migration from Context Computed Fields

Prior versions supported `<computed>` inside `<context>`. This is deprecated.

### Old Format (Deprecated)

```xml
<context>
    <computed name="is_valid" type="bool" when="GT(input.x, 10)" />
</context>
```

### New Format (Current)

```xml
<computed>
    <bool name="is_valid" eval="GT(input.x, 10)" />
</computed>
```

Use the `gollum migrate` command to automatically convert old flows to the new format.

## Best Practices

1. **Keep Expressions Simple**: Complex expressions are harder to debug
2. **Use Descriptive Names**: Make computed field names self-documenting
3. **Avoid Deep Chains**: More than 3 levels of computed dependencies can be confusing
4. **Document Intent**: Add comments explaining non-obvious computations
5. **Test Dependencies**: Use the linter to verify all references are valid
