# Channel-Specific Supervisor Agents Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Jeder Channel erhält bei Registrierung einen eigenen Supervisor-Agenten (Channel ID = Agent ID)

**Architecture:** 1:1 Beziehung zwischen Channel und Agent. ChannelFacade erstellt Agenten via AgentFactory mit Options-Pattern. SubmitInput routed an channel-spezifischen Agenten.

**Tech Stack:** Go 1.26, samber/do DI, google/uuid, m-mizutani/gollem

---

## Chunk 1: AgentFactory Options Pattern

### Task 1: WithAgentID Option hinzufügen

**Files:**
- Modify: `pkg/agents/factory.go:246-298` (CreateSupervisorAgent)
- Create: `pkg/agents/factory_test.go` (Tests für Options)

- [ ] **Step 1: Schreibe failing Test für WithAgentID Option**

```go
// pkg/agents/factory_test.go
func TestCreateSupervisorAgent_WithAgentID(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    // Setup DI injector with mocks
    injector := setupTestInjector(ctrl)

    factory, err := NewAgentFactory(injector)
    require.NoError(t, err)

    ctx := context.Background()
    customID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

    agent, config, err := factory.CreateSupervisorAgent(ctx, WithAgentID(customID))

    require.NoError(t, err)
    assert.NotNil(t, agent)
    assert.Equal(t, customID, agent.GetID())
    assert.Equal(t, customID, config.ID)
}
```

- [ ] **Step 2: Test ausführen (erwarte FAIL)**

Run: `go test ./pkg/agents/... -run TestCreateSupervisorAgent_WithAgentID -v`
Expected: `undefined: WithAgentID`

- [ ] **Step 3: WithAgentID Option implementieren**

```go
// pkg/agents/factory.go - nach Zeile 298

// SupervisorAgentOption is a functional option for CreateSupervisorAgent
type SupervisorAgentOption func(*shared.AgentConfig)

// WithAgentID sets a specific agent ID instead of generating a new one
func WithAgentID(id uuid.UUID) SupervisorAgentOption {
    return func(cfg *shared.AgentConfig) {
        cfg.ID = id
    }
}
```

- [ ] **Step 4: Test ausführen (erwarte PASS)**

Run: `go test ./pkg/agents/... -run TestCreateSupervisorAgent_WithAgentID -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/agents/factory.go pkg/agents/factory_test.go
git commit -m "feat(agents): add WithAgentID option for CreateSupervisorAgent"
```

### Task 2: CreateSupervisorAgent mit Options Pattern

- [ ] **Step 1: Schreibe Test für Options-Anwendung**

```go
// pkg/agents/factory_test.go
func TestCreateSupervisorAgent_AppliesOptions(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    injector := setupTestInjector(ctrl)
    factory, err := NewAgentFactory(injector)
    require.NoError(t, err)

    ctx := context.Background()
    customID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

    // Create agent with custom ID
    agent1, config1, err := factory.CreateSupervisorAgent(ctx, WithAgentID(customID))
    require.NoError(t, err)
    assert.Equal(t, customID, agent1.GetID())
    assert.Equal(t, customID, config1.ID)

    // Create agent without options (should generate new ID)
    agent2, config2, err := factory.CreateSupervisorAgent(ctx)
    require.NoError(t, err)
    assert.NotEqual(t, uuid.Nil, agent2.GetID())
    assert.NotEqual(t, customID, agent2.GetID())
}
```

- [ ] **Step 2: Test ausführen (erwarte FAIL - ID wird nicht aus Option gesetzt)**

Run: `go test ./pkg/agents/... -run TestCreateSupervisorAgent_AppliesOptions -v`
Expected: FAIL oder IDs sind gleich

- [ ] **Step 3: CreateSupervisorAgent um Options-Parameter erweitern**

