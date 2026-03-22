# Step Result Notation Design

**Date:** 2026-03-22
**Status:** Approved

## Problem Statement

The flow XML currently uses `<output>` for two distinct concepts:

1. **Flow-Root Output** - defines the flow's public interface
2. **Step-Level Output** - captures step execution results

This causes confusion because the syntax is nearly identical:

```xml
<!-- Flow-Root Output -->
<output>
    <string name="verbose_result"/>
</output>

<!-- Step-Level Output -->
<output>
    <string path="stdout" assign="test_output" />
</output>
```

## Solution

Rename Step-Level Output from `<output>` to `<result>`.

## New Notation

### Step-Level Output (new)
```xml
<step type="shell" name="run-test">
    <cmd><![CDATA[go test ./...]]></cmd>
    <result>
        <string path="stdout" assignTo="test_output"/>
        <int path="exit_code" assignTo="exit_code"/>
    </result>
</step>
```

### Flow-Root Output (unchanged)
```xml
<flow name="my-flow">
    <output>
        <string name="test_status"/>
        <int name="exit_code"/>
    </output>
</flow>
```

## Attribute Changes

| Old | New | Notes |
|-----|-----|-------|
| `<output>` (step) | `<result>` | New wrapper element |
| `assign` | `assignTo` | More explicit |
| `name` (with path) | `path` | Indicates source path |
| `name` (flow-root) | `name` | Unchanged |

## Migration Impact

### Files to Update
- All flow XML files with step-level `<output>` blocks
- Parser/executor code that processes step outputs
- Documentation and examples

### Backward Compatibility
Consider supporting both `<output>` and `<result>` during transition period, then deprecate `<output>` for steps.

## Examples

### Shell Step
```xml
<step type="shell" name="gocyclo">
    <cmd><![CDATA[gocyclo -avg ${input.target_dir}]]></cmd>
    <result>
        <string path="stdout" assignTo="cyclomatic_output"/>
        <int path="exit_code" assignTo="exit_code"/>
    </result>
</step>
```

### API Call Step
```xml
<step type="api-call" name="fetch-pr">
    <url>https://api.github.com/pr/123</url>
    <result>
        <string path="title" assignTo="pr_title"/>
        <string path="state" assignTo="pr_state"/>
        <int path="changed_files" assignTo="file_count"/>
    </result>
</step>
```

### Sub-Flow Call
```xml
<step type="call-flow" name="analyze">
    <flow>code-analysis/main.xml</flow>
    <result>
        <int path="complexity_score" assignTo="score"/>
        <int path="security_issues" assignTo="issues"/>
    </result>
</step>
```

## Benefits

1. **Clarity** - `<result>` clearly indicates step return values
2. **Separation** - Distinct from flow-level output interface
3. **Explicit** - `assignTo` is more self-documenting than `assign`
4. **Consistent** - `path` attribute consistently indicates source location
