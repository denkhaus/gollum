# Skill Types

Gollum supports three skill types, each designed for different use cases.

## Agent Type (Default)

The `agent` type is the most common skill type. It executes as a subagent with its own system prompt derived from the skill content.

### Use Cases

- Code review and analysis
- Document generation
- Data transformation
- Research and exploration
- Task automation

### Behavior

When invoked:
1. Creates a new subagent with the skill's content as system prompt
2. Inherits or isolates context based on `context_mode` parameter
3. Has access to tools based on `tools` and `tool_scope` configuration
4. Returns results to the parent agent

### Example

```yaml
---
name: code-reviewer
description: Reviews code for quality and best practices
type: agent
tools:
  - Read
  - Glob
  - Grep
---

# Code Reviewer

You are an expert code reviewer. Analyze the provided code for:

- Code quality and readability
- Potential bugs and edge cases
- Security vulnerabilities
- Performance considerations
- Adherence to best practices

Provide actionable feedback with specific line references.
```

## MCP Type

The `mcp` type connects to Model Context Protocol (MCP) servers, enabling integration with external tools and services.

### Use Cases

- Database queries
- API integrations
- External service access
- Custom tool implementations

### Behavior

When invoked:
1. Connects to the configured MCP server
2. Exposes MCP tools to the agent
3. Facilitates communication between agent and external services

### Example

```yaml
---
name: database-query
description: Query databases through MCP
type: mcp
---

# Database Query Skill

Connect to the database MCP server to execute queries...

**Note**: MCP skills require additional server configuration.
```

## Workflow Type

The `workflow` type orchestrates multi-step processes that may involve multiple agents or skills.

### Use Cases

- CI/CD pipelines
- Multi-stage transformations
- Complex decision trees
- Parallel task execution

### Behavior

When invoked:
1. Executes a defined workflow sequence
2. May invoke other skills or agents
3. Manages state between steps
4. Handles branching and error recovery

### Example

```yaml
---
name: release-pipeline
description: Orchestrates the release process
type: workflow
priority: 100
---

# Release Pipeline

Execute the release workflow:

1. **Validation Stage**
   - Run test suite
   - Check code coverage
   - Validate changelog

2. **Build Stage**
   - Compile binaries
   - Build containers
   - Generate artifacts

3. **Deployment Stage**
   - Deploy to staging
   - Run smoke tests
   - Deploy to production

4. **Notification Stage**
   - Update release notes
   - Notify stakeholders
```

## Type Comparison

| Feature | Agent | MCP | Workflow |
|---------|-------|-----|----------|
| Subagent creation | Yes | No | Yes |
| Tool access | Full | MCP tools | Full |
| Context inheritance | Yes | No | Yes |
| External services | Limited | Full | Limited |
| Multi-step orchestration | Manual | N/A | Built-in |
| Complexity | Low | Medium | High |

## Choosing a Type

### Use Agent When:
- You need a specialized assistant for a specific task
- The task involves file operations or code analysis
- You want context inheritance from the parent agent

### Use MCP When:
- You need to connect to external services
- You want to expose custom tools
- You need database or API access

### Use Workflow When:
- You have a multi-step process
- Steps have dependencies
- You need parallel execution
- Error handling and recovery are important

## Type Validation

The skill type is validated during parsing:

```go
type SkillType string

const (
    SkillTypeAgent    SkillType = "agent"
    SkillTypeMcp      SkillType = "mcp"
    SkillTypeWorkflow SkillType = "workflow"
)
```

Invalid types will cause a parse error.