```go
// pkg/agents/factory.go - CreateSupervisorAgent Signatur ändern

func (p *defaultAgentFactory) CreateSupervisorAgent(
    ctx context.Context,
    opts ...SupervisorAgentOption,
) (shared.Agent, *shared.AgentConfig, error) {
    // Get supervisor prompt
    systemPrompt, err := p.promptManager.GetPromptWithContext(ctx,
        prompt.PromptIDSupervisorSystem,
        &prompt.RenderContext{
            Workspace: &shared.WorkspaceContext{
                SkillsXML:   p.skillsService.GetSkillsXML(),
                Skills:      p.skillsService.GetSkillInfos(),
                CurrentPath: p.workspaceService.GetCurrentWorkspace(),
            },
        },
    )
    if err != nil {
        return nil, nil, fmt.Errorf("failed to get supervisor prompt: %w", err)
    }

    // Combine MCP tools and built-in tools
    allowedTools := p.mcpRegistry.GetToolNames()
    for _, toolName := range shared.SupervisorBuiltinTools {
        allowedTools = append(allowedTools, toolName.String())
    }

    // Create base agent config
    agentConfig := &shared.AgentConfig{
        AllowCompaction: true,
        SystemPrompt:    systemPrompt,
        AllowedTools:    allowedTools,
        Role:            "Supervisor Agent",
        LLMClientConfig: &shared.LLMClientConfig{
            Model: "anthropic/glm-4.7",
        },
    }

    // Apply options
    for _, opt := range opts {
        opt(agentConfig)
    }

    // Generate ID if not set by options
    if agentConfig.ID == uuid.Nil {
        agentConfig.ID = uuid.New()
    }

    // Create agent
    agent, err := p.CreateAgent(ctx, agentConfig)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to create supervisor agent: %v", err)
    }

    // Register in registry
    if err := p.registry.Register(agent, agentConfig); err != nil {
        return nil, nil, fmt.Errorf("failed to register supervisor agent: %w", err)
    }
    p.logService.Infof("Supervisor agent %s registered", agent.GetID())

    return agent, agentConfig, nil
}
```

- [ ] **Step 4: Test ausführen (erwarte PASS)**

Run: `go test ./pkg/agents/... -run TestCreateSupervisorAgent_AppliesOptions -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/agents/factory.go pkg/agents/factory_test.go
git commit -m "refactor(agents): apply options pattern to CreateSupervisorAgent"
```

---

## Chunk 2: Channel Interface zu UUID

### Task 3: Channel.ID() zu UUID ändern

**Files:**
- Modify: `pkg/channel/interface.go:14` (Channel Interface)
- Modify: `pkg/tui/channel.go:14,25,39-41` (TUIChannel)
- Test: `pkg/tui/channel_test.go`

- [ ] **Step 1: Schreibe Test für UUID ID**

```go
// pkg/tui/channel_test.go
func TestTUIChannel_ID_ReturnsUUID(t *testing.T) {
    channel := NewTUIChannel(nil)

    id := channel.ID()

    assert.NotEqual(t, uuid.Nil, id)
    // Should be a valid UUID v4
    assert.Equal(t, 4, id.Version())
}
```

- [ ] **Step 2: Test ausführen (erwarte FAIL - string vs UUID)**

Run: `go test ./pkg/tui/... -run TestTUIChannel_ID_ReturnsUUID -v`
Expected: FAIL - Type mismatch

- [ ] **Step 3: Channel Interface ändern**

```go
// pkg/channel/interface.go

type Channel interface {
    // ID returns a unique identifier for this channel
    ID() uuid.UUID  // Changed from: ID() string

    // ... rest unchanged
}
```

- [ ] **Step 4: TUIChannel anpassen**

```go
// pkg/tui/channel.go

type TUIChannel struct {
    id          uuid.UUID  // Changed from: id string
    messageChan chan<- channel.Message
    agentID     uuid.UUID
    agentRole   string
}

func NewTUIChannel(messageChan chan<- channel.Message) *TUIChannel {
    return &TUIChannel{
        id:          uuid.New(),  // Generate UUID
        messageChan: messageChan,
        agentID:     uuid.Nil,
        agentRole:   "assistant",
    }
}

func (c *TUIChannel) ID() uuid.UUID {
    return c.id
}
```

- [ ] **Step 5: Test ausführen (erwarte PASS)**

Run: `go test ./pkg/tui/... -run TestTUIChannel_ID_ReturnsUUID -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add pkg/channel/interface.go pkg/tui/channel.go pkg/tui/channel_test.go
git commit -m "refactor(channel): change Channel.ID() to return uuid.UUID"
```

---

## Chunk 3: ChannelFacade AgentFactory Integration

### Task 4: AgentFactory zu ChannelFacade hinzufügen

**Files:**
- Modify: `pkg/channel/facade.go:24-32` (channelFacadeImpl struct)
- Modify: `pkg/di/container.go` (DI registration)
- Test: `pkg/channel/facade_test.go`

