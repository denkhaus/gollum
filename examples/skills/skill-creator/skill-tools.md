# Tool Configuration

Skills can control which tools are available during execution. This is configured through `tools`, `tool_scope`, and `tool_filter` fields.

## Tool Scope

The `tool_scope` field provides high-level access control:

### all (Default)

Full access to all available tools.

```yaml
tool_scope: all
```

Use when:
- The skill needs full capabilities
- You trust the skill's behavior
- No security concerns exist

### read-only

Only read operations are permitted.

```yaml
tool_scope: read-only
```

Allowed tools:
- `Read` - Read file contents
- `Glob` - Find files by pattern
- `Grep` - Search file contents

Use when:
- The skill only analyzes code
- No modifications should be made
- Running in restricted environments

### none

No tool access at all.

```yaml
tool_scope: none
```

Use when:
- The skill is purely informational
- All data comes from the prompt
- Maximum isolation is required

### custom

Use the `tools` list for fine-grained control.

```yaml
tool_scope: custom
tools:
  - Read
  - Write
  - Edit
```

Use when:
- You need specific tool combinations
- Some tools should be excluded
- Custom access patterns are needed

## Tools Whitelist

The `tools` field specifies which tools are explicitly allowed:

```yaml
tools:
  - Read      # Read file contents
  - Write     # Create new files
  - Edit      # Modify existing files
  - Glob      # Find files by pattern
  - Grep      # Search file contents
  - Bash      # Execute shell commands
```

### Common Tool Combinations

#### Code Analysis

```yaml
tools:
  - Read
  - Glob
  - Grep
tool_scope: custom
```

#### Code Generation

```yaml
tools:
  - Read
  - Write
  - Edit
  - Glob
tool_scope: custom
```

#### Full Development

```yaml
tools:
  - Read
  - Write
  - Edit
  - Glob
  - Grep
  - Bash
tool_scope: custom
```

## Tool Filter

The `tool_filter` field excludes specific tools:

```yaml
tool_filter:
  - Bash      # No shell access
  - Write     # No file creation
```

This is useful when:
- Using `tool_scope: all` but excluding dangerous tools
- Building on existing configurations
- Implementing defense in depth

## Available Tools

| Tool | Category | Description |
|------|----------|-------------|
| `Read` | File | Read file contents |
| `Write` | File | Create new files |
| `Edit` | File | Modify existing files |
| `Glob` | Search | Find files by pattern |
| `Grep` | Search | Search file contents |
| `Bash` | System | Execute shell commands |
| `WebFetch` | Network | Fetch web content |
| `WebSearch` | Network | Search the web |

## Security Considerations

### Principle of Least Privilege

Grant only the tools needed for the skill's task:

```yaml
# Good: Minimal access for a review skill
name: code-reviewer
tool_scope: read-only

# Bad: Unnecessary access
name: code-reviewer
tool_scope: all
```

### Dangerous Tools

Some tools pose higher risk:

| Tool | Risk | Mitigation |
|------|------|------------|
| `Bash` | Command injection | Use `tool_filter` to exclude |
| `Write` | Data loss | Limit to specific directories |
| `Edit` | Unintended changes | Review before applying |

### Defense in Depth

Combine multiple controls:

```yaml
tool_scope: custom
tools:
  - Read
  - Glob
  - Grep
tool_filter:
  - Bash
  - Write
```

## Tool Checking

Skills can check tool availability programmatically:

```go
// Check if a tool is allowed
if skill.HasTool("Read") {
    // Tool is available
}

// Check if a tool is filtered
if skill.IsToolFiltered("Bash") {
    // Tool is not available
}
```

## Examples

### Read-Only Analysis Skill

```yaml
---
name: security-scanner
description: Scans code for security vulnerabilities
tool_scope: read-only
---

# Security Scanner

Scan the provided code for security issues...

### Selective Access Skill

```yaml
---
name: documentation-generator
description: Generates documentation from code
tool_scope: custom
tools:
  - Read
  - Write
  - Glob
  - Grep
tool_filter:
  - Bash
---

# Documentation Generator

Generate comprehensive documentation...
```

### Maximum Restriction Skill

```yaml
---
name: code-formatter
description: Formats code according to style guidelines
tool_scope: none
---

# Code Formatter

Format the provided code according to...
```

In this case, the skill receives code in the prompt and returns formatted code without any tool access.
