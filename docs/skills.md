# Gollum Skill System

Skills are reusable capability modules that extend Gollum agents with specialized knowledge and behaviors. This guide covers everything you need to know about creating, discovering, and invoking skills.

## Quick Start

### What is a Skill?

A skill is a `SKILL.md` file that defines:
- **Metadata** (YAML frontmatter): Name, description, tools, type, etc.
- **Content** (Markdown): The system prompt for the skill's execution

### Creating Your First Skill

1. Create a directory for your skill:
   ```bash
   mkdir -p skills/my-first-skill
   ```

2. Create a `SKILL.md` file:
   ```markdown
   ---
   name: my-first-skill
   description: My first Gollum skill
   version: 1.0.0
   type: agent
   ---

   # My First Skill

   You are a helpful assistant specialized in [domain].

   ## Instructions

   1. Analyze the input
   2. Process according to guidelines
   3. Return structured output
   ```

3. The skill will be automatically discovered when Gollum starts in that directory.

## Skill Discovery

Skills are automatically discovered in:

1. **Current workspace directory** - The directory Gollum is running in
2. **Workspace history** - Previously visited workspaces
3. **Configured search paths** - Paths added via `AddSearchPath()`

### Discovery Process

```go
// Skills are discovered on startup and can be refreshed
skillService.Discover(ctx)

// Add additional search paths
skillService.AddSearchPath("/path/to/skills")

// Refresh to find new skills
skillService.Refresh(ctx)
```

### File Structure

```
project/
├── .claude/
│   └── skills/
│       └── code-reviewer/
│           └── SKILL.md
├── skills/
│   ├── api-documenter/
│   │   └── SKILL.md
│   └── test-generator/
│       └── SKILL.md
└── examples/
    └── skills/
        └── skill-creator/
            ├── SKILL.md
            └── skill-structure.md
```

## Invoking Skills

### Using the invoke_skill Tool

```json
{
  "name": "skill-name",
  "input": "Your task description here",
  "context_mode": "inherited",
  "model": "sonnet"
}
```

### Parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| `name` | Yes | Skill name (case-insensitive) |
| `input` | Yes | Task description for the skill |
| `context_mode` | No | `inherited` or `isolated` (default: inherited) |
| `model` | No | `sonnet`, `opus`, or `haiku` |

### Context Modes

- **inherited**: Skill shares parent agent's conversation history
- **isolated**: Skill starts with fresh context

### Example Invocation

```json
{
  "name": "code-reviewer",
  "input": "Review the authentication module in src/auth/",
  "context_mode": "isolated"
}
```

## Skill File Format

### Frontmatter Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | Yes | - | Unique skill name (no spaces) |
| `description` | string | No | "" | Brief description |
| `version` | string | No | "" | Semantic version |
| `type` | string | No | agent | `agent`, `mcp`, or `workflow` |
| `tools` | []string | No | [] | Allowed tools |
| `tool_scope` | string | No | all | `all`, `read-only`, `none`, `custom` |
| `tool_filter` | []string | No | [] | Tools to exclude |
| `arguments` | string | No | "" | Slash command arguments |
| `user_invocable` | bool | No | false | Slash command availability |
| `priority` | int | No | 0 | Listing priority |
| `auto_invoke` | bool | No | false | Auto-invoke on context |
| `author` | string | No | "" | Author name |
| `tags` | []string | No | [] | Categorization tags |
| `category` | string | No | "" | Category name |

### Complete Example

```yaml
---
name: security-scanner
description: Scans code for security vulnerabilities
version: 1.2.0
type: agent
author: Security Team
category: security
tags:
  - security
  - vulnerability
  - audit
user_invocable: true
priority: 100
tools:
  - read_file
  - glob
  - grep
tool_scope: custom
---

# Security Scanner

You are a security expert. Scan the provided code for vulnerabilities.

## Check Categories

1. **Injection**: SQL, command, LDAP injection
2. **Authentication**: Weak passwords, session issues
3. **XSS**: Cross-site scripting vulnerabilities
4. **CSRF**: Cross-site request forgery
5. **Data Exposure**: Sensitive data leakage

## Output Format

Provide findings in this structure:
- Severity (Critical/High/Medium/Low)
- Location (file:line)
- Description
- Remediation
```

## Tool Configuration

### Tool Scopes

| Scope | Description | Allowed Tools |
|-------|-------------|---------------|
| `all` | Full access | All available tools |
| `read-only` | Read operations | read_file, glob, grep |
| `none` | No access | None |
| `custom` | Custom list | Per `tools` field |

### Tool Filtering

```yaml
# Whitelist approach
tools:
  - Read
  - Glob
tool_scope: custom

# Blacklist approach
tool_scope: all
tool_filter:
  - bash
  - write_file
```

## Hook Events

Skills trigger specific hook events during execution:

| Event | When Fired |
|-------|------------|
| `BeforeSkillInvoked` | Before skill execution starts |
| `AfterSkillInvoked` | After skill completes successfully |
| `OnSkillError` | When skill execution fails |

### Hook Context Data

```go
&hooks.HookContext{
    Data: map[string]any{
        "skill_name":    skill.Name,
        "skill_type":    string(skill.Type),
        "context_mode":  string(contextMode),
        "model":         string(llmProvider),
        "invoker_id":    senderID.String(),
        "skill_path":    skill.FilePath,
        "skill_version": skill.Version,
    },
}
```

## Best Practices

### Naming
- Use lowercase with hyphens: `code-reviewer`
- Be descriptive: `api-documenter` not `doc`
- Avoid generic names: `python-test-runner` not `test`

### Structure
- Start with clear role definition
- Use numbered steps for processes
- Include examples
- Document limitations

### Security
- Use minimal tool access
- Filter dangerous tools
- Validate inputs
- Handle errors gracefully

### Documentation
- Include usage examples
- Document expected inputs
- Specify output format
- List limitations

## Example Skills

See the `examples/skills/` directory for complete examples:

- **skill-creator**: Comprehensive guide for creating skills
  - `skill-structure.md`: SKILL.md format reference
  - `skill-types.md`: Agent, MCP, and workflow types
  - `skill-tools.md`: Tool configuration
  - `skill-context.md`: Context modes
  - `skill-best-practices.md`: Authoring guidelines
  - `templates.md`: Ready-to-use templates

## API Reference

For detailed API documentation, see [skills-api.md](skills-api.md).

## Troubleshooting

### Skill Not Found

1. Verify the skill file is named `SKILL.md` (case-sensitive)
2. Check the skill is in a search path
3. Run `skillService.Refresh(ctx)` to rediscover

### Parse Errors

1. Ensure frontmatter is between `---` delimiters
2. Validate YAML syntax
3. Check required `name` field exists

### Tool Access Issues

1. Verify `tool_scope` allows the tool
2. Check tool is not in `tool_filter`
3. Ensure tool is in `tools` list (for `custom` scope)
