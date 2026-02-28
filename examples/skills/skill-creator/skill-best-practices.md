# Skill Best Practices

This guide covers best practices for creating effective, maintainable, and secure skills.

## Naming Conventions

### Skill Names

- Use lowercase with hyphens: `code-reviewer`, not `CodeReviewer`
- Be descriptive but concise: `api-documenter`, not `doc`
- Avoid generic names: `python-test-runner`, not `test`
- Use consistent patterns across skills

```yaml
# Good
name: security-auditor
name: api-version-migrator
name: dockerfile-optimizer

# Bad
name: SecurityAuditor    # Wrong case
name: sa                 # Too abbreviated
name: do-stuff           # Too vague
```

### File Organization

```
skills/
├── code-reviewer/
│   ├── SKILL.md
│   └── templates/
│       └── review-template.md
├── api-documenter/
│   ├── SKILL.md
│   └── examples/
│       └── openapi-example.yaml
```

## Frontmatter Best Practices

### Always Include

```yaml
---
name: my-skill           # Required
description: Clear, actionable description
version: 1.0.0           # Track changes
author: Team Name        # Attribution
---
```

### Use Tags Effectively

```yaml
tags:
  - code-quality         # Category tag
  - security             # Domain tag
  - python               # Language tag
  - ci-cd                # Workflow tag
```

### Set Appropriate Priority

```yaml
# Critical skills that should appear first
priority: 100

# Standard skills
priority: 50

# Niche or specialized skills
priority: 10
```

## Content Structure

### Opening Statement

Start with a clear role definition:

```markdown
# Code Reviewer

You are an expert code reviewer specializing in Python applications.
Your task is to analyze code for quality, security, and maintainability.
```

### Organized Sections

Use consistent section structure:

```markdown
## Objective
What this skill accomplishes

## Instructions
Step-by-step guidance

## Guidelines
Rules and constraints

## Output Format
Expected response structure

## Examples
Sample inputs and outputs
```

### Clear Instructions

Use numbered steps for processes:

```markdown
## Instructions

1. **Analyze** the code structure and identify main components
2. **Review** each component for:
   - Code quality issues
   - Security vulnerabilities
   - Performance concerns
3. **Document** findings with specific line references
4. **Prioritize** issues by severity
5. **Recommend** actionable improvements
```

## Tool Configuration

### Principle of Least Privilege

```yaml
# Good: Only what's needed
tool_scope: read-only

# Avoid unless necessary
tool_scope: all
```

### Document Tool Requirements

```markdown
## Tool Requirements

This skill requires:
- **Read**: To access source files
- **Glob**: To find relevant files
- **Grep**: To search for patterns

It does NOT require:
- Write access (analysis only)
- Bash execution (no system commands)
```

## Security Considerations

### Validate Inputs

```markdown
## Input Validation

Before processing:
1. Verify the target path exists
2. Check file permissions
3. Validate file types
```

### Avoid Sensitive Operations

```yaml
# Exclude dangerous tools
tool_filter:
  - Bash
  - Write  # If not needed
```

### Handle Errors Gracefully

```markdown
## Error Handling

If you encounter:
- Missing files: Report clearly, suggest alternatives
- Permission errors: Explain the requirement
- Invalid input: Provide guidance on correct format
```

## Documentation

### Include Examples

```markdown
## Examples

### Input
Review the authentication module at src/auth/

### Output
## Authentication Module Review

### Summary
The authentication module implements JWT-based auth...

### Findings
1. **Security Issue** (Line 45): Token expiration not validated
2. **Code Quality** (Line 78): Missing error handling

### Recommendations
1. Add token expiration check
2. Implement comprehensive error handling
```

### Document Limitations

```markdown
## Limitations

- Only supports Python 3.8+
- Does not analyze binary files
- Requires network access for dependency checks
```

## Testing Skills

### Manual Testing

1. Invoke with various inputs
2. Test edge cases
3. Verify output format
4. Check error handling

### Test Cases to Cover

```markdown
## Test Scenarios

1. **Normal operation**: Standard input
2. **Empty input**: How does the skill handle no input?
3. **Invalid input**: Malformed or unexpected data
4. **Large input**: Performance with extensive data
5. **Edge cases**: Boundary conditions
```

## Versioning

### Semantic Versioning

```yaml
version: 1.2.3
# 1 = Major (breaking changes)
# 2 = Minor (new features)
# 3 = Patch (bug fixes)
```

### Changelog

Maintain a changelog in the skill or supporting files:

```markdown
## Changelog

### 1.2.0 (2024-01-15)
- Added support for async functions
- Improved error messages

### 1.1.0 (2024-01-10)
- Added Python 3.12 support

### 1.0.0 (2024-01-01)
- Initial release
```

## Performance

### Minimize Tool Calls

```markdown
## Efficiency Guidelines

- Batch file reads when possible
- Use specific glob patterns
- Limit search scope
```

### Use Appropriate Context Mode

```yaml
# For independent analysis
context_mode: isolated  # Faster, less token usage

# For context-aware tasks
context_mode: inherited  # When needed
```

## Maintenance

### Regular Reviews

- Check for outdated information
- Update tool requirements
- Refresh examples
- Address user feedback

### Deprecation Process

```markdown
## Deprecation Notice

This skill is deprecated as of version 2.0.0.
Please migrate to `new-skill-name` by 2024-06-01.

Migration guide: [link]
```

## Checklist

Before publishing a skill:

- [ ] Name follows conventions
- [ ] Description is clear and accurate
- [ ] Frontmatter is complete
- [ ] Tool access is minimal but sufficient
- [ ] Instructions are clear and actionable
- [ ] Examples are provided
- [ ] Error handling is documented
- [ ] Limitations are stated
- [ ] Version is set
- [ ] Tested with various inputs
