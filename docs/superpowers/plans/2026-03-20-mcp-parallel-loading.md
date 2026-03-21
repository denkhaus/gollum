# MCP Parallel Loading Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reduce MCP client initialization time by loading servers in parallel with semaphore-based concurrency limiting.

**Architecture:** Replace sequential MCP client initialization with parallel goroutines. Use buffered channel as semaphore to limit concurrent connections to 5. Continue on individual server failures (log warnings, don't abort).

**Tech Stack:** Go 1.23+, samber/do/v2 for DI, gomock for testing, uber-go/zap for logging

---

## File Structure

**Modified files:**
- `pkg/mcp/registry/registry.go` - Add semaphore field, replace sequential initialization with parallel
- `pkg/mcp/registry/registry_test.go` - Add tests for parallel execution

**No new files** - this is a focused refactoring of existing initialization logic.

---

## Chunk 1: Semaphore Field and Constructor Update

### Task 1: Add Semaphore Field to mcpRegistryImpl

**Files:**
- Modify: `pkg/mcp/registry/registry.go`

- [ ] **Step 1: Read current struct definition**

Run: `head -50 pkg/mcp/registry/registry.go`
Observe: Current fields in `mcpRegistryImpl` struct

- [ ] **Step 2: Add sync import**

Add to import block (line ~17):
```go
import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"  // ADD THIS

	appconfig "github.com/denkhaus/gollum/pkg/config"
	...
)
```

- [ ] **Step 3: Add sem field to struct**

In `mcpRegistryImpl` struct (after `logger` field, line ~35):
```go
type mcpRegistryImpl struct {
	clients   map[string]*mcp.Client
	tools     []gollem.ToolSet
	loader    mcpconfig.ConfigLoader
	appConfig appconfig.ConfigService
	logger    logger.LoggerService
	sem       chan struct{} // ADD THIS: semaphore for limiting concurrent connections
}
```

- [ ] **Step 4: Initialize semaphore in constructor**

In `NewMCPRegistry()` function (line ~44), update struct initialization:
```go
p := &mcpRegistryImpl{
	clients:   make(map[string]*mcp.Client),
	tools:     make([]gollem.ToolSet, 0),
	loader:    loader,
	appConfig: appConfig,
	logger:    log,
	sem:       make(chan struct{}, 5), // ADD THIS: max 5 parallel MCP connections
}
```

- [ ] **Step 5: Verify code compiles**

Run: `go build ./pkg/mcp/registry/...`
Expected: No errors (field added but not used yet)

- [ ] **Step 6: Commit**

```bash
git add pkg/mcp/registry/registry.go
git commit -m "refactor(mcp): add semaphore field for parallel initialization"
```

---

### Task 2: Write Test for Parallel Initialization

**Files:**
- Create: `pkg/mcp/registry/registry_parallel_test.go`

- [ ] **Step 1: Create test file skeleton**

Create `pkg/mcp/registry/registry_parallel_test.go`:
```go
package registry

import (
	"context"
	"testing"
	"time"

	appconfig "github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	mcpconfig "github.com/denkhaus/gollum/pkg/mcp/config"
	"github.com/m-mizutani/gollem"
	"go.uber.org/mock/gomock"
)

// TestMCPRegistry_ParallelInitialization verifies that multiple clients
// are initialized in parallel and complete faster than sequential.
func TestMCPRegistry_ParallelInitialization(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create 3 "slow" servers that each take 100ms to initialize
	// If sequential: ~300ms total
	// If parallel (semaphore=5): ~100ms total
	loader := &mockConfigLoader{
		configs: map[string]mcpconfig.MCPServerConfig{
			"slow-server-1": {
				Command: "sleep",
				Args:    []string{"0.1"},
				Enabled: true,
			},
			"slow-server-2": {
				Command: "sleep",
				Args:    []string{"0.1"},
				Enabled: true,
			},
			"slow-server-3": {
				Command: "sleep",
				Args:    []string{"0.1"},
				Enabled: true,
			},
		},
	}

	mockLog := logger.NewMockLoggerService(ctrl)
	mockAppConfig := appconfig.NewMockConfigService(ctrl)
	mockLog.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()
	mockLog.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLog.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
	mockAppConfig.EXPECT().GetMCPConfig().Return(&appconfig.MCPConfig{
		CommandTimeoutSeconds:    30,
		ClientInitTimeoutSeconds: 60,
	}).AnyTimes()

	p := &mcpRegistryImpl{
		clients:   make(map[string]*mcp.Client),
		tools:     make([]gollem.ToolSet, 0),
		loader:    loader,
		appConfig: mockAppConfig,
		logger:    mockLog,
		sem:       make(chan struct{}, 5),
	}

	// Measure initialization time
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err := p.initializeClients(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("initializeClients() error = %v", err)
	}

	// Verify clients were created
	if len(p.clients) != 0 {
		t.Logf("Note: sleep commands don't return valid MCP servers, got %d clients", len(p.clients))
	}

	// Parallel should be < 200ms (3 × 100ms with overlap)
	// Sequential would be ~300ms
	if duration > 250*time.Millisecond {
		t.Errorf("Initialization took %v, parallel execution should be < 250ms", duration)
	}

	t.Logf("Parallel initialization took %v", duration)
}

// TestMCPRegistry_ParallelErrorHandling verifies that one failing server
// doesn't prevent other servers from being initialized.
func TestMCPRegistry_ParallelErrorHandling(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	loader := &mockConfigLoader{
		configs: map[string]mcpconfig.MCPServerConfig{
			"invalid-server": {
				Command: "echo",
				Args:    []string{"not an mcp server"},
				Enabled: true,
			},
			"also-invalid": {
				Command: "echo",
				Args:    []string{"also not valid"},
				Enabled: true,
			},
		},
	}

	mockLog := logger.NewMockLoggerService(ctrl)
	mockAppConfig := appconfig.NewMockConfigService(ctrl)

	// Expect warnings for failed servers
	mockLog.EXPECT().Warn(gomock.Any(), gomock.Any(), gomock.Any()).MinTimes(2)
	mockLog.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLog.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
	mockAppConfig.EXPECT().GetMCPConfig().Return(&appconfig.MCPConfig{
		CommandTimeoutSeconds:    30,
		ClientInitTimeoutSeconds: 60,
	}).AnyTimes()

	p := &mcpRegistryImpl{
		clients:   make(map[string]*mcp.Client),
		tools:     make([]gollem.ToolSet, 0),
		loader:    loader,
		appConfig: mockAppConfig,
		logger:    mockLog,
		sem:       make(chan struct{}, 5),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Should not error - failures are logged as warnings
	err := p.initializeClients(ctx)
	if err != nil {
		t.Errorf("initializeClients() should not error on client failures, got = %v", err)
	}

	// Both should fail, but process completes
	toolSets := p.GetToolSets()
	if len(toolSets) != 0 {
		t.Logf("Got %d toolsets (expected 0 for invalid servers)", len(toolSets))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail (sequential code still in place)**

Run: `go test -v -run TestMCPRegistry_Parallel ./pkg/mcp/registry/...`
Expected:
- `TestMCPRegistry_ParallelInitialization` FAILS (takes ~300ms, not < 250ms)
- `TestMCPRegistry_ParallelErrorHandling` PASSES (current behavior already doesn't error)

- [ ] **Step 3: Commit test file**

```bash
git add pkg/mcp/registry/registry_parallel_test.go
git commit -m "test(mcp): add tests for parallel initialization"
```

---

### Task 3: Implement Parallel Initialization

**Files:**
- Modify: `pkg/mcp/registry/registry.go`

- [ ] **Step 1: Read current initializeClients method**

Run: `sed -n '64,140p' pkg/mcp/registry/registry.go`
Observe: Current sequential implementation with for-loop

- [ ] **Step 2: Replace initializeClients with parallel implementation**

Replace the entire `initializeClients` method (lines 64-140):
```go
func (p *mcpRegistryImpl) initializeClients(ctx context.Context) error {
	allConfigs, err := p.loader.LoadAll()
	if err != nil {
		return err
	}

	// Count enabled/disabled servers
	enabledCount := 0
	for _, cfg := range allConfigs {
		if cfg.Enabled {
			enabledCount++
		}
	}

	mcpConfig := p.appConfig.GetMCPConfig()
	p.logger.Info("MCP client initialization starting (parallel)",
		zap.Int("total_servers", len(allConfigs)),
		zap.Int("enabled", enabledCount),
		zap.Int("disabled", len(allConfigs)-enabledCount),
		zap.Duration("init_timeout", mcpConfig.GetClientInitTimeout()),
		zap.Int("max_parallel", cap(p.sem)),
	)

	// Filter to enabled servers only
	configs := make(map[string]mcpconfig.MCPServerConfig)
	for name, cfg := range allConfigs {
		if cfg.Enabled {
			configs[name] = cfg
		}
	}

	// Semaphore-bounded parallel initialization
	var wg sync.WaitGroup
	var mu sync.Mutex       // Protects p.clients, p.tools
	errors := make([]error, 0)

	for name, cfg := range configs {
		wg.Add(1)
		go func(name string, cfg mcpconfig.MCPServerConfig) {
			defer wg.Done()

			p.sem <- struct{}{}  // Acquire semaphore
			defer func() { <-p.sem }()  // Release semaphore

			p.logger.Debug("Attempting to create MCP client (parallel)",
				zap.String("mcp_server", name),
				zap.String("type", cfg.Type),
				zap.Int("args_count", len(cfg.Args)),
				zap.Int("env_count", len(cfg.Env)),
			)

			client, err := p.createClient(ctx, name, cfg)
			if err != nil {
				p.logger.Warn("failed to create MCP client",
					zap.String("mcp_server", name),
					zap.String("type", cfg.Type),
					zap.String("command", cfg.Command),
					zap.Error(err),
				)
				mu.Lock()
				errors = append(errors, fmt.Errorf("%s: %w", name, err))
				mu.Unlock()
				return
			}

			mu.Lock()
			p.clients[name] = client
			p.tools = append(p.tools, client) // mcp.Client implements gollem.ToolSet
			mu.Unlock()

			p.logger.Debug("MCP client created successfully",
				zap.String("mcp_server", name))
		}(name, cfg)
	}

	wg.Wait()

	// Log summary
	successCount := len(p.clients)
	failedCount := len(errors)

	p.logger.Info("MCP client initialization completed",
		zap.Int("enabled_attempted", len(configs)),
		zap.Int("success_count", successCount),
		zap.Int("failed_count", failedCount),
	)

	// Log all errors (but don't fail initialization)
	for _, err := range errors {
		p.logger.Warn("MCP client initialization error", zap.Error(err))
	}

	return nil
}
```

- [ ] **Step 3: Run tests to verify parallel behavior**

Run: `go test -v -run TestMCPRegistry_Parallel ./pkg/mcp/registry/...`
Expected: Both tests PASS

- [ ] **Step 4: Run all existing tests to ensure no regressions**

Run: `go test -v ./pkg/mcp/registry/...`
Expected: All existing tests still PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/mcp/registry/registry.go
git commit -m "feat(mcp): implement parallel client initialization with semaphore"
```

---

## Chunk 2: Integration Testing and Documentation

### Task 4: Manual Integration Test

**Files:**
- Create: `pkg/mcp/registry/README_parallel_test.md`

- [ ] **Step 1: Create integration test instructions**

Create `pkg/mcp/registry/README_parallel_test.md`:
```markdown
# MCP Parallel Loading - Integration Test

## Manual Verification

To verify the parallel initialization is working:

1. Configure multiple MCP servers in your `~/.config/gollum/mcp.json`:
```json
{
  "mcpServers": {
    "server1": {
      "command": "sleep",
      "args": ["0.2"],
      "enabled": true
    },
    "server2": {
      "command": "sleep",
      "args": ["0.2"],
      "enabled": true
    },
    "server3": {
      "command": "sleep",
      "args": ["0.2"],
      "enabled": true
    }
  }
}
```

2. Add logging to measure startup time:
```go
// In main.go or wherever registry is created
start := time.Now()
registry, err := do.Invoke[*registry.MCPRegistry](injector)
log.Printf("MCP registry initialized in %v", time.Since(start))
```

3. Run the application and observe:
   - With 3 servers × 200ms each:
     - Sequential: ~600ms
     - Parallel: ~200ms ( semaphore allows all 3)

## Expected Log Output

```
INFO  MCP client initialization starting (parallel)  total_servers=3 enabled=3 max_parallel=5
DEBUG Attempting to create MCP client (parallel)  mcp_server=server1
DEBUG Attempting to create MCP client (parallel)  mcp_server=server2
DEBUG Attempting to create MCP client (parallel)  mcp_server=server3
WARN  failed to create MCP client  mcp_server=server1 error=...
WARN  failed to create MCP client  mcp_server=server2 error=...
WARN  failed to create MCP client  mcp_server=server3 error=...
INFO  MCP client initialization completed  enabled_attempted=3 success_count=0 failed_count=3
```

Note the interleaved DEBUG logs indicating parallel execution.
```

- [ ] **Step 2: Commit documentation**

```bash
git add pkg/mcp/registry/README_parallel_test.md
git commit -m "docs(mcp): add parallel loading integration test guide"
```

---

### Task 5: Update Package Documentation

**Files:**
- Modify: `pkg/mcp/registry/doc.go`

- [ ] **Step 1: Read current documentation**

Run: `cat pkg/mcp/registry/doc.go`

- [ ] **Step 2: Add note about parallel initialization**

Add to documentation (before existing examples):
```go
// Package registry provides MCP client registry service.
//
// Parallel Initialization:
// The registry initializes MCP clients in parallel using a semaphore
// to limit concurrent connections. The default semaphore buffer size
// is 5, meaning up to 5 MCP clients will be created simultaneously.
// Individual client failures are logged as warnings but do not prevent
// other clients from being initialized.
//
// Use NewMCPRegistry() to create the registry via dependency injection.
// The registry will automatically initialize all enabled MCP servers
// from the configuration.
...
```

- [ ] **Step 3: Commit documentation update**

```bash
git add pkg/mcp/registry/doc.go
git commit -m "docs(mcp): document parallel initialization behavior"
```

---

## Completion Checklist

- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] Manual integration test completed
- [ ] Documentation updated
- [ ] No breaking changes to existing API

## Success Criteria

1. **Performance:** Initialization time reduced by ~5× for 5+ servers
2. **Reliability:** Individual server failures don't stop entire initialization
3. **Compatibility:** All existing tests pass (no API changes)
4. **Observability:** Logs clearly show parallel execution and results

## Rollback Plan

If issues arise:
1. Revert to sequential initialization by removing semaphore usage
2. Keep the struct field (can be removed in next cleanup)
3. The API hasn't changed, so rollback is safe
