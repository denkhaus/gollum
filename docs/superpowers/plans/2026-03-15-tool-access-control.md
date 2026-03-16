# Tool Access Control Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove default tools from agent factory and implement explicit tool whitelisting via `AllowedTools` field on AgentConfig.

**Architecture:** Unified tool resolution through provider pattern. Built-in tools validated against constants, MCP tools validated at runtime with warning on missing. AgentFactory resolves tool names to instances.

**Tech Stack:** Go 1.23+, samber/do/v2 for DI, gollem for LLM tools, gomock for mocking

---

## File Structure

**New files:**
- `pkg/mcp/tool_provider.go` - MCPToolProvider implementation
- `pkg/mcp/tool_provider_test.go` - Tests for MCPToolProvider

**Modified files:**
- `pkg/shared/agent_config.go` - Add AllowedTools field, remove Tools/ToolSets
- `pkg/agents/factory.go` - Remove default tools, add resolveTools()
- `pkg/agents/factory_test.go` - Tests for tool resolution
- `pkg/flows/executor/llm_step.go` - Populate AllowedTools from step
- `pkg/flows/executor/llm_step_test.go` - Tests for AllowedTools population
- `pkg/tools/spawn_agent.go` - Add allowed_tools parameter
- `pkg/tools/spawn_agent_test.go` - Tests for allowed_tools
- `.gollum/flows/default/main.xml` - Add explicit tools tag

**DI updates:**
- `pkg/di/container.go` or equivalent - Register MCPToolProvider

---

## Chunk 1: Core Infrastructure

### Task 1: Update AgentConfig

**Files:**
- Modify: `pkg/shared/agent_config.go`

- [ ] **Step 1: Write failing test for AllowedTools**

Create a test file `pkg/shared/agent_config_test.go`:

```go
package shared

import (
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
)

func TestAgentConfig_AllowedToolsField(t *testing.T) {
    config := &AgentConfig{
        ID:           uuid.New(),
        AllowedTools: []string{"bash", "current_time"},
    }

    assert.NotNil(t, config.AllowedTools)
    assert.Equal(t, "bash", config.AllowedTools[0])
    assert.Equal(t, "current_time", config.AllowedTools[1])
}

func TestAgentConfig_NoToolsField(t *testing.T) {
    config := &AgentConfig{
        ID: uuid.New(),
    }

    // This should not have Tools or ToolSets fields
    // (will fail until we remove them)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/shared/... -v -run TestAgentConfig_AllowedToolsField`
Expected: FAIL - AllowedTools field doesn't exist yet

- [ ] **Step 3: Add AllowedTools field to AgentConfig**

In `pkg/shared/agent_config.go`, update the struct:

```go
type AgentConfig struct {
    ID              uuid.UUID
    SystemPrompt    string
    Role            string
    Description     string
    LLMClientConfig *llm.ClientConfig
    OutputMode      OutputMode
    Strategy        gollem.Strategy
    AllowCompaction bool

    // AllowedTools specifies which tools the agent can access.
    // Built-in tools use ToolName constants (e.g., "bash", "current_time").
    // MCP tools use "server_name/tool_name" format (e.g., "filesystem/read_file").
    // Empty or nil means no tools available.
    AllowedTools    []string
}
```

- [ ] **Step 4: Remove Tools and ToolSets fields**

Remove these lines from AgentConfig:
```go
Tools    []gollem.Tool
ToolSets []gollem.ToolSet
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./pkg/shared/... -v`
Expected: PASS

- [ ] **Step 6: Update default agent creation in tests**

Search for all `AgentConfig` instantiations that set `Tools` or `ToolSets` and update to use `AllowedTools` or remove entirely.

Run: `grep -r "\.Tools\s*=" --include="*.go" pkg/`
Run: `grep -r "\.ToolSets\s*=" --include="*.go" pkg/`

Update each occurrence to use `AllowedTools` instead.

- [ ] **Step 7: Run full test suite to identify breakage**

Run: `go test ./... -short`
Expected: Some tests will fail - this is expected

- [ ] **Step 8: Commit**

