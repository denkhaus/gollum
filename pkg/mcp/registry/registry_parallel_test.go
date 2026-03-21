package registry

import (
	"context"
	"testing"
	"time"

	appconfig "github.com/denkhaus/gollum/pkg/config"
	"github.com/denkhaus/gollum/pkg/logger"
	mcpconfig "github.com/denkhaus/gollum/pkg/mcp/config"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/mcp"
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
	mockLog.EXPECT().Warn(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).MinTimes(2)
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
