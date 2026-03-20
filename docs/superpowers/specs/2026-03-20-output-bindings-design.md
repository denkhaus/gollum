# Output Bindings Design

**Date:** 2026-03-20
**Status:** Approved
**Author:** AI Agent

## Overview

Add optional `from` attribute to output fields for declarative value binding, eliminating the verbose `assign` function while maintaining backward compatibility.

## Problem Statement

The `assign` function is:
1. **Verbose** - 6 lines of XML per assignment
2. **Repetitive** - Multiple fields require identical boilerplate
3. **Conceptually mismatched** - Assignment is fundamental, not a "function call"

**Example verbosity:**
```xml
<step type="func" function="assign">
    <params>
        <param name="from" value="computed.sum" />
        <param name="to" value="output.sum" />
    </params>
</step>
```

## Design Goals

1. **Reduce verbosity** - 6 lines → 1 line per assignment
2. **Declarative over imperative** - Data flow visible at definition level
3. **Type-safe** - Validate at parse time
4. **No magic** - Clear, predictable data flow
5. **Backward compatible** - Existing flows continue to work

## Solution: Output Field Bindings

Output fields can optionally declare their source using a `from` attribute.

### Declarative Outputs (with `from`)

```xml
<output>
    <int name="sum" from="computed.sum" />
    <string name="message" from="input.text" />
    <bool name="is_valid" from="computed.is_valid" />
</output>
```

**Properties:**
- **Readonly at runtime** - Cannot be set by tools/steps
- **Auto-populated** - Value flows after source evaluation
- **Type-safe** - Linter validates source exists

### Imperative Outputs (without `from`)

```xml
<output>
    <string name="result" />
    <int name="count" />
</output>
```

**Properties:**
- **Mutable at runtime** - Set by tools/steps
- **Default behavior** - Traditional flow execution

### Architecture

```
                    ┌─────────────┐
                    │    Input    │
                    └──────┬──────┘
                           │
                           ▼
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│  Computed   │◄─────│   Context   │      │   Output    │
│  (reactive) │      │  (mutable)  │      └──────┬──────┘
└──────┬──────┘      └─────────────┘             │
       │                                        ▲
       │                                        │
       └─────────────── "from" attributes ───────┘
                      (declarative, readonly)

                       Steps can also write via set_output_value
                       (imperative, mutable - but NOT if "from" exists)
```

## Schema Changes

### Output Field Type

```xml
<xs:complexType name="OutputField">
    <xs:attribute name="name" type="xs:string" use="required"/>
    <xs:attribute name="from" type="xs:string"/>  <!-- NEW: optional source -->
</xs:complexType>
```

### Go Types

```go
type OutputField struct {
    XMLName xml.Name `xml:"string,int,bool,float,object"`
    Name    string   `xml:"name,attr"`
    From    string   `xml:"from,attr,omitempty"`  // NEW
}

type OutputValues struct {
    declarative map[string]OutputBinding  // readonly, auto-populated
    imperative  *MutableOutputFields      // mutable, set by tools
    defs        map[string]OutputField
}

type OutputBinding struct {
    TargetName  string
    SourceScope string  // "input", "context", "computed"
    SourceName  string
    Value       FieldValue
}
```

## Conflict Prevention

| Output Field Type | Has `from`? | Settable by `set_output_value`? |
|-------------------|-------------|----------------------------------|
| Declarative       | Yes         | **NO** - Runtime error           |
| Imperative        | No          | Yes                              |

**Runtime error:**
```
OutputFieldReadOnlyError: field 'sum' has a 'from' attribute and cannot be set at runtime
  └─ value flows from: computed.sum
```

## Linter Validation

### Error Codes

| Code | Message | Severity |
|------|---------|----------|
| E005 | Invalid 'from' reference format | Error |
| E006 | Invalid scope in 'from' attribute | Error |
| E007 | Referenced field does not exist | Error |
| E008 | Circular dependency in output bindings | Error |
| W003 | Multiple outputs from same source | Warning |

### Validation Rules

1. `from` format must be `prefix.fieldname`
2. Valid scopes: `input`, `context`, `computed`, `output`
3. Referenced field must exist
4. No cycles in output→output references
5. At most one output per unique source (warning for duplicates)

