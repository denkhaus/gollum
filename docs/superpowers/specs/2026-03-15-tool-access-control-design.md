# Tool Access Control Design

**Date:** 2026-03-15
**Status:** Approved
**Author:** Design collaboration

## Overview

Remove the concept of "default tools" from the agent factory. Implement fine-grained control over which tools agents can access through explicit whitelisting.

## Problem Statement

Currently, the `AgentFactory` automatically adds ~15 default tools to ALL agents, ignoring what tools are specified in flow definitions. This is a security/design flaw because:

1. Flow definitions cannot control which tools LLMs have access to
2. Flows that specify `<tools>set_context_field</tools>` actually get access to ALL default tools
3. No unified way to control tool access for both flow LLM agents and spawned sub-agents

## Requirements

1. **Explicit opt-in**: All tools must be explicitly requested
2. **Unified interface**: Single mechanism for both built-in and MCP tools
3. **Type-safe validation**: Built-in tools validated against constants, MCP tools validated at runtime
4. **Format standardization**: Clear naming convention for tool types
5. **DI compliance**: New services follow project DI patterns

## Solution Design

### Core Architecture

**AgentConfig Changes:**
- Remove: `Tools []gollem.Tool`
- Remove: `ToolSets []gollem.ToolSet`
- Add: `AllowedTools []string`

**Tool Name Format:**
- Built-in tools: Use `ToolName` constants from `shared` package
  - Examples: `"bash"`, `"current_time"`, `"write_file"`
- MCP tools: `"server_name/tool_name"` format
  - Examples: `"filesystem/read_file"`, `"brave-search/web_search"`