- [ ] **Step 1: Schreibe Test für AgentFactory Integration**

```go
// pkg/channel/facade_test.go
func TestNewChannelFacade_WithAgentFactory(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    injector := do.New()

    // Setup mocks
    mockLogger := logger.NewMockLoggerService(ctrl)
    mockRegistry := registry.NewMockAgentRegistry(ctrl)
    mockAgentFactory := agents.NewMockAgentFactory(ctrl)

    mockLogger.EXPECT().Debug(gomock.Any()).AnyTimes()

    do.ProvideValue[logger.LoggerService](injector, mockLogger)
    do.ProvideValue[registry.AgentRegistry](injector, mockRegistry)
    do.ProvideValue[shared.AgentFactory](injector, mockAgentFactory)
    do.Provide(injector, NewCommandManager)

    // Mock config
    mockCfg := config.NewMockConfigService(ctrl)
    mockCfg.EXPECT().GetLoggingConfig().Return(&config.LoggingConfig{
        SessionLogBufferSize: 100,
    }).AnyTimes()
    do.ProvideValue[config.ConfigService](injector, mockCfg)

    facade, err := NewChannelFacade(injector)

    require.NoError(t, err)
    assert.NotNil(t, facade)
}
```

- [ ] **Step 2: Test ausführen (erwarte FAIL - AgentFactory nicht in DI)**

Run: `go test ./pkg/channel/... -run TestNewChannelFacade_WithAgentFactory -v`
Expected: FAIL - DI error

- [ ] **Step 3: AgentFactory zu channelFacadeImpl hinzufügen**

```go
// pkg/channel/facade.go

type channelFacadeImpl struct {
    mu             sync.RWMutex
    channels       map[string]Channel
    commandManager CommandManager
    registry       registry.AgentRegistry
    logs           []LogEntry
    maxLogs        int
    agentFactory   shared.AgentFactory  // NEW
}
```

- [ ] **Step 4: NewChannelFacade mit AgentFactory**

```go
// pkg/channel/facade.go

func NewChannelFacade(injector do.Injector) (ChannelFacadeService, error) {
    cm := do.MustInvoke[CommandManagerService](injector)
    reg := do.MustInvoke[registry.AgentRegistry](injector)
    cfg := do.MustInvoke[config.ConfigService](injector)
    af := do.MustInvoke[shared.AgentFactory](injector)  // NEW

    maxLogs := cfg.GetLoggingConfig().SessionLogBufferSize

    return &channelFacadeImpl{
        commandManager: cm,
        registry:       reg,
        agentFactory:   af,  // NEW
        channels:       make(map[string]Channel),
        logs:           make([]LogEntry, 0, maxLogs),
        maxLogs:        maxLogs,
    }, nil
}
```

- [ ] **Step 5: Test ausführen (erwarte PASS)**

Run: `go test ./pkg/channel/... -run TestNewChannelFacade_WithAgentFactory -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add pkg/channel/facade.go pkg/channel/facade_test.go pkg/di/container.go
git commit -m "feat(channel): add AgentFactory to ChannelFacade"
```

---

## Chunk 4: RegisterChannel mit Agent-Erstellung

### Task 5: RegisterChannel erstellt Supervisor-Agent

**Files:**
- Modify: `pkg/channel/facade.go:56-65` (RegisterChannel)
- Test: `pkg/channel/facade_test.go`

- [ ] **Step 1: Schreibe Test für Agent-Erstellung bei Registrierung**

```go
// pkg/channel/facade_test.go
func TestChannelFacade_RegisterChannel_CreatesAgent(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    injector := setupChannelFacadeInjector(ctrl)

    // Mock agent factory
    mockAgentFactory := agents.NewMockAgentFactory(ctrl)
    channelID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

    // Expect CreateSupervisorAgent to be called with channel ID
    mockAgent := agents.NewMockAgent(ctrl)
    mockAgent.EXPECT().GetID().Return(channelID).AnyTimes()

    mockAgentConfig := &shared.AgentConfig{ID: channelID}
    mockAgentFactory.EXPECT().CreateSupervisorAgent(gomock.Any(), gomock.Any()).
        Return(mockAgent, mockAgentConfig, nil)

    // Update injector with mock factory
    do.ProvideValue[shared.AgentFactory](injector, mockAgentFactory)

    facade, err := NewChannelFacade(injector)
    require.NoError(t, err)

    // Create mock channel
    mockChannel := channel.NewMockChannel(ctrl)
    mockChannel.EXPECT().ID().Return(channelID).AnyTimes()

    // Register channel
    err = facade.RegisterChannel(mockChannel)
    require.NoError(t, err)

    // Verify agent was created with channel ID
    // (This is verified by the mock expectation above)
}
```

