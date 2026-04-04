# ACP Integration Design for Gollum

**Date:** 2026-04-05
**Status:** Approved
**Author:** Claude (Brainstorming Session)

## Overview

This document describes the integration of the [Agent Client Protocol (ACP)](https://github.com/zed-industries/agent-client-protocol) into Gollum, enabling code editors (like Zed) to directly control Gollum's agent loop and flows.

## Goals

1. **ACP Agent Server** - Make Gollum act as an ACP-compatible agent server
2. **Go-based Agent Loop** - Connect existing `shared.Agent.Execute()` to ACP
3. **Flow Execution Tools** - Provide `execute_flow` and `list_flows` tools for agents

## Architecture

### Package Structure

```
pkg/acp/
├── service.go           # AcpService implements acp.Agent interface
├── agent_adapter.go     # Adapter between ACP and shared.Agent
├── tools.go             # executeFlowTool, listFlowsTool
├── session_store.go     # Session-Verwaltung (acp.SessionStore)
└── middleware.go        # Logging, Recovery Middleware

pkg/flows/registry/
└── service.go           # Add: ListFlows() method (returns []*FlowInfo)

pkg/cli/
└── acp.go               # gollum acp command

pkg/tools/
└── flow_tools.go        # Neue Tools: ExecuteFlow, ListFlows
```

### Data Flow

```
┌─────────────┐     stdio      ┌──────────────────┐
│ ACP-Client  │◄──────────────►│   AcpService     │
│ (Zed, etc.) │  JSON-RPC 2.0  │  (implements    │
└─────────────┘                │   acp.Agent)     │
                               └────────┬─────────┘
                                        │
                        ┌───────────────┼───────────────┐
                        ↓               ↓               ↓
                  ┌──────────┐   ┌──────────┐   ┌──────────┐
                  │   Flow   │   │   Agent  │   │  Logger  │
                  │ Registry │   │ Adapter  │   │ Service  │
                  └──────────┘   └────┬─────┘   └──────────┘
                                      │
                              ┌───────┴────────┐
                              ↓                ↓
                      ┌─────────────┐  ┌─────────────┐
                      │shared.Agent │  │   Tools     │
                      │  Execute()  │  │  (gollem)   │
                      └─────────────┘  └─────────────┘
```

### Component Interfaces

#### AcpService

```go
package acp

type AcpService interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
}

type acpServiceImpl struct {
    conn         *acp.AgentSideConnection
    agentAdapter *AgentAdapter
    store        acp.SessionStore[*AgentSession]
    flowRegistry flows.FlowRegistry
    logger       logger.LoggerService
}

// DI-Konstruktor
func NewAcpService(injector do.Injector) (AcpService, error)
```

#### AgentAdapter

```go
package acp

type AgentAdapter struct {
    agent        shared.Agent
    flowRegistry flows.FlowRegistry
    logger       logger.LoggerService
    tools        []gollem.Tool
}

// NewAgentAdapter erstellt den Adapter mit DI-Abhängigkeiten
func NewAgentAdapter(injector do.Injector) *AgentAdapter

// Implementiert acp.Agent Interface:
// - Initialize(ctx, *acp.InitializeRequest) (*acp.InitializeResponse, error)
// - CreateSession(ctx, *acp.NewRequestSessionRequest) (*acp.NewRequestSessionResponse, error)
// - Prompt(ctx, *acp.PromptRequest) (*acp.PromptResponse, error)
// - CloseSession(ctx, *acp.CloseSessionRequest) (*acp.CloseSessionResponse, error)
// - etc.
```

#### FlowRegistry Erweiterung

```go
package registry

type FlowInfo struct {
    Name         string
    Description  string
    Version      string
    InputFields  []FieldInfo
    OutputFields []FieldInfo
    States       []string
}

type FieldInfo struct {
    Name     string
    Type     string
    Required bool
}

type FlowRegistry interface {
    // ... existing methods ...
    ListFlows() ([]*FlowInfo, error)
    GetFlowInfo(name string) (*FlowInfo, error)
}
```

### Tools (gollem.Tools)

#### execute_flow Tool

```go
type ExecuteFlowTool struct {
    flowRegistry flows.FlowRegistry
    executor     flows.Executor
    logger       logger.LoggerService
}

// Spec(): Tool-Definition mit Parametern (flowName, inputs)
// Run(): Führt Flow aus, gibt Outputs zurück
```

#### list_flows Tool

```go
type ListFlowsTool struct {
    flowRegistry flows.FlowRegistry
    logger       logger.LoggerService
}

// Spec(): Tool-Definition ohne Parameter
// Run(): Gibt Liste aller Flows mit Schema zurück
```

## Important: Tool Model in ACP

**Tools are NOT registered in Initialize.** Instead:

1. **ToolCalls are SessionUpdates** – Agent sends ToolCalls during execution
2. **Client displays ToolCalls** – But doesn't ask for "available tools"
3. **Tools discovered at runtime** – Through ToolCalls sent by agent

`executeFlow` and `listFlows` are **gollem.Tools** used by `shared.Agent`, NOT ACP-specific tools. The ACP adapter just sends ToolCalls as notifications.

## Error Handling

```go
// ACP-spezifische Fehler
acpErrorToACP := func(err error) *acp.Error {
    switch {
    case errors.Is(err, flows.ErrFlowNotFound):
        return acp.ErrInvalidParams(fmt.Sprintf("flow not found: %v", err))
    case errors.Is(err, context.Canceled):
        return acp.ErrCancelled
    default:
        return acp.ErrInternal(fmt.Sprintf("execution failed: %v", err))
    }
}
```

## Testing Strategy

1. **Unit Tests** für Tools (ExecuteFlow, ListFlows)
2. **Integration Tests** für AcpService (Initialize, Prompt)
3. **E2E Tests** mit ACP-Client
4. **Contract Tests** für DI

## Implementation Phases

| Phase | Description | Files |
|-------|-------------|-------|
| 1 | FlowRegistry Erweiterung | `pkg/flows/registry/service.go` |
| 2 | Gollum Tools | `pkg/tools/flow_tools.go` |
| 3 | ACP Service | `pkg/acp/*.go` |
| 4 | CLI Command | `pkg/cli/acp.go` |
| 5 | DI Integration | `pkg/di/container.go` |
| 6 | Go Mod Dependency | `go.mod` |

## Dependencies

```go
// go.mod
require github.com/ironpark/acp-go v<version>
```

## References

- [ACP Official Repository](https://github.com/zed-industries/agent-client-protocol)
- [ACP Go Implementation](https://github.com/ironpark/acp-go)
- [ACP Documentation](https://agentclientprotocol.com)
