# Skill Structure

This document describes the complete structure of a `SKILL.md` file following ASOS v1.0 with Gollum extensions.

## File Format

A skill file consists of two parts:

1. **YAML Frontmatter** (between `---` delimiters) - Metadata configuration
2. **Markdown Content** - The skill's system prompt

```
---
# YAML frontmatter here
---

# Markdown content here
```

## Frontmatter Fields

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Unique skill name (no spaces, use hyphens) |

### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `description` | string | "" | Brief description of the skill |
| `version` | string | "" | Semantic version (e.g., "1.0.0") |
| `type` | string | "agent" | Skill type: `agent`, `mcp`, or `workflow` |
| `tools` | []string | [] | List of allowed tools (empty = all) |
| `tool_scope` | string | "all" | Tool access: `all`, `read-only`, `none`, `custom` |
| `tool_filter` | []string | [] | Tools to explicitly exclude |
| `arguments` | string | "" | Argument specification for slash commands |
| `user_invocable` | bool | false | Can be invoked via slash command |
| `priority` | int | 0 | Execution priority (higher = more important) |
| `auto_invoke` | bool | false | Automatically invoke on matching context |
| `author` | string | "" | Skill author |
| `tags` | []string | [] | Tags for categorization |
| `category` | string | "" | Category for grouping |

## Field Details

### name (Required)

The skill name must:
- Be unique within the search paths
- Contain no spaces (use hyphens or underscores)
- Be case-insensitive for invocation

```yaml
name: code-reviewer        # Good
name: Code Reviewer        # Bad - contains spaces
```

### description

A brief, clear description of what the skill does. Used in skill listings.

```yaml
description: Reviews code for quality, security, and best practices
```

### type

Determines how the skill is executed:

- `agent` - Standard skill executed as a subagent (default)
- `mcp` - MCP server skill (connects to external tools)
- `workflow` - Orchestration skill for multi-step processes

```yaml
type: agent
```

### tools

Whitelist of tools the skill can use. If empty, all tools are allowed (subject to `tool_scope`).

```yaml
tools:
  - Read
  - Write
  - Edit
  - Glob
  - Grep
  - Bash
```

### tool_scope

Controls overall tool access:

| Value | Description |
|-------|-------------|
| `all` | Full access to all tools (default) |
| `read-only` | Only read operations (Read, Glob, Grep) |
| `none` | No tool access |
| `custom` | Use `tools` list for custom restrictions |

```yaml
tool_scope: read-only
```

### tool_filter

Blacklist of tools to exclude. Useful for removing dangerous operations.

```yaml
tool_filter:
  - Bash        # No shell access
  - Write       # No file creation
```

### arguments

Defines expected arguments for slash command invocation.

```yaml
arguments: "[file-path] [options]"
```

### user_invocable

When `true`, the skill can be invoked via slash command (`/skill-name`).

```yaml
user_invocable: true
```

### priority

Higher priority skills are presented first in listings. Range: 0-1000.

```yaml
priority: 100
```

### tags

Tags for categorization and filtering.

```yaml
tags:
  - code-quality
  - security
  - review
```

## Complete Example

```yaml
---
name: api-documenter
description: Generates API documentation from code
version: 1.2.0
type: agent
author: DevOps Team
category: documentation
tags:
  - api
  - documentation
  - openapi
user_invocable: true
priority: 50
tools:
  - Read
  - Glob
  - Grep
  - Write
tool_scope: custom
arguments: "[source-dir] [output-format]"
---

# API Documenter

You are an expert at generating comprehensive API documentation...

## Instructions

1. Scan the source directory for API endpoints
2. Extract route definitions, parameters, and responses
3. Generate documentation in the specified format
```

## Parsed Fields (Not in YAML)

These fields are populated during parsing, not from frontmatter:

| Field | Description |
|-------|-------------|
| `FilePath` | Path to the SKILL.md file |
| `Content` | Raw content after frontmatter |
| `FullContent` | Complete file including frontmatter |

## Validation Rules

1. `name` is required and must not contain spaces
2. `type` must be one of: `agent`, `mcp`, `workflow`
3. `tool_scope` must be one of: `all`, `read-only`, `none`, `custom`
4. All field names use snake_case