- [ ] **Step 2: Test ausführen (erwarte FAIL - keine Agent-Erstellung)**

Run: `go test ./pkg/channel/... -run TestChannelFacade_RegisterChannel_CreatesAgent -v`
Expected: FAIL - Mock expectation not met

- [ ] **Step 3: RegisterChannel implementieren**

```go
// pkg/channel/facade.go

func (p *channelFacadeImpl) RegisterChannel(channel Channel) error {
    p.mu.Lock()
    defer p.mu.Unlock()

    channelID := channel.ID()
    channelIDStr := channelID.String()

    if _, exists := p.channels[channelIDStr]; exists {
        return fmt.Errorf("channel %s already registered", channelIDStr)
    }

    // Create supervisor agent for this channel
    ctx := context.Background()
    agent, _, err := p.agentFactory.CreateSupervisorAgent(
        ctx,
        WithAgentID(channelID),  // Use channel ID as agent ID
    )
    if err != nil {
        return fmt.Errorf("failed to create channel agent: %w", err)
    }

    p.channels[channelIDStr] = channel
    return nil
}
```

Hinweis: `WithAgentID` muss importiert werden - füge hinzu:
```go
import (
    // ... existing imports
    "github.com/denkhaus/gollum/pkg/agents"
)
```

Und der Aufruf:
```go
    agent, _, err := p.agentFactory.CreateSupervisorAgent(
        ctx,
        agents.WithAgentID(channelID),
    )
```

- [ ] **Step 4: Test ausführen (erwarte PASS)**

Run: `go test ./pkg/channel/... -run TestChannelFacade_RegisterChannel_CreatesAgent -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/channel/facade.go pkg/channel/facade_test.go
git commit -m "feat(channel): create supervisor agent on RegisterChannel"
```

---

## Chunk 5: UnregisterChannel mit Agent-Cleanup

### Task 6: UnregisterChannel entfernt Agent

**Files:**
- Modify: `pkg/channel/facade.go:68-74` (UnregisterChannel)
- Modify: `pkg/channel/interface.go:72` (Interface signature)
- Test: `pkg/channel/facade_test.go`

- [ ] **Step 1: Schreibe Test für Agent-Cleanup**

```go
// pkg/channel/facade_test.go
func TestChannelFacade_UnregisterChannel_RemovesAgent(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    injector := setupChannelFacadeInjector(ctrl)

    // Mock registry
    mockRegistry := registry.NewMockAgentRegistry(ctrl)
    channelID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
    mockRegistry.EXPECT().Cleanup(channelID).Return(nil)

    do.ProvideValue[registry.AgentRegistry](injector, mockRegistry)

    facade, err := NewChannelFacade(injector)
    require.NoError(t, err)

    // Register a channel first
    mockChannel := channel.NewMockChannel(ctrl)
    mockChannel.EXPECT().ID().Return(channelID)

    err = facade.RegisterChannel(mockChannel)
    require.NoError(t, err)

    // Unregister
    err = facade.UnregisterChannel(channelID)
    require.NoError(t, err)
}
```

- [ ] **Step 2: Test ausführen (erwarte FAIL - signature mismatch)**

Run: `go test ./pkg/channel/... -run TestChannelFacade_UnregisterChannel_RemovesAgent -v`
Expected: FAIL - string vs uuid.UUID

- [ ] **Step 3: ChannelFacade Interface Signatur ändern**

```go
// pkg/channel/interface.go

type ChannelFacade interface {
    // ...
    // UnregisterChannel removes a channel and its agent
    UnregisterChannel(channelID uuid.UUID) error  // Changed from: channelID string
    // ...
}
```

- [ ] **Step 4: UnregisterChannel implementieren**

```go
// pkg/channel/facade.go

func (p *channelFacadeImpl) UnregisterChannel(channelID uuid.UUID) error {
    p.mu.Lock()
    defer p.mu.Unlock()

    channelIDStr := channelID.String()

    // Cleanup channel agent from registry
    if err := p.registry.Cleanup(channelID); err != nil {
        // Log warning but don't fail - channel should still be removed
        // p.logService.Warn("failed to cleanup channel agent",
        //     zap.String("channel_id", channelIDStr),
        //     zap.Error(err))
    }

    delete(p.channels, channelIDStr)
    return nil
}
```

