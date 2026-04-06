# ACP Integration Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Integrate Agent Client Protocol (ACP) into Gollum to enable code editors to directly control Gollum's agent loop and flows.

**Architecture:** AcpService implements acp.Agent interface, connects to existing shared.Agent.Execute(), with execute_flow and list_flows as gollem.Tools.

**Tech Stack:** github.com/ironpark/acp-go, samber/do/v2 (DI), existing gollem framework

---

## Chunk 1: FlowRegistry Extension

### Task 1: Add FlowInfo structs

**Files:**
- Modify: `pkg/flows/registry/service.go`

- [ ] **Step 1: Add FlowInfo and FieldInfo structs**

Add to `pkg/flows/registry/service.go`:

```go
// FlowInfo holds metadata about a flow for tool discovery
type FlowInfo struct {
    Name         string
    Description  string
    Version      string
    InputFields  []FieldInfo
    OutputFields []FieldInfo
    States       []string
}

// FieldInfo describes a flow field
type FieldInfo struct {
    Name     string
    Type     string
    Required bool
}
```

- [ ] **Step 2: Run tests to verify**

Run: `go test ./pkg/flows/registry/...`
Expected: PASS (no new tests yet)

- [ ] **Step 3: Commit**

```bash
git add pkg/flows/registry/service.go
git commit -m "feat(registry): add FlowInfo structs for tool discovery"
```

### Task 2: Implement ListFlows method

**Files:**
- Modify: `pkg/flows/registry/service.go`
- Test: `pkg/flows/registry/service_test.go`

- [ ] **Step 1: Write failing test for ListFlows**

Add to `pkg/flows/registry/service_test.go`:

```go
func TestFlowRegistryService_ListFlows_ReturnsAllFlows(t *testing.T) {
    svc := &flowRegistryServiceImpl{
        flows: make(map[string]*flows.Flow),
        logger: nil,
    }
    
    testFlow := &flows.Flow{
        Name:    "test-flow",
        Version: "1.0",
        States: []flows.State{
            {Name: "init", Initial: true},
        },
        InputFields: []flows.Field{
            {Name: "url", Type: "string", Required: true},
        },
        OutputFields: []flows.Field{
            {Name: "result", Type: "string"},
        },
    }
    svc.flows["test-flow"] = testFlow
    
    infos, err := svc.ListFlows()
    
    require.NoError(t, err)
    assert.Len(t, infos, 1)
    assert.Equal(t, "test-flow", infos[0].Name)
    assert.Equal(t, "1.0", infos[0].Version)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/registry/... -run TestFlowRegistryService_ListFlows -v`
Expected: FAIL with "method ListFlows not defined"

- [ ] **Step 3: Implement ListFlows method**

Add to `pkg/flows/registry/service.go`:

```go
// ListFlows returns information about all registered flows
func (s *flowRegistryServiceImpl) ListFlows() ([]*FlowInfo, error) {
    result := make([]*FlowInfo, 0, len(s.flows))
    for _, flow := range s.flows {
        info := s.flowToFlowInfo(flow)
        result = append(result, info)
    }
    return result, nil
}
```

- [ ] **Step 4: Add flowToFlowInfo helper**

```go
func (s *flowRegistryServiceImpl) flowToFlowInfo(flow *flows.Flow) *FlowInfo {
    inputFields := make([]FieldInfo, 0, len(flow.InputFields))
    for _, f := range flow.InputFields {
        inputFields = append(inputFields, FieldInfo{
            Name:     f.Name,
            Type:     f.Type,
            Required: f.Required,
        })
    }
    
    outputFields := make([]FieldInfo, 0, len(flow.OutputFields))
    for _, f := range flow.OutputFields {
        outputFields = append(outputFields, FieldInfo{
            Name: f.Name,
            Type: f.Type,
        })
    }
    
    states := make([]string, 0, len(flow.States))
    for _, s := range flow.States {
        states = append(states, s.Name)
    }
    
    return &FlowInfo{
        Name:         flow.Name,
        Description:  flow.Description,
        Version:      flow.Version,
        InputFields:  inputFields,
        OutputFields: outputFields,
        States:       states,
    }
}
```

