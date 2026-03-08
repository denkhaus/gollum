# Flow Module Resolution

## Overview

Flows can reference and call other flows using the `<call>` element. The parser automatically resolves these references to their actual file locations.

## Resolution Rules

### 1. Module Entry Points (`main.xml`)

Every module directory must have a `main.xml` file that serves as the entry point.

**Structure:**
```
.gollum/flows/
├── modules/
│   └── code-analysis/
│       ├── main.xml          ← Module entry point
│       ├── complexity-check.xml
│       └── security-scan.xml
```

**Calling a module:**
```xml
<!-- Calls modules/code-analysis/main.xml -->
<call ref="code-analysis">
    <input>
        <field name="target_dir" value="${repo_path}" />
    </input>
</call>
```

### 2. Sub-Flow References

Modules can contain multiple sub-flows that can be called directly.

**Calling a sub-flow:**
```xml
<!-- Calls modules/code-analysis/complexity-check.xml -->
<call ref="code-analysis/complexity-check">
    <input>
        <field name="target_dir" value="${repo_path}" />
    </input>
</call>
```

### 3. Same-Workflow Calls

Flows within the same workflow directory can reference each other directly.

**Structure:**
```
.gollum/flows/
└── forgejo-workflow/
    ├── main.xml
    ├── fetch-pr.xml
    ├── determine-phase.xml
    └── review-phase.xml
```

**Calling a sibling flow:**
```xml
<!-- Calls forgejo-workflow/fetch-pr.xml -->
<call ref="fetch-pr">
    <input>
        <field name="pr_number" value="${pr_number}" />
    </input>
</call>
```

### 4. Relative Path References

You can use relative paths for cross-workflow calls.

```xml
<!-- Calls ../other-workflow/flow.xml -->
<call ref="../other-workflow/analyze">
    <input>
        <field name="target" value="${target_path}" />
    </input>
</call>
```

## Resolution Priority

When resolving a `<call ref="...">`, the resolver tries in this order:

1. **Absolute paths** - If `ref` starts with `/`, use as-is
2. **Relative paths** - If `ref` starts with `./` or `../`, resolve relative to current flow's directory
3. **Sub-flow references** - If `ref` contains `/`, look in `modules/` directories
4. **Local directory** - Check current flow's directory for `<ref>.xml`
5. **Module entry points** - Look in `modules/<ref>/main.xml`

## Module Requirements

### Required Structure

A valid module must have:

1. **Directory**: `modules/<module-name>/`
2. **Entry Point**: `modules/<module-name>/main.xml`

### Example: code-analysis module

```
modules/code-analysis/
├── main.xml                ← Required: Entry point
├── complexity-check.xml    ← Optional: Sub-flow
└── security-scan.xml       ← Optional: Sub-flow
```

### Module Entry Point (`main.xml`)

The `main.xml` file:
- Can call other sub-flows within the module
- Defines the module's public interface (`<input>`, `<output>`)
- Coordinates the execution of sub-flows

**Example:**
```xml
<flow name="code-analysis" version="1.0">
    <input>
        <string name="target_dir" required="true" />
    </input>
    <output>
        <int name="complexity_score" />
        <int name="security_issues" />
    </output>

    <states>
        <state name="init" initial="true">
            <steps>
                <!-- Call sub-flows -->
                <call ref="complexity-check" timeout="120s">
                    <input>
                        <field name="target_dir" value="${target_dir}" />
                    </input>
                    <output>
                        <field name="score" value="${complexity_score}" />
                    </output>
                </call>
            </steps>
        </state>
    </states>
</flow>
```

## Validation

The linter validates that:

1. **Module references resolve correctly** - All `<call ref="...">` references can be resolved
2. **Modules have main.xml** - Direct module references (e.g., `ref="code-analysis"`) must point to `main.xml`
3. **Sub-flows don't need main.xml** - Sub-flow references (e.g., `ref="code-analysis/complexity-check"`) can point to any XML file

## Examples

### Example 1: Call Module Entry Point

```xml
<!-- In any flow -->
<call ref="code-analysis">
    <input>
        <field name="target_dir" value="${repo_path}" />
        <field name="scan_type" value="full" />
    </input>
    <output>
        <field name="complexity_score" value="${analysis.complexity}" />
    </output>
</call>
```

Resolves to: `.gollum/flows/modules/code-analysis/main.xml`

### Example 2: Call Sub-Flow

```xml
<!-- In any flow -->
<call ref="code-analysis/complexity-check">
    <input>
        <field name="target_dir" value="${repo_path}" />
    </input>
    <output>
        <field name="score" value="${complexity_score}" />
    </output>
</call>
```

Resolves to: `.gollum/flows/modules/code-analysis/complexity-check.xml`

### Example 3: Conditional Sub-Flow Calls

```xml
<call ref="code-analysis/security-scan" timeout="300s" when="is_deep_scan">
    <input>
        <field name="target_dir" value="${target_dir}" />
        <field name="scan_type" value="deep" />
    </input>
    <output>
        <field name="issues" value="${security_issues}" />
    </output>
</call>
```

Resolves to: `.gollum/flows/modules/code-analysis/security-scan.xml`

## Best Practices

1. **Use module entry points** - Call `main.xml` for module orchestration
2. **Use sub-flows for components** - Break complex logic into reusable sub-flows
3. **Document interfaces** - Always define `<input>` and `<output>` in `main.xml`
4. **Use clear naming** - Module and flow names should be descriptive
5. **Handle errors** - Add `<on-error>` handlers to calls for robustness