- [ ] **Step 5: Test ausführen (erwarte PASS)**

Run: `go test ./pkg/channel/... -run TestChannelFacade_UnregisterChannel_RemovesAgent -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add pkg/channel/facade.go pkg/channel/interface.go pkg/channel/facade_test.go
git commit -m "feat(channel): cleanup agent on UnregisterChannel"
```

---

## Chunk 6: SubmitInput mit Channel-Routing

### Task 7: SubmitInput routet zu Channel-Agent

**Files:**
- Modify: `pkg/channel/facade.go:113-171` (SubmitInput)
- Modify: `pkg/channel/interface.go:63` (Interface signature)
- Test: `pkg/channel/facade_test.go`

- [ ] **Step 1: Schreibe Test für Channel-Routing**

```go
// pkg/channel/facade_test.go
func TestChannelFacade_SubmitInput_RoutesToChannelAgent(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    injector := setupChannelFacadeInjector(ctrl)

    channelID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

    // Mock registry with agent
    mockRegistry := registry.NewMockAgentRegistry(ctrl)
    mockAgent := agents.NewMockAgent(ctrl)

    mockRegistry.EXPECT().GetAgent(channelID).Return(mockAgent, true)

    // Mock agent execution
    expectedResponse := &gollem.ExecuteResponse{
        Texts: []string{"Hello from channel agent"},
    }
    mockAgent.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(expectedResponse, nil)

    do.ProvideValue[registry.AgentRegistry](injector, mockRegistry)

    // Mock command manager (not a command)
    mockCmdMgr := NewMockCommandManager(ctrl)
    mockCmdMgr.EXPECT().Execute(gomock.Any(), "test input").Return(false, "", nil)
    do.ProvideValue[CommandManager](injector, mockCmdMgr)

    facade, err := NewChannelFacade(injector)
    require.NoError(t, err)

    // Submit input
    ctx := context.Background()
    result, err := facade.SubmitInput(ctx, channelID, "test input")

    require.NoError(t, err)
    assert.True(t, result.Handled)
    assert.Equal(t, "Hello from channel agent", result.Response)
}
```

- [ ] **Step 2: Test ausführen (erwarte FAIL - signature/implementation)**

Run: `go test ./pkg/channel/... -run TestChannelFacade_SubmitInput_RoutesToChannelAgent -v`
Expected: FAIL

- [ ] **Step 3: ChannelFacade Interface Signatur ändern**

```go
// pkg/channel/interface.go

type ChannelFacade interface {
    // SubmitInput handles user input from a specific channel
    SubmitInput(ctx context.Context, channelID uuid.UUID, input string) (InputResult, error)
    // ...
}
```

- [ ] **Step 4: SubmitInput implementieren**

```go
// pkg/channel/facade.go

func (p *channelFacadeImpl) SubmitInput(
    ctx context.Context,
    channelID uuid.UUID,
    input string,
) (InputResult, error) {
    // First check if it's a slash command
    handled, response, err := p.commandManager.Execute(ctx, input)
    if handled {
        return InputResult{
            Handled:   true,
            IsCommand: true,
            Response:  response,
            Error:     err,
        }, nil
    }

    // Get channel-specific agent from registry
    agent, exists := p.registry.GetAgent(channelID)
    if !exists {
        return InputResult{}, fmt.Errorf("channel agent not found: %s", channelID)
    }

    // Execute agent with input
    resp, err := agent.Execute(ctx, gollem.Text(input))
    if err != nil {
        return InputResult{
            Handled: true,
            Error:   err,
        }, nil
    }

    // Extract response content
    var content string
    if resp != nil && len(resp.Texts) > 0 {
        content = strings.Join(resp.Texts, "\n")
    }

    return InputResult{
        Handled:  true,
        Response: content,
    }, nil
}
```

- [ ] **Step 5: Test ausführen (erwarte PASS)**

Run: `go test ./pkg/channel/... -run TestChannelFacade_SubmitInput_RoutesToChannelAgent -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add pkg/channel/facade.go pkg/channel/interface.go pkg/channel/facade_test.go
git commit -m "feat(channel): route SubmitInput to channel-specific agent"
```

