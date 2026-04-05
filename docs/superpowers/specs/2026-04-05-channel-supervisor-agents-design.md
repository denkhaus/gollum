# Channel-Specific Supervisor Agents Design

**Date:** 2025-04-05
**Status:** Approved

## Problem

Die `SubmitInput`-Methode im `ChannelFacade` hat keine Möglichkeit, Input an einen channel-spezifischen Agenten zu routen. Aktuell wird einfach der erste verfügbare Agent aus der Registry verwendet.

## Solution

1:1-Beziehung zwischen Channel und Supervisor-Agent: **Channel ID = Agent ID**

Jeder Channel erhält bei der Registrierung einen eigenen Supervisor-Agenten, der in der `AgentRegistry` gespeichert wird.

## Architecture

### Components

1. **Channel Interface** - Ändert `ID()` von `string` zu `uuid.UUID`
2. **AgentFactory** - Options-Pattern für `CreateSupervisorAgent`
3. **ChannelFacade** - Erstellt Agenten bei `RegisterChannel`, entfernt bei `UnregisterChannel`
4. **SubmitInput** - Erhält `channelID` Parameter und routed zum channel-spezifischen Agenten

### Data Flow

```
Channel Registration:
RegisterChannel(channel) → CreateSupervisorAgent(WithAgentID(channel.ID()))
                          → registry.Register(agent, config)
                          → channels[channelID.String()] = channel

Input Handling:
SubmitInput(ctx, channelID, input) → commandManager.Execute()
                                   → registry.GetAgent(channelID)
                                   → agent.Execute(ctx, input)

Channel Unregistration:
UnregisterChannel(channelID) → registry.Cleanup(channelID)
                            → delete(channels, channelID.String())
```

## Changes

### 1. Channel Interface (`pkg/channel/interface.go`)

```go
type Channel interface {
    ID() uuid.UUID  // Changed from string
    // ... rest unchanged
}
```

### 2. AgentFactory Options (`pkg/agents/factory.go`)

```go
type SupervisorAgentOption func(*shared.AgentConfig)

func WithAgentID(id uuid.UUID) SupervisorAgentOption {
    return func(cfg *shared.AgentConfig) {
        cfg.ID = id
    }
}

func (f *defaultAgentFactory) CreateSupervisorAgent(
    ctx context.Context,
    opts ...SupervisorAgentOption,
) (shared.Agent, *shared.AgentConfig, error) {
    agentConfig := &shared.AgentConfig{
        AllowCompaction: true,
        SystemPrompt:    /* supervisor prompt */,
        AllowedTools:    /* MCP + builtin */,
        Role:           "Supervisor Agent",
        LLMClientConfig: /* default */,
    }

    for _, opt := range opts {
        opt(agentConfig)
    }

    if agentConfig.ID == uuid.Nil {
        agentConfig.ID = uuid.New()
    }

    // ... rest of implementation
}
```

### 3. ChannelFacade (`pkg/channel/facade.go`)

```go
type channelFacadeImpl struct {
    // ... existing fields
    agentFactory shared.AgentFactory
}

func (p *channelFacadeImpl) RegisterChannel(channel Channel) error {
    channelID := channel.ID()
    if _, exists := p.channels[channelID.String()]; exists {
        return fmt.Errorf("channel %s already registered", channelID)
    }

    agent, _, err := p.agentFactory.CreateSupervisorAgent(
        context.Background(),
        WithAgentID(channelID),
    )
    if err != nil {
        return fmt.Errorf("failed to create channel agent: %w", err)
    }

    p.channels[channelID.String()] = channel
    return nil
}

func (p *channelFacadeImpl) UnregisterChannel(channelID uuid.UUID) error {
    p.registry.Cleanup(channelID)
    delete(p.channels, channelID.String())
    return nil
}

func (p *channelFacadeImpl) SubmitInput(
    ctx context.Context,
    channelID uuid.UUID,
    input string,
) (InputResult, error) {
    // Command check
    handled, response, err := p.commandManager.Execute(ctx, input)
    if handled {
        return InputResult{Handled: true, IsCommand: true, Response: response, Error: err}, nil
    }

    // Get channel-specific agent
    agent, exists := p.registry.GetAgent(channelID)
    if !exists {
        return InputResult{}, fmt.Errorf("channel agent not found: %s", channelID)
    }

    // Execute
    resp, err := agent.Execute(ctx, gollem.Text(input))
    if err != nil {
        return InputResult{Handled: true, Error: err}, nil
    }

    content := strings.Join(resp.Texts, "\n")
    return InputResult{Handled: true, Response: content}, nil
}
```

### 4. TUIChannel (`pkg/tui/channel.go`)

```go
type TUIChannel struct {
    id          uuid.UUID  // Changed from string
    messageChan chan<- channel.Message
    agentID     uuid.UUID
    agentRole   string
}

func NewTUIChannel(messageChan chan<- channel.Message) *TUIChannel {
    return &TUIChannel{
        id:          uuid.New(),
        messageChan: messageChan,
        agentID:     uuid.Nil,
        agentRole:   "assistant",
    }
}

func (c *TUIChannel) ID() uuid.UUID {
    return c.id
}
```

### 5. ChannelFacade Interface (`pkg/channel/interface.go`)

```go
type ChannelFacade interface {
    // ... existing methods
    SubmitInput(ctx context.Context, channelID uuid.UUID, input string) (InputResult, error)
    RegisterChannel(channel Channel) error
    UnregisterChannel(channelID uuid.UUID) error
}
```

## Breaking Changes

- `Channel.ID()` returns `uuid.UUID` instead of `string`
- `SubmitInput()` requires `channelID` parameter
- `UnregisterChannel()` requires `uuid.UUID` instead of `string`

## Testing

- Unit tests for `RegisterChannel` with agent creation
- Unit tests for `UnregisterChannel` with agent cleanup
- Unit tests for `SubmitInput` routing to correct agent
- Integration tests with TUIChannel

## Migration

All channel implementations must update `ID()` to return `uuid.UUID`:

1. `pkg/tui/channel.go` - Update to use UUID
2. Any future channel implementations