- [ ] **Step 5: Update FlowRegistry interface**

```go
type FlowRegistry interface {
    Register(name string, flow *flows.Flow)
    GetFlow(ref string) (*flows.Flow, error)
    ListFlows() ([]*FlowInfo, error)  // NEW
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./pkg/flows/registry/... -run TestFlowRegistryService_ListFlows -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add pkg/flows/registry/service.go pkg/flows/registry/service_test.go
git commit -m "feat(registry): implement ListFlows method for tool discovery"
```

### Task 3: Implement GetFlowInfo method

**Files:**
- Modify: `pkg/flows/registry/service.go`
- Test: `pkg/flows/registry/service_test.go`

- [ ] **Step 1: Write failing test**

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/flows/registry/... -run TestFlowRegistryService_GetFlowInfo -v`

- [ ] **Step 3: Implement GetFlowInfo**

```go
func (s *flowRegistryServiceImpl) GetFlowInfo(name string) (*FlowInfo, error) {
    flow, err := s.GetFlow(name)
    if err != nil {
        return nil, err
    }
    return s.flowToFlowInfo(flow), nil
}
```

- [ ] **Step 4: Update interface**

- [ ] **Step 5: Run tests**

- [ ] **Step 6: Commit**

```bash
git add pkg/flows/registry/
git commit -m "feat(registry): implement GetFlowInfo method"
```

---

## Chunk 2: Gollum Flow Tools

### Task 4: Add tool name constants

**Files:**
- Modify: `pkg/shared/tool_names.go` (or where ToolName is defined)

- [ ] **Step 1: Add tool name constants**

```go
const (
    // ... existing ...
    ToolNameExecuteFlow ToolName = "execute_flow"
    ToolNameListFlows   ToolName = "list_flows"
)
```

- [ ] **Step 2: Run tests**

- [ ] **Step 3: Commit**

```bash
git add pkg/shared/tool_names.go
git commit -m "feat(tools): add execute_flow and list_flows tool names"
```

### Task 5: Implement ExecuteFlowTool

**Files:**
- Modify: `pkg/tools/flow_tools.go`
- Test: `pkg/tools/flow_tools_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestExecuteFlowTool_Run_Success(t *testing.T) {
    mockRegistry := &MockFlowRegistry{}
    mockExecutor := &MockExecutor{}
    mockLogger := &MockLoggerService{}
    
    tool := NewExecuteFlowTool(mockRegistry, mockExecutor, mockLogger)
    
    result, err := tool.Run(ctx, map[string]any{
        "flowName": "test-flow",
        "inputs": map[string]any{
            "url": "https://example.com",
        },
    })
    
    require.NoError(t, err)
    assert.NotNil(t, result["outputs"])
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/tools/... -run TestExecuteFlowTool -v`

- [ ] **Step 3: Implement ExecuteFlowTool**

Add to `pkg/tools/flow_tools.go`:

```go
type executeFlowTool struct {
    flowRegistry flows.FlowRegistry
    executor     flows.Executor
    logger       logger.LoggerService
}

func NewExecuteFlowTool(flowRegistry flows.FlowRegistry, executor flows.Executor, logger logger.LoggerService) gollem.Tool {
    return &executeFlowTool{
        flowRegistry: flowRegistry,
        executor:     executor,
        logger:       logger,
    }
}

func (t *executeFlowTool) Spec() gollem.ToolSpec {
    return gollem.ToolSpec{
        Name:        shared.ToolNameExecuteFlow.String(),
        Description: "Executes a Gollum flow by name with the provided input parameters",
        Parameters: map[string]*gollem.Parameter{
            "flowName": {
                Type:        gollem.TypeString,
                Description: "The name of the flow to execute",
            },
            "inputs": {
                Type:        gollem.TypeObject,
                Description: "Input parameters for the flow (field name -> value)",
            },
        },
    }
}

func (t *executeFlowTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
    flowName, ok := args["flowName"].(string)
    if !ok || flowName == "" {
        return nil, fmt.Errorf("flowName is required and must be a string")
    }
    
    inputs, _ := args["inputs"].(map[string]any)
    if inputs == nil {
        inputs = make(map[string]any)
    }
    
    flow, err := t.flowRegistry.GetFlow(flowName)
    if err != nil {
        return nil, fmt.Errorf("flow not found: %s", flowName)
    }
    
    // Execute flow
    result, err := t.executor.Execute(ctx, flow, inputs)
    if err != nil {
        return nil, fmt.Errorf("flow execution failed: %w", err)
    }
    
    return map[string]any{
        "outputs": result.Outputs,
    }, nil
}
```

- [ ] **Step 4: Run tests**

- [ ] **Step 5: Commit**

```bash
git add pkg/tools/flow_tools.go pkg/tools/flow_tools_test.go
git commit -m "feat(tools): implement ExecuteFlowTool"
```

### Task 6: Implement ListFlowsTool

Similar TDD cycle for ListFlowsTool.

---

## Chunk 3: ACP Service Infrastructure

### Task 7: Add ACP dependency

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Add dependency**

Run: `go get github.com/ironpark/acp-go@latest`

- [ ] **Step 2: Tidy**

Run: `go mod tidy`

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add github.com/ironpark/acp-go"
```

### Task 8: Create ACP session types

**Files:**
- Create: `pkg/acp/session.go`
- Test: `pkg/acp/session_test.go`

- [ ] **Step 1: Write failing test for AcpSession**

Add to `pkg/acp/session_test.go`:

```go
func TestAcpSession_NewSession_HasRequiredFields(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    session := NewAcpSession(ctx, cancel)

    assert.NotNil(t, session)
    assert.NotNil(t, session.Context)
    assert.NotNil(t, session.CancelFunc)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/acp/... -run TestAcpSession -v`
Expected: FAIL with "undefined: NewAcpSession"

- [ ] **Step 3: Implement AcpSession**

Create `pkg/acp/session.go`:

```go
package acp

import (
    "context"

    acppkg "github.com/ironpark/acp-go"
)

// AcpSession holds session state for ACP connections
type AcpSession struct {
    // Context for the current prompt/turn - can be cancelled
    Context context.Context
    // CancelFunc cancels the current turn
    CancelFunc context.CancelFunc
    // SessionID from ACP protocol
    SessionID acppkg.SessionID
}

// NewAcpSession creates a new session with cancellable context
func NewAcpSession(ctx context.Context, cancel context.CancelFunc) *AcpSession {
    return &AcpSession{
        Context:    ctx,
        CancelFunc: cancel,
    }
}
```

- [ ] **Step 4: Run tests**

- [ ] **Step 5: Commit**

```bash
git add pkg/acp/session.go pkg/acp/session_test.go
git commit -m "feat(acp): add AcpSession type for managing ACP session state"
```

### Task 9: Create ACP service agent implementation (DI-compliant)

**Files:**
- Modify: `pkg/generate.go` (add mockgen directive)
- Create: `pkg/acp/service.go`
- Test: `pkg/acp/service_test.go`

- [ ] **Step 1: Add go generate directive for ACP service**

Add to `pkg/generate.go` (in alphabetical order):

```go
//go:generate go run go.uber.org/mock/mockgen -source=acp/service.go -destination=acp/service_mock.go -package=acp github.com/denkhaus/gollum/pkg/acp Service
```

- [ ] **Step 2: Generate mocks**

Run: `go generate ./pkg/...`

- [ ] **Step 3: Commit generate file update**

```bash
git add pkg/generate.go pkg/acp/service_mock.go
git commit -m "feat(acp): add mock generation for ACP service"
```

- [ ] **Step 4: Write failing test for Initialize**

Add to `pkg/acp/service_test.go`:

```go
package acp

import (
    "context"
    "testing"

    acppkg "github.com/ironpark/acp-go"
    "github.com/samber/do/v2"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "go.uber.org/mock/gomock"
    
    "github.com/denkhaus/gollum/pkg/flows"
    "github.com/denkhaus/gollum/pkg/logger"
    "github.com/denkhaus/gollum/pkg/shared"
)

func TestAcpService_Initialize_ReturnsCorrectCapabilities(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockAgent := shared.NewMockAgent(ctrl)
    mockFlowRegistry := flows.NewMockFlowRegistry(ctrl)
    mockLogger := logger.NewMockLoggerService(ctrl)

    injector := do.New()
    do.ProvideValue[shared.Agent](injector, mockAgent)
    do.ProvideValue[flows.FlowRegistry](injector, mockFlowRegistry)
    do.ProvideValue[logger.LoggerService](injector, mockLogger)

    svc, err := NewAcpService(injector)
    require.NoError(t, err)

    resp, err := svc.Initialize(context.Background(), &acppkg.InitializeRequest{
        ProtocolVersion: "1",
    })

    require.NoError(t, err)
    assert.Equal(t, acppkg.ProtocolVersion(acppkg.CurrentProtocolVersion), resp.ProtocolVersion)
    assert.NotNil(t, resp.AgentCapabilities)
    assert.False(t, resp.AgentCapabilities.LoadSession)
}
```

- [ ] **Step 5: Run test to verify it fails**

- [ ] **Step 6: Implement AcpService interface and implementation**

Create `pkg/acp/service.go`:

```go
package acp

import (
    "context"
    "fmt"

    acppkg "github.com/ironpark/acp-go"
    "github.com/samber/do/v2"
    "github.com/denkhaus/gollum/pkg/flows"
    "github.com/denkhaus/gollum/pkg/logger"
    "github.com/denkhaus/gollum/pkg/shared"
)

// Service defines the public interface for ACP agent operations
type Service interface {
    // acp.Agent interface methods
    Initialize(ctx context.Context, params *acppkg.InitializeRequest) (*acppkg.InitializeResponse, error)
    Authenticate(ctx context.Context, params *acppkg.AuthenticateRequest) (*acppkg.AuthenticateResponse, error)
    SetSessionMode(ctx context.Context, params *acppkg.SetSessionModeRequest) (*acppkg.SetSessionModeResponse, error)
    SetSessionConfigOption(ctx context.Context, params *acppkg.SetSessionConfigOptionRequest) (*acppkg.SetSessionConfigOptionResponse, error)
    Prompt(ctx context.Context, params *acppkg.PromptRequest) (*acppkg.PromptResponse, error)
    Cancel(ctx context.Context, params *acppkg.CancelNotification) error
    
    // ACP-specific methods
    SetClient(client acppkg.Client)
    SetSessionStore(store acppkg.SessionStore[*AcpSession])
}

// acpServiceImpl implements the Service interface (PRIVATE)
type acpServiceImpl struct {
    agent        shared.Agent
    flowRegistry flows.FlowRegistry
    logger       logger.LoggerService
    client       acppkg.Client
    store        acppkg.SessionStore[*AcpSession]
}

// Ensure acpServiceImpl implements Service at compile time
var _ Service = (*acpServiceImpl)(nil)

// NewAcpService creates a new ACP service via DI
// Constructor follows DI pattern: func NewService(injector do.Injector) (Service, error)
func NewAcpService(injector do.Injector) (Service, error) {
    agent := do.MustInvoke[shared.Agent](injector)
    flowRegistry := do.MustInvoke[flows.FlowRegistry](injector)
    logger := do.MustInvoke[logger.LoggerService](injector)

    logger.Debug("startup ACP service")

    return &acpServiceImpl{
        agent:        agent,
        flowRegistry: flowRegistry,
        logger:       logger,
    }, nil
}

// SetClient sets the ACP client (called by connection factory)
func (p *acpServiceImpl) SetClient(client acppkg.Client) {
    p.client = client
}

// SetSessionStore sets the session store (called by connection factory)
func (p *acpServiceImpl) SetSessionStore(store acppkg.SessionStore[*AcpSession]) {
    p.store = store
}

// Initialize implements acp.Agent.Initialize
func (p *acpServiceImpl) Initialize(ctx context.Context, params *acppkg.InitializeRequest) (*acppkg.InitializeResponse, error) {
    return &acppkg.InitializeResponse{
        ProtocolVersion: acppkg.ProtocolVersion(acppkg.CurrentProtocolVersion),
        AgentCapabilities: &acppkg.AgentCapabilities{
            LoadSession: false,
            MCPCapabilities: &acppkg.MCPCapabilities{
                HTTP: false,
                SSE:  false,
            },
            PromptCapabilities: &acppkg.PromptCapabilities{
                Audio:           false,
                EmbeddedContext: false,
                Image:           false,
            },
        },
        AuthMethods: []acppkg.AuthMethod{},
    }, nil
}

// Authenticate implements acp.Agent.Authenticate
func (p *acpServiceImpl) Authenticate(ctx context.Context, params *acppkg.AuthenticateRequest) (*acppkg.AuthenticateResponse, error) {
    return &acppkg.AuthenticateResponse{}, nil
}

// SetSessionMode implements acp.Agent.SetSessionMode
func (p *acpServiceImpl) SetSessionMode(ctx context.Context, params *acppkg.SetSessionModeRequest) (*acppkg.SetSessionModeResponse, error) {
    return &acppkg.SetSessionModeResponse{}, nil
}

// SetSessionConfigOption implements acp.Agent.SetSessionConfigOption
func (p *acpServiceImpl) SetSessionConfigOption(ctx context.Context, params *acppkg.SetSessionConfigOptionRequest) (*acppkg.SetSessionConfigOptionResponse, error) {
    return &acppkg.SetSessionConfigOptionResponse{ConfigOptions: []acppkg.SessionConfigOption{}}, nil
}

// Prompt implements acp.Agent.Prompt - core agent execution loop
func (p *acpServiceImpl) Prompt(ctx context.Context, params *acppkg.PromptRequest) (*acppkg.PromptResponse, error) {
    session, ok := p.store.Get(params.SessionID)
    if !ok {
        return nil, fmt.Errorf("session %s not found", params.SessionID)
    }

    session.CancelFunc()
    sessionCtx, cancelFunc := context.WithCancel(context.Background())
    session.Context = sessionCtx
    session.CancelFunc = cancelFunc

    err := p.executeTurn(sessionCtx, params.SessionID)
    if err != nil {
        if sessionCtx.Err() == context.Canceled {
            return &acppkg.PromptResponse{
                StopReason: acppkg.StopReasonCancelled,
            }, nil
        }
        return nil, err
    }

    return &acppkg.PromptResponse{
        StopReason: acppkg.StopReasonEndTurn,
    }, nil
}

// Cancel implements acp.Agent.Cancel
func (p *acpServiceImpl) Cancel(ctx context.Context, params *acppkg.CancelNotification) error {
    if session, ok := p.store.Get(params.SessionID); ok {
        session.CancelFunc()
    }
    return nil
}

// executeTurn runs the agent execution loop
func (p *acpServiceImpl) executeTurn(ctx context.Context, sessionID acppkg.SessionID) error {
    stream := acppkg.NewSessionStream(p.client, sessionID)
    
    // TODO: Integrate with shared.Agent.Execute()
    return stream.SendText(ctx, "Gollum ACP service ready")
}
```

- [ ] **Step 7: Run tests**

Run: `go test ./pkg/acp/... -v`

- [ ] **Step 8: Check coverage**

Run: `go test ./pkg/acp -coverprofile=/tmp/acp_coverage.out && go tool cover -func=/tmp/acp_coverage.out | tail -5`

Ensure coverage >= 80%

- [ ] **Step 9: Commit**

```bash
git add pkg/acp/service.go pkg/acp/service_test.go
git commit -m "feat(acp): implement ACP agent service with DI-compliant constructor"
```

### Task 10: Create ACP connection factory (DI-compliant)

**Files:**
- Create: `pkg/acp/connection.go`
- Test: `pkg/acp/connection_test.go`

- [ ] **Step 1: Add go generate directive for Connection**

Add to `pkg/generate.go`:

```go
//go:generate go run go.uber.org/mock/mockgen -source=acp/connection.go -destination=acp/connection_mock.go -package=acp github.com/denkhaus/gollum/pkg/acp Connection
```

- [ ] **Step 2: Generate mocks**

Run: `go generate ./pkg/...`

- [ ] **Step 3: Write failing test for NewConnection**

Add to `pkg/acp/connection_test.go`:

```go
func TestNewConnection_CreatesValidConnection(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockAgent := shared.NewMockAgent(ctrl)
    mockFlowRegistry := flows.NewMockFlowRegistry(ctrl)
    mockLogger := logger.NewMockLoggerService(ctrl)

    injector := do.New()
    do.ProvideValue[shared.Agent](injector, mockAgent)
    do.ProvideValue[flows.FlowRegistry](injector, mockFlowRegistry)
    do.ProvideValue[logger.LoggerService](injector, mockLogger)

    reader := bytes.NewReader([]byte{})
    writer := &bytes.Buffer{}

    conn := NewConnection(injector, reader, writer)

    assert.NotNil(t, conn)
    assert.NotNil(t, conn.Start)
    assert.NotNil(t, conn.Done)
}
```

- [ ] **Step 4: Run test to verify it fails**

- [ ] **Step 5: Implement Connection interface and factory**

Create `pkg/acp/connection.go`:

```go
package acp

import (
    "context"
    "io"

    acppkg "github.com/ironpark/acp-go"
    "github.com/samber/do/v2"
    "github.com/denkhaus/gollum/pkg/flows"
    "github.com/denkhaus/gollum/pkg/logger"
    "github.com/denkhaus/gollum/pkg/shared"
)

// Connection defines the ACP connection interface
type Connection interface {
    Start(ctx context.Context) error
    Close() error
    Done() <-chan struct{}
}

// connectionImpl implements Connection (PRIVATE)
type connectionImpl struct {
    conn *acppkg.AgentSideConnection
}

// Ensure connectionImpl implements Connection at compile time
var _ Connection = (*connectionImpl)(nil)

// NewConnection creates a new ACP connection with DI
func NewConnection(injector do.Injector, reader io.Reader, writer io.Writer) Connection {
    // Get dependencies via DI
    agent := do.MustInvoke[shared.Agent](injector)
    flowRegistry := do.MustInvoke[flows.FlowRegistry](injector)
    loggerService := do.MustInvoke[logger.LoggerService](injector)

    // Create session store
    store := acppkg.NewMemoryStore[*AcpSession]()

    // Create ACP service via its DI constructor
    acpService, err := NewAcpService(injector)
    if err != nil {
        panic(err)
    }

    // Set ACP-specific fields
    acpService.SetClient(nil) // Will be set after connection creation
    acpService.SetSessionStore(store)

    // Create connection with session store and middleware
    conn := acppkg.NewAgentSideConnection(acpService, reader, writer,
        acppkg.WithSessionStore(store, func(ctx context.Context, params *acppkg.NewSessionRequest) (acppkg.SessionID, *AcpSession, error) {
            ctx, cancel := context.WithCancel(context.Background())
            return acppkg.GenerateSessionID(), NewAcpSession(ctx, cancel), nil
        }),
        acppkg.WithMiddleware(acppkg.RecoveryMiddleware()),
    )

    // Set client on service
    acpService.SetClient(conn.Client())

    return &connectionImpl{conn: conn}
}

// Start starts the ACP connection
func (p *connectionImpl) Start(ctx context.Context) error {
    return p.conn.Start(ctx)
}

// Close closes the connection gracefully
func (p *connectionImpl) Close() error {
    return p.conn.Close()
}

// Done returns a channel that is closed when connection is done
func (p *connectionImpl) Done() <-chan struct{} {
    return p.conn.Done()
}
```

- [ ] **Step 6: Run tests**

- [ ] **Step 7: Check coverage**

Run: `go test ./pkg/acp -coverprofile=/tmp/acp_coverage.out && go tool cover -func=/tmp/acp_coverage.out | tail -5`

- [ ] **Step 8: Commit**

```bash
git add pkg/generate.go pkg/acp/connection.go pkg/acp/connection_mock.go pkg/acp/connection_test.go
git commit -m "feat(acp): add ACP connection factory with DI-compliant constructor"
```

---

## Chunk 4: CLI Command

### Task 11: Create ACP CLI command

**Files:**
- Create: `pkg/cli/acp.go`
- Modify: `pkg/cli/root.go`

- [ ] **Step 1: Write failing test for ACP command**

- [ ] **Step 2: Implement ACP CLI command**

Create `pkg/cli/acp.go`:

```go
package cli

import (
    "context"
    "os"
    "os/signal"
    "syscall"

    "github.com/spf13/cobra"
    "github.com/samber/do/v2"
    "github.com/denkhaus/gollum/pkg/acp"
)

// NewACPCommand creates the ACP server command
func NewACPCommand(injector do.Injector) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "acp",
        Short: "Start Gollum ACP server (Agent Client Protocol)",
        Long:  "Start an ACP server that allows code editors to control Gollum flows directly",
        RunE: func(cmd *cobra.Command, args []string) error {
            return runACPServer(cmd.Context(), injector)
        },
    }
    return cmd
}

func runACPServer(ctx context.Context, injector do.Injector) error {
    // Create ACP connection via factory
    conn := acp.NewConnection(injector, os.Stdin, os.Stdout)

    // Handle shutdown gracefully
    ctx, cancel := context.WithCancel(ctx)
    defer cancel()

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-sigCh
        cancel()
    }()

    // Start connection
    if err := conn.Start(ctx); err != nil {
        return err
    }

    // Wait for completion
    <-conn.Done()
    return nil
}
```

- [ ] **Step 3: Add to root command**

Modify `pkg/cli/root.go` to register the ACP command:

```go
// In NewRootCommand, add:
acpCmd := acp.NewACPCommand(injector)
rootCmd.AddCommand(acpCmd)
```

- [ ] **Step 4: Run tests**

- [ ] **Step 5: Commit**

```bash
git add pkg/cli/acp.go pkg/cli/root.go
git commit -m "feat(cli): add ACP server command"
```

---

## Chunk 5: DI Integration

### Task 12: Register ACP service in DI container

**Files:**
- Modify: `pkg/di/container.go`
- Test: `pkg/di/container_test.go`

- [ ] **Step 1: Write test for ACP service availability**

Add to `pkg/di/container_test.go`:

```go
func TestContainer_ACPService_IsRegistered(t *testing.T) {
    ctx := context.Background()
    container := NewContainer()
    _ = container.RegisterServices(ctx)

    injector := container.GetInjector()

    // Should be able to invoke ACP Service
    service := do.MustInvoke[acp.Service](injector)
    assert.NotNil(t, service)
}
```

- [ ] **Step 2: Update imports in container.go**

Add to `pkg/di/container.go` imports:

```go
import (
    // ... existing imports ...
    "github.com/denkhaus/gollum/pkg/acp"
)
```

- [ ] **Step 3: Register ACP service**

Add to `pkg/di/container.go` in `RegisterServices` method (after MCP services):

```go
    // ACP (Agent Client Protocol) services
    do.Provide(p.injector, acp.NewAcpService)
```

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/di/... -v`

- [ ] **Step 5: Check coverage**

- [ ] **Step 6: Commit**

```bash
git add pkg/di/container.go pkg/di/container_test.go
git commit -m "feat(di): register ACP service in DI container"
```

---

## Chunk 6: End-to-End Integration

### Task 13: Connect shared.Agent.Execute to ACP Prompt

**Files:**
- Modify: `pkg/acp/service.go`

The `executeTurn` method needs to integrate with `shared.Agent.Execute()`:

```go
func (s *acpService) executeTurn(ctx context.Context, sessionID acppkg.SessionID) error {
    stream := acppkg.NewSessionStream(s.client, sessionID)

    // Send initial status
    if err := stream.SendText(ctx, "Processing request..."); err != nil {
        return err
    }

    // Execute agent turn with available tools
    // TODO: Convert flow tools to ACP tool format
    result, err := s.agent.Execute(ctx, shared.ExecuteRequest{
        // Prompt from ACP client
        // Tools: execute_flow, list_flows
    })
    if err != nil {
        return err
    }

    // Send result back to client
    return stream.SendText(ctx, result.Content)
}
```

- [ ] **Step 1: Implement full Execute integration
- [ ] **Step 2: Test with ACP client
- [ ] **Step 3: Commit**