---

## Chunk 7: Bestehende Tests aktualisieren

### Task 8: Veraltete Tests reparieren

**Files:**
- Modify: `pkg/channel/facade_test.go` (alle betroffenen Tests)

- [ ] **Step 1: Alle Tests ausführen um Probleme zu finden**

Run: `go test ./pkg/channel/... -v`
Expected: Einige Tests FAIL wg. Signatur-Änderungen

- [ ] **Step 2: Tests aktualisieren**

Suche nach Tests die:
- `UnregisterChannel(string)` aufrufen → ändere zu `UnregisterChannel(uuid)`
- `SubmitInput(ctx, input)` aufrufen → ändere zu `SubmitInput(ctx, channelID, input)`
- `ID() string` erwarten → ändere zu `ID() uuid.UUID`

Beispiel:
```go
// ALT
err := facade.UnregisterChannel("test-channel")

// NEU
testID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
err := facade.UnregisterChannel(testID)
```

- [ ] **Step 3: Alle Tests ausführen**

Run: `go test ./pkg/channel/... -v`
Expected: Alle PASS

- [ ] **Step 4: Commit**

```bash
git add pkg/channel/facade_test.go
git commit -m "test(channel): update tests for UUID channel IDs"
```

---

## Chunk 8: Integration Tests

### Task 9: End-to-End Test für Channel-Agent Lifecycle

- [ ] **Step 1: Integration Test schreiben**

```go
// pkg/channel/facade_integration_test.go
package channel_test

import (
    "context"
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/denkhaus/gollum/pkg/channel"
)

func TestChannelAgent_Lifecycle(t *testing.T) {
    if testing.Short() {
        t.Skip("integration test")
    }

    // Setup full DI container
    // ... (use existing test setup)

    channelID := uuid.New()

    // 1. Register channel - should create agent
    mockChannel := channel.NewMockChannel(ctrl)
    mockChannel.EXPECT().ID().Return(channelID).AnyTimes()

    err := facade.RegisterChannel(mockChannel)
    require.NoError(t, err)

    // 2. Verify agent exists in registry
    agent, exists := registry.GetAgent(channelID)
    assert.True(t, exists)
    assert.NotNil(t, agent)

    // 3. Submit input - should route to channel agent
    result, err := facade.SubmitInput(context.Background(), channelID, "hello")
    require.NoError(t, err)
    assert.True(t, result.Handled)

    // 4. Unregister channel - should cleanup agent
    err = facade.UnregisterChannel(channelID)
    require.NoError(t, err)

    // 5. Verify agent is gone
    _, exists = registry.GetAgent(channelID)
    assert.False(t, exists)
}
```

- [ ] **Step 2: Test ausführen**

Run: `go test ./pkg/channel/... -run TestChannelAgent_Lifecycle -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add pkg/channel/facade_integration_test.go
git commit -m "test(channel): add integration test for channel-agent lifecycle"
```

---

## Final Steps

### Task 10: Documentation und Cleanup

- [ ] **Step 1: Go Vet**

Run: `go vet ./pkg/channel/... ./pkg/agents/... ./pkg/tui/...`
Expected: Keine Warnungen

- [ ] **Step 2: Alle Tests ausführen**

Run: `go test ./... -count=1`
Expected: Alle PASS (außer pre-existing failures)

- [ ] **Step 3: Build verifizieren**

Run: `just build`
Expected: Erfolgreich

- [ ] **Step 4: Final Commit**

```bash
git add .
git commit -m "feat: complete channel-specific supervisor agents implementation"
```

---

## Summary

This plan implements a 1:1 relationship between Channels and Supervisor Agents:

1. **AgentFactory**: Options-Pattern mit `WithAgentID()` für Channel-spezifische IDs
2. **Channel Interface**: `ID()` gibt `uuid.UUID` statt `string` zurück
3. **ChannelFacade**: Erstellt Agent bei `RegisterChannel`, entfernt bei `UnregisterChannel`
4. **SubmitInput**: Route zu Channel-Agent via `channelID` Parameter

**Breaking Changes:**
- `Channel.ID()`: `string` → `uuid.UUID`
- `SubmitInput()`: Neuer Parameter `channelID uuid.UUID`
- `UnregisterChannel()`: `string` → `uuid.UUID`

**Migration:** Alle Channel-Implementierungen müssen `ID()` zu UUID ändern.