## Migration

### Before: arithmetic-computed.xml (66 lines)

```xml
<flow name="arithmetic-computed" version="1.0">
    <input>
        <int name="a" default="10" />
        <int name="b" default="5" />
    </input>

    <output>
        <int name="sum" />
        <int name="difference" />
        <int name="product" />
        <int name="quotient" />
    </output>

    <computed>
        <int name="sum" eval="ADD(input.a, input.b)" />
        <int name="difference" eval="SUB(input.a, input.b)" />
        <int name="product" eval="MUL(input.a, input.b)" />
        <int name="quotient" eval="DIV(input.a, input.b)" />
    </computed>

    <states>
        <state name="assign_outputs" initial="true">
            <steps>
                <step type="func" function="assign">
                    <params><param name="from" value="computed.sum" /><param name="to" value="output.sum" /></params>
                </step>
                <!-- repeated 4 times -->
            </steps>
            <transitions><transition to="done" /></transitions>
        </state>
        <state name="done" />
    </states>
</flow>
```

### After: arithmetic-computed.xml (35 lines)

```xml
<flow name="arithmetic-computed" version="1.0">
    <input>
        <int name="a" default="10" />
        <int name="b" default="5" />
    </input>

    <computed>
        <int name="sum" eval="ADD(input.a, input.b)" />
        <int name="difference" eval="SUB(input.a, input.b)" />
        <int name="product" eval="MUL(input.a, input.b)" />
        <int name="quotient" eval="DIV(input.a, input.b)" />
    </computed>

    <output>
        <int name="sum" from="computed.sum" />
        <int name="difference" from="computed.difference" />
        <int name="product" from="computed.product" />
        <int name="quotient" from="computed.quotient" />
    </output>

    <states>
        <state name="done" initial="true" />
    </states>
</flow>
```

**Result:** 47% reduction in lines, eliminates entire state just for assignment.

## Implementation Phases

### Phase 1: Schema (Non-Breaking)
- Add `from` attribute to `OutputField` type
- Update XSD schema
- Add linter validation
- Keep `assign` function working

### Phase 2: Executor
- Implement `OutputValues` declarative/imperative split
- Add `OutputFieldReadOnlyError`
- Update `ExecutionContext` for binding initialization
- Implement TDD tests

### Phase 3: Flow Migration
- Refactor all 9 flows using `assign`
- Update example flows
- Add deprecation warning for `assign`

### Phase 4: Deprecation
- Mark `assign` as deprecated
- Update documentation
- Provide migration guide

### Phase 5: Removal (Future)
- Remove `assign` function
- Clean up remaining references

## Flows to Migrate

```
.gollum/flows/examples/arithmetic-computed.xml
.gollum/flows/examples/assign-input-to-output.xml
.gollum/flows/examples/step-types-example.xml
.gollum/flows/examples/nested-context-example.xml
.gollum/flows/modules/code-analysis/complexity-check.xml
.gollum/flows/modules/code-analysis/security-scan.xml
.gollum/flows/forgejo-workflow/fetch-pr.xml
.gollum/flows/forgejo-workflow/review-phase.xml
test/fixtures/flows/func_step_test.xml
```

## Testing Strategy (TDD)

### Unit Tests
- `output_test.go` - declarative/imperative split
- `execution_context_test.go` - binding initialization
- `linter/output_test.go` - validation rules

### Integration Tests
- Complete flow execution with declarative outputs
- Mixed declarative/imperative outputs
- Error cases (readonly violation)

## Success Criteria

- [ ] Schema supports `from` attribute
- [ ] Declarative outputs auto-populated
- [ ] Readonly enforcement at runtime
- [ ] Linter validates all rules
- [ ] All tests pass (TDD)
- [ ] All 9 flows migrated
- [ ] Documentation updated
- [ ] Backward compatible
- [ ] `assign` deprecated

## Benefits

1. **90% less verbose** - 6 lines → 1 line per assignment
2. **Declarative** - Data flow at definition level
3. **Type-safe** - Parse-time validation
4. **No magic** - Clear, predictable
5. **Eliminates boilerplate states** - Cleaner flows