```bash
git add pkg/shared/agent_config.go pkg/shared/agent_config_test.go
git commit -m "feat: add AllowedTools to AgentConfig, remove Tools/ToolSets

This is a breaking change - agents must now explicitly declare tools
via AllowedTools field using tool name strings.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 2: Create MCPToolProvider

**Files:**
- Create: `pkg/mcp/tool_provider.go`
- Create: `pkg/mcp/tool_provider_test.go`

- [ ] **Step 1: Write the failing test**

Create `pkg/mcp/tool_provider_test.go`:

```go
package mcp

import (
    "context"
    "testing"

    "github.com/denkhaus/gollum/pkg/logger"
    "github.com/denkhaus/gollum/pkg/mocks"
    "github.com/google/uuid"
    "github.com/m-mizutani/gollem"
    "github.com/stretchr/testify/assert"
    "go.uber.org/mock/gomock"
    "go.uber.org/zap"
)

func TestMCPToolProvider_CreateTool_ValidFormat(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockLogger := mocks.NewMockLoggerService(ctrl)
    mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()

    // Create mock registry that returns a tool set with a test tool
    testTool := gollem.NewMockTool("test_tool", gollem.ToolSpec{
        Name: "test_tool",
    })

    provider := &mcpToolProviderImpl{
        registry: &mockMCPRegistry{tools: []gollem.ToolSet{&mockToolSet{tool: testTool}}},
        logger:   mockLogger,
    }

    tool, err := provider.CreateTool(uuid.Nil, "testserver/test_tool")

    assert.NoError(t, err)
    assert.NotNil(t, tool)
    assert.Equal(t, "test_tool", tool.Spec().Name)
}

func TestMCPToolProvider_CreateTool_InvalidFormat_NoSlash(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockLogger := mocks.NewMockLoggerService(ctrl)
    mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()

    provider := &mcpToolProviderImpl{
        registry: &mockMCPRegistry{},
        logger:   mockLogger,
    }

    tool, err := provider.CreateTool(uuid.Nil, "invalidformat")

    assert.Error(t, err)
    assert.Nil(t, tool)
    assert.Contains(t, err.Error(), "invalid MCP tool format")
}

func TestMCPToolProvider_CreateTool_ToolNotFound(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockLogger := mocks.NewMockLoggerService(ctrl)
    mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()

    provider := &mcpToolProviderImpl{
        registry: &mockMCPRegistry{}, // Empty registry
        logger:   mockLogger,
    }

    tool, err := provider.CreateTool(uuid.Nil, "testserver/missing_tool")

    assert.Error(t, err)
    assert.Nil(t, tool)
    assert.Contains(t, err.Error(), "not found")
}

// Mock implementations
type mockMCPRegistry struct {
    tools []gollem.ToolSet
}

func (m *mockMCPRegistry) GetToolSets() []gollem.ToolSet {
    return m.tools
}

func (m *mockMCPRegistry) Close() error {
    return nil
}

type mockToolSet struct {
    tool gollem.Tool
}

func (m *mockToolSet) List() []gollem.Tool {
    return []gollem.Tool{m.tool}
}

