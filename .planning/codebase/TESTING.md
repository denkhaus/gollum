# Testing Patterns

**Analysis Date:** 2026-01-31

## Test Framework

**Runner:**
- Go built-in `testing` package
- Test configuration via `go test -v`
- No custom test runner detected

**Mock Framework:**
- `go.uber.org/mock` v0.6.0
- Centralized mock generation in `pkg/mocks/generate.go`
- Interface-based mocking for all dependencies

**Assertion Library:**
- `github.com/stretchr/testify` v1.11.1
- Used for assertions and test helpers

**Run Commands:**
```bash
go test ./...              # Run all tests
go test -v ./pkg/tools    # Run tests with verbose output
go test -cover ./...      # Run with coverage report
go test -bench ./...      # Run benchmarks
```

## Test File Organization

**Location:**
- Co-located with source files (same directory)
- Separate integration test files with `_integration_test.go` suffix
- Spec tests with `*_spec_test.go` suffix

**Naming:**
- Unit tests: `*_test.go`
- Integration tests: `*_integration_test.go`
- Spec tests: `*_spec_test.go`

**Structure:**
```
pkg/
├── tools/
│   ├── bash.go          # Implementation
│   ├── bash_test.go     # Unit tests
│   ├── bash_spec_test.go # Specification tests
│   └── tools_test.go     # General tool tests
└── registry/
    ├── registry.go      # Implementation
    ├── registry_test.go # Unit tests
    └── integration_test.go # Integration tests
```

## Test Structure

### Unit Tests
```go
// Test naming follows pattern: Test[Component]_[Scenario]
func TestBashTool_Run_Success(t *testing.T) {
    // Setup
    injector := setupTestInjector()
    provider := do.MustInvoke[BashToolProvider](injector)
    tool := provider.CreateTool(uuid.New())

    // Test input
    args := map[string]any{
        "command": "echo 'hello world'",
    }

    // Execute
    result, err := tool.Run(context.Background(), args)

    // Assert
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }

    if success, ok := result["success"].(bool); !ok || !success {
        t.Errorf("Expected success=true, got %v", result["success"])
    }
}
```

### Specification Tests
```go
// Test naming follows pattern: [Component]_[SpecName]
func EditTool_Spec(t *testing.T) {
    // Test tool specification compliance
    // Verify tool spec matches expected format
    // Test parameter validation
}
```

### Test Setup Pattern
```go
// Centralized test injector setup
func setupTestInjector() do.Injector {
    // DI container for tests
    container := di.NewTestContainer()
    return container.GetInjector()
}

// Test helpers in shared_test.go
func setupTestContext() context.Context {
    return context.Background()
}
```

## Mocking

### Mock Generation
**Centralized Mocks:**
- Location: `pkg/mocks/generate.go`
- All mock generation commands in single file
- Run with: `go generate ./pkg/mocks/`

**Mock Pattern:**
```go
// Mock generated from interface
func TestTool_UseMock(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockRegistry := mocks.NewMockAgentRegistry(ctrl)
    // Setup mock expectations
    mockRegistry.EXPECT().GetAgent(gomock.Any()).Return(nil)

    // Test with mock
}
```

### Mock Usage Patterns
```go
// 1. Create mock controller
ctrl := gomock.NewController(t)
defer ctrl.Finish()

// 2. Create mock instance
mockService := mocks.NewMockLoggerService(ctrl)

// 3. Setup expectations
mockService.EXPECT().Error(gomock.Any(), gomock.Any()).Return(nil)

// 4. Test with mock
tool := &BashTool{
    logService: mockService,
    // other dependencies...
}
```

### What to Mock
- **External dependencies**: LLM clients, file systems, network calls
- **Complex services**: Logger, configuration, registry
- **Time-sensitive operations**: For predictable testing
- **Side-effecting operations**: Database writes, file modifications

### What NOT to Mock
- **Simple data structures**: Use real instances
- **Pure functions**: Test directly
- **Internal implementation details**: Test through public interface
- **Trivial dependencies**: Use real instances for simplicity

## Fixtures and Test Data

### Test Data Pattern
```go
// Shared test data
var testUUID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// Test data creators
func createTestAgentConfig() *shared.AgentConfig {
    return &shared.AgentConfig{
        ID:           uuid.New(),
        SystemPrompt: "test prompt",
        Role:         "TestAgent",
    }
}

// Test data constants
const (
    testCommand = "echo 'test'"
    testTimeout = 30
)
```

### Test Helpers
```go
// pkg/shared/shared_test.go
package shared

// Common test helpers
func createTestAgent() *MockAgent {
    // Helper for creating test agents
}
```

## Test Coverage

**Requirements:**
- No explicit coverage targets detected
- CI includes codecov reports (`.github/workflows/codecov.yml`)

**View Coverage:**
```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Test Types

### Unit Tests
**Scope:**
- Single function/method isolation
- Mock all external dependencies
- Fast execution (< 1ms typically)

**Pattern:**
- Test input → output transformation
- Error condition testing
- Edge case validation

### Integration Tests
**Scope:**
- Multiple components working together
- Real file system operations (when safe)
- End-to-end tool execution

**Location:**
- Separate `*_integration_test.go` files
- Run with `go test -integration` or similar flag

**Example:**
```go
func TestFileStateIntegration(t *testing.T) {
    // Use temporary directory
    tmpDir := t.TempDir()

    // Create real file state manager
    fsm := state.NewFileStateManager(tmpDir)

    // Test real file operations
}
```

### Spec Tests
**Purpose:**
- Verify tool specification compliance
- Parameter validation
- Expected behavior documentation

**Pattern:**
- Tool spec validation
- Parameter type checking
- Required parameter enforcement

## Common Patterns

### Async Testing
```go
func TestAsyncOperation(t *testing.T) {
    done := make(chan bool)

    go func() {
        // Async operation
        time.Sleep(100 * time.Millisecond)
        done <- true
    }()

    select {
    case <-done:
        // Success
    case <-time.After(1 * time.Second):
        t.Error("Operation timed out")
    }
}
```

### Concurrent Testing
```go
func TestConcurrentAccess(t *testing.T) {
    var wg sync.WaitGroup

    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            // Concurrent operation
            t.Logf("Goroutine %d", id)
        }(i)
    }

    wg.Wait()
}
```

### Error Testing
```go
func TestTool_ErrorConditions(t *testing.T) {
    tests := []struct {
        name        string
        input       map[string]any
        expectError bool
    }{
        {
            name:        "missing command",
            input:       map[string]any{},
            expectError: true,
        },
        {
            name:        "invalid timeout",
            input:       map[string]any{"command": "echo test", "timeout": -1},
            expectError: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Table-Driven Tests
```go
func TestBashToolVariousCommands(t *testing.T) {
    tests := []struct {
        name     string
        command  string
        expected string
    }{
        {"echo", "echo hello", "hello"},
        {"ls", "ls /etc", "passwd"}, // specific file
        {"pwd", "pwd", "/tmp"},     // or current dir
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Background Agent Testing
**Specialized tests** for background agent functionality:
- `background_agent_lifecycle_test.go`
- `background_agent_async_test.go`
- `background_agent_sync_test.go`
- `background_agent_concurrent_test.go`

---

*Testing analysis: 2026-01-31*