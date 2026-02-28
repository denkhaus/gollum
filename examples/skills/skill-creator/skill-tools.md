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
- `read_file` - Read file contents
- `glob` - Find files by pattern
- `grep` - Search file contents

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
  - read_file
  - write_file
  - edit
```

Use when:
- You need specific tool combinations
- Some tools should be excluded
- Custom access patterns are needed

## Tools Whitelist

The `tools` field specifies which tools are explicitly allowed. Use Gollum tool names:

```yaml
tools:
  - read_file      # Read file contents
  - write_file     # Create new files
  - edit           # Modify existing files
  - glob           # Find files by pattern
  - grep           # Search file contents
  - bash           # Execute shell commands
```

### Common Tool Combinations

#### Code Analysis

```yaml
tools:
  - read_file
  - glob
  - grep
tool_scope: custom
```

#### Code Generation

```yaml
tools:
  - read_file
  - write_file
  - edit
  - glob
tool_scope: custom
```

#### Full Development

```yaml
tools:
  - read_file
  - write_file
  - edit
  - glob
  - grep
  - bash
tool_scope: custom
```

## Tool Filter

The `tool_filter` field excludes specific tools:

```yaml
tool_filter:
  - bash        # No shell access
  - write_file  # No file creation
```

This is useful when:
- Using `tool_scope: all` but excluding dangerous tools
- Building on existing configurations
- Implementing defense in depth

## Available Gollum Tools

| Tool | Category | Description |
|------|----------|-------------|
| `read_file` | File | Read file contents |
| `write_file` | File | Create new files |
| `edit` | File | Modify existing files |
| `glob` | Search | Find files by pattern |
| `grep` | Search | Search file contents |
| `bash` | System | Execute shell commands |
| `spawn_agent` | Agent | Create subagent |
| `resume_agent` | Agent | Resume subagent |
| `remove_agent` | Agent | Remove subagent |
| `list_agents` | Agent | List subagents |
| `agent_output` | Agent | Get agent output |
| `invoke_skill` | Skill | Invoke another skill |
| `change_directory` | Navigation | Change workspace |
| `current_time` | Utility | Get current time |
| `session_logs` | Utility | Get session logs |

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
| `bash` | Command injection | Use `tool_filter` to exclude |
| `write_file` | Data loss | Limit to specific directories |
| `edit` | Unintended changes | Review before applying |
| `spawn_agent` | Resource usage | Limit agent creation |

### Defense in Depth

Combine multiple controls:

```yaml
tool_scope: custom
tools:
  - read_file
  - glob
  - grep
tool_filter:
  - bash
  - write_file
```

## Tool Checking

Skills can check tool availability programmatically:

```go
// Check if a tool is allowed
if skill.HasTool("read_file") {
    // Tool is available
}

// Check if a tool is filtered
if skill.IsToolFiltered("bash") {
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
```

### Selective Access Skill

```yaml
---
name: documentation-generator
description: Generates documentation from code
tool_scope: custom
tools:
  - read_file
  - write_file
  - glob
  - grep
tool_filter:
  - bash
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