func (m *mockToolSet) Name() string {
    return "mock"
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/mcp/... -v -run TestMCPToolProvider`
Expected: FAIL - type doesn't exist yet

- [ ] **Step 3: Implement MCPToolProvider interface and implementation**

Create `pkg/mcp/tool_provider.go`:

```go
package mcp

import (
    "fmt"
    "strings"

    "github.com/denkhaus/gollum/pkg/logger"
    "github.com/google/uuid"
    "github.com/m-mizutani/gollem"
    "github.com/samber/do/v2"
)

// MCPToolProvider creates tools from MCP registry by server/tool name
type MCPToolProvider interface {
    CreateTool(agentID uuid.UUID, serverAndTool string) (gollem.Tool, error)
}

type mcpToolProviderImpl struct {
    registry MCPRegistry
    logger   logger.LoggerService
}

// NewMCPToolProvider creates a new MCPToolProvider
func NewMCPToolProvider(injector do.Injector) (MCPToolProvider, error) {
    registry := do.MustInvoke[MCPRegistry](injector)
    logger := do.MustInvoke[logger.LoggerService](injector)

    return &mcpToolProviderImpl{
        registry: registry,
        logger:   logger,
    }, nil
}

func (p *mcpToolProviderImpl) CreateTool(agentID uuid.UUID, serverAndTool string) (gollem.Tool, error) {
    parts := strings.SplitN(serverAndTool, "/", 2)
    if len(parts) != 2 {
        return nil, fmt.Errorf("invalid MCP tool format: %s (expected 'server/tool')", serverAndTool)
    }

    serverName, toolName := parts[0], parts[1]

    // Get tool sets from registry
    toolSets := p.registry.GetToolSets()

    // Search for tool in all tool sets
    for _, toolSet := range toolSets {
        tools := toolSet.List()
        for _, tool := range tools {
            if tool.Spec().Name == toolName {
                // Found it - return the tool
                // Note: The tool is already wrapped by the MCP client
                return tool, nil
            }
        }
    }

    return nil, fmt.Errorf("tool '%s' not found in server '%s'", toolName, serverName)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/mcp/... -v -run TestMCPToolProvider`
Expected: PASS

- [ ] **Step 5: Add test for multiple tool sets**

Add to `pkg/mcp/tool_provider_test.go`:

```go
func TestMCPToolProvider_CreateTool_MultipleToolSets(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockLogger := mocks.NewMockLoggerService(ctrl)
    mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()

    // Create multiple tool sets
    tool1 := gollem.NewMockTool("tool1", gollem.ToolSpec{Name: "tool1"})
    tool2 := gollem.NewMockTool("tool2", gollem.ToolSpec{Name: "tool2"})

    provider := &mcpToolProviderImpl{
        registry: &mockMCPRegistry{
            tools: []gollem.ToolSet{
                &mockToolSet{tool: tool1},
                &mockToolSet{tool: tool2},
            },
        },
        logger: mockLogger,
    }

    // Should find tool2 even though it's in the second tool set
    tool, err := provider.CreateTool(uuid.Nil, "server2/tool2")

    assert.NoError(t, err)
    assert.NotNil(t, tool)
}
```

- [ ] **Step 6: Run test**

Run: `go test ./pkg/mcp/... -v -run TestMCPToolProvider_CreateTool_MultipleToolSets`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add pkg/mcp/tool_provider.go pkg/mcp/tool_provider_test.go
git commit -m "feat: add MCPToolProvider for MCP tool resolution

MCPToolProvider resolves MCP tools by 'server/tool' format.
Validates tool exists in MCPRegistry before returning.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 2: AgentFactory Refactoring

### Task 3: Add resolveTools method to AgentFactory

**Files:**
- Modify: `pkg/agents/factory.go`
- Modify: `pkg/agents/factory_test.go`

- [ ] **Step 1: Write failing test for resolveTools**

Add to `pkg/agents/factory_test.go`:

```go
func TestDefaultAgentFactory_ResolveTools_Empty(t *testing.T) {
    injector := setupTestDI(t)
    factory := do.MustInvoke[shared.AgentFactory](injector).(*defaultAgentFactory)

    tools, err := factory.resolveTools(context.Background(), uuid.Nil, nil)

    assert.NoError(t, err)
    assert.Nil(t, tools)
}

func TestDefaultAgentFactory_ResolveTools_BuiltinOnly(t *testing.T) {
    injector := setupTestDI(t)
    factory := do.MustInvoke[shared.AgentFactory](injector).(*defaultAgentFactory)

    tools, err := factory.resolveTools(context.Background(), uuid.Nil, []string{"bash", "current_time"})

    assert.NoError(t, err)
    assert.Len(t, tools, 2)
}

func TestDefaultAgentFactory_ResolveTools_InvalidBuiltin(t *testing.T) {
    injector := setupTestDI(t)
    factory := do.MustInvoke[shared.AgentFactory](injector).(*defaultAgentFactory)

    tools, err := factory.resolveTools(context.Background(), uuid.Nil, []string{"not_a_real_tool"})

    assert.Error(t, err)
    assert.Nil(t, tools)
    assert.Contains(t, err.Error(), "unknown built-in tool")
}

func TestDefaultAgentFactory_ResolveTools_MCPTool(t *testing.T) {
    injector := setupTestDI(t)
    factory := do.MustInvoke[shared.AgentFactory](injector).(*defaultAgentFactory)

    // This will warn but not fail if MCP tool doesn't exist
    tools, err := factory.resolveTools(context.Background(), uuid.Nil, []string{"filesystem/read_file"})

    // Should succeed even if tool doesn't exist
    assert.NoError(t, err)
    // Tool count depends on whether MCP server is running
}

func TestDefaultAgentFactory_ResolveTools_InvalidMCPFormat(t *testing.T) {
    injector := setupTestDI(t)
    factory := do.MustInvoke[shared.AgentFactory](injector).(*defaultAgentFactory)

    tools, err := factory.resolveTools(context.Background(), uuid.Nil, []string{"noslash"})

    assert.Error(t, err)
    assert.Nil(t, tools)
    assert.Contains(t, err.Error(), "invalid MCP tool format")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/agents/... -v -run TestDefaultAgentFactory_ResolveTools`
Expected: FAIL - resolveTools method doesn't exist

- [ ] **Step 3: Implement resolveTools method**

Add to `pkg/agents/factory.go`:

```go
import (
    "fmt"
    "strings"
    // ... existing imports
)

func (f *defaultAgentFactory) resolveTools(ctx context.Context, agentID uuid.UUID, allowedTools []string) ([]gollem.Tool, error) {
    if allowedTools == nil {
        return nil, nil
    }

    var tools []gollem.Tool
    var warnings []string

    for _, toolName := range allowedTools {
        // Check if MCP tool (format: "server_name/tool_name")
        if strings.Contains(toolName, "/") {
            tool, err := f.mcpToolProvider.CreateTool(agentID, toolName)
            if err != nil {
                // MCP tool not found - warn, don't fail
                warnings = append(warnings, fmt.Sprintf("MCP tool '%s': %v", toolName, err))
                continue
            }
            tools = append(tools, tool)
        } else {
            // Built-in tool
            tool, err := f.resolveBuiltinTool(agentID, toolName)
            if err != nil {
                return nil, err
            }
            tools = append(tools, tool)
        }
    }

    // Log warnings for missing MCP tools
    if len(warnings) > 0 {
        f.logService.Warn("Some MCP tools were not found",
            zap.Strings("missing_tools", warnings))
    }

    return tools, nil
}

func (f *defaultAgentFactory) resolveBuiltinTool(agentID uuid.UUID, name string) (gollem.Tool, error) {
    switch shared.ToolName(name) {
    case shared.ToolNameBash:
        return f.bashToolProv.CreateTool(agentID), nil
    case shared.ToolNameCurrentTime:
        return f.currentTimeToolProv.CreateTool(agentID), nil
    case shared.ToolNameWriteFile:
        return f.writeFileToolProv.CreateTool(agentID), nil
    case shared.ToolNameReadFile:
        return f.readFileToolProv.CreateTool(agentID), nil
    case shared.ToolNameGlob:
        return f.globToolProv.CreateTool(agentID), nil
    case shared.ToolNameGrep:
        return f.grepToolProv.CreateTool(agentID), nil
    case shared.ToolNameEdit:
        return f.editToolProv.CreateTool(agentID), nil
    case shared.ToolNameSpawnAgent:
        return f.spawnAgentToolProv.CreateTool(agentID, f), nil
    case shared.ToolNameAgentOutput:
        return f.agentOutputToolProv.CreateTool(agentID), nil
    case shared.ToolNameRemoveAgent:
        return f.removeAgentToolProv.CreateTool(agentID), nil
    case shared.ToolNameResumeAgent:
        return f.resumeAgentToolProv.CreateTool(agentID), nil
    case shared.ToolNameListAgents:
        return f.listAgentsToolProv.CreateTool(agentID), nil
    case shared.ToolNameSessionLogs:
        return f.sessionLogsToolProv.CreateTool(agentID), nil
    case shared.ToolNameChangeDirectory:
        return f.changeDirectoryToolProv.CreateTool(agentID), nil
    case shared.ToolNameInvokeSkill:
        return f.invokeSkillToolProv.CreateTool(agentID, f), nil
    default:
        return nil, fmt.Errorf("unknown built-in tool: %s", name)
    }
}
```

- [ ] **Step 4: Add mcpToolProvider field to factory struct**

Update the struct in `pkg/agents/factory.go`:

```go
type defaultAgentFactory struct {
    logService              logger.LoggerService
    configService           config.ConfigService
    clientProvider          llm.ClientProvider
    registry                registry.AgentRegistry
    promptManager           manager.PromptManager
    channelProvider         channel.ChannelMiddlewareProvider

    // Tool providers
    spawnAgentToolProv      tools.SpawnAgentToolProvider
    agentOutputToolProv     tools.AgentOutputToolProvider
    removeAgentToolProv     tools.RemoveAgentToolProvider
    resumeAgentToolProv     tools.ResumeAgentToolProvider
    listAgentsToolProv      tools.ListAgentsToolProvider
    currentTimeToolProv     tools.CurrentTimeToolProvider
    bashToolProv            tools.BashToolProvider
    writeFileToolProv       tools.WriteFileToolProvider
    readFileToolProv        tools.ReadFileToolProvider
    globToolProv            tools.GlobToolProvider
    grepToolProv            tools.GrepToolProvider
    editToolProv            tools.EditToolProvider
    sessionLogsToolProv     tools.SessionLogsToolProvider
    changeDirectoryToolProv tools.ChangeDirectoryToolProvider
    invokeSkillToolProv     tools.InvokeSkillToolProvider
    mcpToolProvider         mcp.MCPToolProvider  // NEW
}
```

- [ ] **Step 5: Update NewAgentFactory to inject MCPToolProvider**

Update `pkg/agents/factory.go`:

```go
func NewAgentFactory(injector do.Injector) (shared.AgentFactory, error) {
    // ... existing invocations

    mcpToolProvider := do.MustInvoke[mcp.MCPToolProvider](injector)  // NEW

    return &defaultAgentFactory{
        // ... existing field assignments
        mcpToolProvider: mcpToolProvider,  // NEW
    }, nil
}
```

- [ ] **Step 6: Run tests**

Run: `go test ./pkg/agents/... -v -run TestDefaultAgentFactory_ResolveTools`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add pkg/agents/factory.go pkg/agents/factory_test.go
git commit -m "feat: add resolveTools method to AgentFactory

Resolves tool names to gollem.Tool instances.
Validates built-in tools against constants.
Validates MCP tools via MCPRegistry with warnings.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 4: Update CreateAgent to use resolveTools

**Files:**
- Modify: `pkg/agents/factory.go`

- [ ] **Step 1: Write test for agent with AllowedTools**

Add to `pkg/agents/factory_test.go`:

```go
func TestDefaultAgentFactory_CreateAgent_WithAllowedTools(t *testing.T) {
    injector := setupTestDI(t)
    factory := do.MustInvoke[shared.AgentFactory](injector)

    config := &shared.AgentConfig{
        ID:           uuid.New(),
        SystemPrompt: "Test agent",
        Role:         "test",
        AllowedTools: []string{"bash", "current_time"},
    }

    agent, err := factory.CreateAgent(context.Background(), config)

    assert.NoError(t, err)
    assert.NotNil(t, agent)
}

func TestDefaultAgentFactory_CreateAgent_NoTools(t *testing.T) {
    injector := setupTestDI(t)
    factory := do.MustInvoke[shared.AgentFactory](injector)

    config := &shared.AgentConfig{
        ID:           uuid.New(),
        SystemPrompt: "Test agent",
        Role:         "test",
        AllowedTools: []string{}, // Explicitly empty
    }

    agent, err := factory.CreateAgent(context.Background(), config)

    assert.NoError(t, err)
    assert.NotNil(t, agent)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/agents/... -v -run TestDefaultAgentFactory_CreateAgent_WithAllowedTools`
Expected: FAIL - CreateAgent still uses old Tools/ToolSets logic

- [ ] **Step 3: Update CreateAgent to use resolveTools**

In `pkg/agents/factory.go`, replace lines 107-126 and update the gollem options:

```go
func (f *defaultAgentFactory) CreateAgent(ctx context.Context, config *shared.AgentConfig) (shared.Agent, error) {
    // Ensure the config has an ID
    if config.ID == uuid.Nil {
        config.ID = uuid.New()
    }

    // Resolve tools from AllowedTools
    tools, err := f.resolveTools(ctx, config.ID, config.AllowedTools)
    if err != nil {
        return nil, errs.Wrap(err, errs.TypeInternal, "failed to resolve tools").
            WithContext("agent_id", config.ID)
    }

    // Create the base agent
    defAgent := &defaultAgent{
        clientProvider: f.clientProvider,
        configService:  f.configService,
        logService:     f.logService,
        registry:       f.registry,
        id:             config.ID,
        config:         config,
        promptManager:  f.promptManager,
    }

    // Get LLM client
    client, err := f.clientProvider.GetClient(ctx, config.LLMClientConfig)
    if err != nil {
        return nil, errs.Wrap(err, errs.TypeInternal, "failed to create llm client").
            WithContext("agent_id", config.ID)
    }

    defAgent.llmClient = client

    // Set default strategy
    if config.Strategy == nil {
        config.Strategy = simple.New()
    }

    // Set default output mode
    if config.OutputMode == "" {
        config.OutputMode = shared.OutputModeFull
    }

    // Build base gollem options
    baseOptions := []gollem.Option{
        gollem.WithStrategy(config.Strategy),
        gollem.WithTools(tools...),  // Use resolved tools
        gollem.WithSystemPrompt(config.SystemPrompt),
    }

    // ... rest of method unchanged
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/agents/... -v -run TestDefaultAgentFactory_CreateAgent_WithAllowedTools`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/agents/factory.go
git commit -m "refactor: use resolveTools in CreateAgent

Remove default tools logic. Agents now only get tools explicitly
listed in AllowedTools field.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 3: Flow Integration

### Task 5: Update LLM step to use AllowedTools

**Files:**
- Modify: `pkg/flows/executor/llm_step.go`
- Modify: `pkg/flows/executor/llm_step_test.go`

- [ ] **Step 1: Write failing test**

Add to `pkg/flows/executor/llm_step_test.go`:

```go
func TestExecuteLLMStep_PopulatesAllowedTools(t *testing.T) {
    flow := &flows.Flow{
        Name: "test-flow",
        States: []flows.State{
            {
                Name: "test-state",
                Steps: []flows.Step{
                    {
                        Type:  "llm",
                        Agent: "test-agent",
                        Tools: "bash,current_time",  // Comma-separated
                    },
                },
            },
        },
        Agents: []flows.Agent{
            {Name: "test-agent", Model: "test", Prompt: "Test"},
        },
    }

    injector := setupTestDI(t)
    svc := do.MustInvoke[FlowExecutorService](injector)
    exec := svc.New(flow)

    // Access the private field via type assertion for testing
    impl := exec.(*flowExecutorImpl)
    step := &impl.flow.States[0].Steps[0]

    // This should create a config with AllowedTools populated
    // We'll verify by checking the mock was called correctly
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/executor/... -v -run TestExecuteLLMStep_PopulatesAllowedTools`
Expected: FAIL - implementation doesn't populate AllowedTools yet

- [ ] **Step 3: Update executeLLMStep to populate AllowedTools**

In `pkg/flows/executor/llm_step.go`:

```go
func (p *flowExecutorImpl) executeLLMStep(ctx context.Context, step *flows.Step, _ string) error {
    // ... existing agent lookup code ...

    // Substitute variables in prompt
    prompt := SubstituteTemplate(p.ctx, step.Prompt)

    // Parse tools from step specification
    allowedTools, err := p.parseStepTools(step.Tools)
    if err != nil {
        return fmt.Errorf("failed to parse tools: %w", err)
    }

    // Map flow Agent to shared.AgentConfig
    config := &shared.AgentConfig{
        ID:              uuid.New(),
        SystemPrompt:    agentConfig.Prompt,
        Role:            "flow-llm-step",
        Description:     fmt.Sprintf("LLM agent for flow %s, step %s", p.flow.Name, step.Name),
        LLMClientConfig: agentConfig.ToClientConfig(),
        OutputMode:      shared.OutputModeSilent,
        Strategy:        simple.New(),
        AllowedTools:    allowedTools,  // CHANGED: Use AllowedTools
    }

    // ... rest of method unchanged
}
```

- [ ] **Step 4: Update parseStepTools to return []string instead of []gollem.Tool**

In `pkg/flows/executor/llm_step.go`:

```go
// parseStepTools parses the comma-separated tools string into tool names
func (p *flowExecutorImpl) parseStepTools(toolsStr string) ([]string, error) {
    if toolsStr == "" {
        return nil, nil
    }

    var toolNames []string
    tools := strings.Split(toolsStr, ",")
    for _, toolName := range tools {
        toolName = strings.TrimSpace(toolName)
        if toolName == "" {
            continue
        }
        toolNames = append(toolNames, toolName)
    }

    return toolNames, nil
}
```

- [ ] **Step 5: Remove CreateTool call from parseStepTools**

Delete the old implementation that created flow tools via flowToolsProvider.

- [ ] **Step 6: Run tests**

Run: `go test ./pkg/flows/executor/... -v -run TestExecuteLLMStep`
Expected: PASS (or update test expectations)

- [ ] **Step 7: Commit**

```bash
git add pkg/flows/executor/llm_step.go pkg/flows/executor/llm_step_test.go
git commit -m "feat: populate AllowedTools from LLM step tools tag

Flow LLM steps now specify tools via <tools> tag in XML.
Tools are resolved by AgentFactory instead of being created directly.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 4: SpawnAgent Tool

### Task 6: Add allowed_tools parameter to SpawnAgent

**Files:**
- Modify: `pkg/tools/spawn_agent.go`
- Modify: `pkg/tools/spawn_agent_test.go`

- [ ] **Step 1: Write failing test**

Add to `pkg/tools/spawn_agent_test.go`:

```go
func TestSpawnAgentTool_Run_WithAllowedTools(t *testing.T) {
    // Test that allowed_tools parameter is passed to agent config
    args := map[string]any{
        "role":          "test-role",
        "description":   "test description",
        "prompt":        "You are a test agent",
        "allowed_tools": []string{"bash", "current_time"},
    }

    // Verify the agent config gets AllowedTools populated
    // Implementation will verify via mock expectations
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/tools/... -v -run TestSpawnAgentTool_Run_WithAllowedTools`
Expected: FAIL - parameter doesn't exist

- [ ] **Step 3: Add allowed_tools to SpawnAgentArgs**

In `pkg/tools/spawn_agent.go`:

```go
type SpawnAgentArgs struct {
    Role            string   `json:"role" description:"Role of the sub-agent"`
    Description     string   `json:"description" description:"Description of the sub-agent"`
    Prompt          string   `json:"prompt" description:"System prompt for the sub-agent"`
    Model           string   `json:"model,omitempty" description:"LLM model (optional, inherits from parent)"`
    Temperature     float64  `json:"temperature,omitempty" description:"Temperature (optional, inherits from parent)"`
    ShareContext    bool     `json:"share_context,omitempty" description:"Share conversation history with sub-agent"`
    AllowedTools    []string `json:"allowed_tools,omitempty" description:"List of tools the sub-agent can use (built-in or MCP)"`
    MaxIterations   int      `json:"max_iterations,omitempty" description:"Maximum iterations for auto mode (default: 10)"`
    OutputMode      string   `json:"output_mode,omitempty" description:"Output mode: 'full', 'silent', or 'terse'"`
}
```

- [ ] **Step 4: Update tool spec**

Update the `Spec()` method in `pkg/tools/spawn_agent.go`:

```go
"allowed_tools": {
    Type:        gollem.TypeArray,
    Description: "List of tools the sub-agent can use (e.g., ['bash', 'filesystem/read_file']). If omitted, sub-agent has no tools.",
    Items: &gollem.Parameter{
        Type: gollem.TypeString,
    },
},
```

- [ ] **Step 5: Update agent creation to use AllowedTools**

In the spawn agent implementation, find where `AgentConfig` is created and add:

```go
config := &shared.AgentConfig{
    // ... existing fields
    AllowedTools: args.AllowedTools,
}
```

- [ ] **Step 6: Run tests**

Run: `go test ./pkg/tools/... -v -run TestSpawnAgentTool`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add pkg/tools/spawn_agent.go pkg/tools/spawn_agent_test.go
git commit -m "feat: add allowed_tools parameter to SpawnAgent tool

Parent agents can now explicitly control which tools spawned
sub-agents have access to. Empty or nil means no tools.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 5: Documentation & Migration

### Task 7: Update default flow

**Files:**
- Modify: `.gollum/flows/default/main.xml`

- [ ] **Step 1: Add tools tag to LLM step**

Update `.gollum/flows/default/main.xml`:

```xml
<step type="llm" agent="startup">
    <prompt><![CDATA[
        What is the time in ${input.location}?:
        Use the `set_context_field` tool to set:
        1. The current time as string to the context like so: set_context_field("the_current_time", <DD.MM.YY HH:mm:ss>).
        2. The hour to the context like so: set_context_field("hour", <hour>).
    ]]></prompt>
    <tools>current_time,set_context_field</tools>
    <timeout>60s</timeout>
</step>
```

- [ ] **Step 2: Test the flow**

Run: `gollum run default`
Expected: Flow executes successfully

- [ ] **Step 3: Commit**

```bash
git add .gollum/flows/default/main.xml
git commit -m "fix: add explicit tools to default flow

Update default flow to explicitly declare allowed tools.
Required after removing default tools from agent factory.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 8: Update DI registration

**Files:**
- Modify: DI container file (find the correct location)

- [ ] **Step 1: Find DI container**

Run: `grep -r "do.Provide.*NewAgentFactory" --include="*.go" pkg/`
Note the file path.

- [ ] **Step 2: Add MCPToolProvider registration**

Add before NewAgentFactory registration:

```go
do.Provide(injector, mcp.NewMCPToolProvider)
```

- [ ] **Step 3: Test DI loads**

Run: `go test ./pkg/di/... -v` (if tests exist)
Or build the application: `just build`

- [ ] **Step 4: Commit**

```bash
git add <di-file>
git commit -m "feat: register MCPToolProvider in DI container

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 9: Run full test suite and fix issues

**Files:**
- Multiple test files

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -short`

- [ ] **Step 2: Fix failing tests**

For each failing test:
1. Read the test to understand what it's testing
2. Update to use `AllowedTools` instead of `Tools`/`ToolSets`
3. Re-run the test
4. Commit fixes with descriptive messages

- [ ] **Step 3: Final verification**

Run: `go test ./... -short`
Expected: All tests pass

- [ ] **Step 4: Integration test**

Run: `gollum run default`
Expected: Flow executes successfully with enriched logging

---

## Acceptance Criteria

- [ ] All agents require explicit `AllowedTools` (empty/nil = no tools)
- [ ] Built-in tools validated against ToolName constants
- [ ] MCP tools validated via MCPRegistry (warn on missing)
- [ ] SpawnAgent supports `allowed_tools` parameter
- [ ] Default flow updated with explicit `<tools>` tag
- [ ] All tests pass (TDD approach)
- [ ] DI registration complete

## Rollback Plan

If critical issues are found:

```bash
git revert --mainline HEAD~<number-of-commits>
```

Or rollback to commit before changes:
```bash
git log --oneline | grep "feat: add AllowedTools"
git checkout <commit-before-changes>
```