**Behavior:**
- `AllowedTools` is nil or empty → No tools available (strictly opt-in)
- Invalid built-in tool → Error (fail agent creation)
- Missing MCP tool → Warning + skip (don't fail agent creation)

### Components

#### 1. MCPToolProvider (pkg/mcp)

**Interface:**
```go
type MCPToolProvider interface {
    CreateTool(agentID uuid.UUID, serverAndTool string) (gollem.Tool, error)
}
```

**Implementation:**
- Private `mcpToolProviderImpl`
- Constructor: `func NewMCPToolProvider(injector do.Injector) (MCPToolProvider, error)`
- Parses `"server/tool"` format
- Searches MCPRegistry for tool
- Wraps tool with agent context for logging

**DI Registration:**
```go
do.Provide(injector, mcp.NewMCPToolProvider)
```

#### 2. AgentFactory (pkg/agents)

**New Method:**
```go
func (f *defaultAgentFactory) resolveTools(
    ctx context.Context,
    agentID uuid.UUID,
    allowedTools []string,
) ([]gollem.Tool, error)
```

**Resolution Algorithm:**
1. For each tool name in `AllowedTools`:
   - If contains `/`: Use `MCPToolProvider`
     - Warn on missing, continue
   - Otherwise: Use built-in providers
     - Error on invalid name

2. Return resolved tool list

**Changes:**
- Remove lines 107-126 (default tools)
- Update `CreateAgent()` to call `resolveTools()`
- Inject `MCPToolProvider` via DI

#### 3. Flow Executor (pkg/flows/executor)

**llm_step.go changes:**
```go
// Parse tools from step <tools> tag
tools := strings.Split(step.Tools, ",")
config.AllowedTools = tools
```

**Flow XML example:**
```xml
<step type="llm" agent="startup">
    <tools>current_time,set_context_field,filesystem/read_file</tools>
</step>
```

#### 4. SpawnAgent Tool (pkg/tools)

**New parameter:**
```go
type SpawnAgentArgs struct {
    // ... existing fields
    AllowedTools []string `json:"allowed_tools,omitempty"`
}
```

**Tool spec update:**
```go
"allowed_tools": {
    Type:        gollem.TypeArray,
    Description: "List of tools the sub-agent can use",
    Items: &gollem.Parameter{Type: gollem.TypeString},
}
```

**Agent creation:**
```go
config := &shared.AgentConfig{
    AllowedTools: args.AllowedTools,
    // ... other fields
}
```

### Error Handling

| Scenario | Behavior |
|----------|----------|
| Invalid built-in tool name | Error: fail agent creation |
| Invalid MCP tool format (no "/") | Error: fail agent creation |
| MCP tool not found | Warning: log + skip tool |
| Empty AllowedTools array | No tools available |
| Nil AllowedTools | No tools available |
| Duplicate tool names | Deduplicate; last wins |

### Edge Cases

**Tool name collision:**
- Built-in: `"bash"`
- MCP: `"server/bash"`
- No collision possible due to "/" requirement

**MCP server not running:**
- Tool not found in registry
- Warning logged
- Agent creation succeeds

## Implementation Approach

**Use TDD** (Test-Driven Development) for all components.

### Phase 1: Core Infrastructure
1. Add `AllowedTools []string` to `AgentConfig`
2. Create `MCPToolProvider` in `pkg/mcp`
3. Update DI registration
4. Write tests first

### Phase 2: AgentFactory Refactor
1. Add `resolveTools()` method
2. Remove default tools logic (lines 107-126)
3. Update `CreateAgent()` to use `AllowedTools`
4. Write tests first

### Phase 3: Flow Integration
1. Update `llm_step.go` to populate `AllowedTools`
2. Remove `Tools`/`ToolSets` from `AgentConfig`
3. Write tests first

### Phase 4: SpawnAgent Tool
1. Add `allowed_tools` parameter
2. Update agent creation logic
3. Write tests first

### Phase 5: Tests & Migration
1. Update all affected tests
2. Add new test cases
3. Update default flow
4. Verify flows work correctly

## Testing Strategy

### Unit Tests

**AgentFactory:**
- Empty AllowedTools → agent with no tools
- Built-in tools only → resolves correctly
- MCP tools only → resolves with warnings
- Mixed built-in + MCP → both work
- Invalid built-in → error
- Invalid MCP format → error

**MCPToolProvider:**
- Valid `"server/tool"` → creates tool
- Invalid format (no "/") → error
- Tool not found → error
- Multiple tool sets → searches all

**SpawnAgent Tool:**
- With `allowed_tools` → sub-agent gets tools
- Without `allowed_tools` → sub-agent has no tools
- Invalid tool → error
- MCP tool format → resolves correctly

**Integration:**
- Flow with `<tools>` tag → LLM gets only those tools
- Flow without `<tools>` tag → LLM has no tools
- MCP tool in flow → works if server running

### Test Doubles

- Mock `MCPRegistry` for controlled tool availability
- Mock `LoggerService` for verifying warnings

## Migration

### Breaking Changes

1. **AgentConfig**: `Tools` and `ToolSets` fields removed
2. **Flows**: Must include `<tools>` tag or LLMs have no tools
3. **SpawnAgent**: Must explicitly pass `allowed_tools`

### Migration Steps

**1. Update default flow:**
```xml
<step type="llm" agent="startup">
    <tools>current_time,set_context_field</tools>
</step>
```

**2. Update flow linter:**
- Warn if LLM step has no `<tools>` tag
- Suggest adding explicit `<tools>` tag

**3. Update documentation:**
- Flow authoring guide
- Tool name format documentation
- List of available built-in tools

## Files to Modify

### New Files
- `pkg/mcp/tool_provider.go`
- `pkg/mcp/tool_provider_test.go`

### Modified Files
- `pkg/shared/agent_config.go`
- `pkg/agents/factory.go`
- `pkg/agents/factory_test.go`
- `pkg/flows/executor/llm_step.go`
- `pkg/tools/spawn_agent.go`
- `pkg/tools/spawn_agent_test.go`
- `.gollum/flows/default/main.xml`

### DI Updates
- `pkg/di/container.go` (or equivalent)

## Acceptance Criteria

- [ ] All agents require explicit `AllowedTools`
- [ ] Built-in tools validated against constants
- [ ] MCP tools validated with warnings for missing
- [ ] SpawnAgent supports `allowed_tools` parameter
- [ ] Default flow updated with explicit tools
- [ ] All tests pass (TDD approach)
- [ ] Flow linter warns about missing `<tools>` tag

## Future Considerations

- Tool groups/presets for common combinations
- Tool capability metadata (read-only vs write operations)
- Tool usage audit logging
- Fine-grained permissions within tools
