package registry

import (

	"context"
	"github.com/denkhaus/gollum/pkg/logger"
	"testing"
	"time"

	appconfig "github.com/denkhaus/gollum/pkg/config"
	mcpconfig "github.com/denkhaus/gollum/pkg/mcp/config"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/mcp"
	"go.uber.org/mock/gomock"

)

func TestMCPRegistry_CreateStdioClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Test with invalid command to verify graceful failure
	// (echo is not a valid MCP server)
	loader := &mockConfigLoader{
		configs: map[string]mcpconfig.MCPServerConfig{
			"invalid-server": {
				Command: "echo",
				Args:    []string{"not an mcp server"},
				Enabled: true,
			},
		},
	}

	registry, err := NewMCPRegistryWithLoader(t, ctrl, loader)
	if err != nil {
		t.Fatalf("NewMCPRegistryWithLoader() error = %v", err)
	}
	defer func() { _ = registry.Close() }()

	// Invalid servers should be skipped, returning 0 ToolSets
	toolSets := registry.GetToolSets()
	if len(toolSets) != 0 {
		t.Errorf("GetToolSets() returned %d ToolSets, want 0 (invalid server skipped)", len(toolSets))
	}
}

func TestMCPRegistry_CreateSSEClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Test SSE client creation with invalid URL (graceful failure)
	loader := &mockConfigLoader{
		configs: map[string]mcpconfig.MCPServerConfig{
			"invalid-sse": {
				Type:    "sse",
				URL:     "http://127.0.0.1:9999/mcp", // Use IP to avoid slow DNS lookup
				Enabled: true,
			},
		},
	}

	registry, err := NewMCPRegistryWithLoader(t, ctrl, loader)
	if err != nil {
		t.Fatalf("NewMCPRegistryWithLoader() error = %v", err)
	}
	defer func() { _ = registry.Close() }()

	// Invalid SSE servers should be skipped
	toolSets := registry.GetToolSets()
	if len(toolSets) != 0 {
		t.Errorf("GetToolSets() returned %d ToolSets, want 0 (invalid SSE server skipped)", len(toolSets))
	}
}

func TestMCPRegistry_CreateHTTPClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Test HTTP client creation with invalid URL (graceful failure)
	loader := &mockConfigLoader{
		configs: map[string]mcpconfig.MCPServerConfig{
			"invalid-http": {
				Type:    "http",
				URL:     "http://127.0.0.1:9999/mcp", // Use IP to avoid slow DNS lookup
				Enabled: true,
			},
		},
	}

	registry, err := NewMCPRegistryWithLoader(t, ctrl, loader)
	if err != nil {
		t.Fatalf("NewMCPRegistryWithLoader() error = %v", err)
	}
	defer func() { _ = registry.Close() }()

	// Invalid HTTP servers should be skipped
	toolSets := registry.GetToolSets()
	if len(toolSets) != 0 {
		t.Errorf("GetToolSets() returned %d ToolSets, want 0 (invalid HTTP server skipped)", len(toolSets))
	}
}

func TestMCPRegistry_UnsupportedType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Test unsupported MCP type
	loader := &mockConfigLoader{
		configs: map[string]mcpconfig.MCPServerConfig{
			"unsupported-type": {
				Type:    "unsupported",
				Enabled: true,
			},
		},
	}

	registry, err := NewMCPRegistryWithLoader(t, ctrl, loader)
	if err != nil {
		t.Fatalf("NewMCPRegistryWithLoader() error = %v", err)
	}
	defer func() { _ = registry.Close() }()

	// Unsupported type should be skipped
	toolSets := registry.GetToolSets()
	if len(toolSets) != 0 {
		t.Errorf("GetToolSets() returned %d ToolSets, want 0 (unsupported type skipped)", len(toolSets))
	}
}

// mockConfigLoader is a test double
type mockConfigLoader struct {
	configs map[string]mcpconfig.MCPServerConfig
}

func (m *mockConfigLoader) Load() (map[string]mcpconfig.MCPServerConfig, error) {
	return m.configs, nil
}

func (m *mockConfigLoader) LoadAll() (map[string]mcpconfig.MCPServerConfig, error) {
	return m.configs, nil
}

func (m *mockConfigLoader) GetDefaultAllowedSystemEnv() []string {
	return []string{"PATH", "HOME", "USER", "LOGNAME", "SHELL", "LANG", "LC_ALL", "LC_CTYPE", "TERM", "NODE", "NODE_PATH"}
}

// NewMCPRegistryWithLoader creates registry with custom loader for testing.
// Accepts *testing.T and *gomock.Controller for proper mock management.
func NewMCPRegistryWithLoader(t *testing.T, ctrl *gomock.Controller, loader mcpconfig.ConfigLoader) (MCPRegistry, error) {
	t.Helper()

	mockLog := logger.NewMockLoggerService(ctrl)
	mockAppConfig := appconfig.NewMockConfigService(ctrl)
	// Set up expectations for warnings, debug logs, info logs, and config access
	// Warn can be called with 2 args (error summary) or 5+ args (failed server details)
	mockLog.EXPECT().Warn(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLog.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()
	mockLog.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLog.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
	mockAppConfig.EXPECT().GetMCPConfig().Return(&appconfig.MCPConfig{
		CommandTimeoutSeconds:    30,
		ClientInitTimeoutSeconds: 60, // 60 second timeout for tests
	}).AnyTimes()

	p := &mcpRegistryImpl{
		clients:   make(map[string]*mcp.Client),
		tools:    make([]gollem.ToolSet, 0),
		loader:    loader,
		appConfig: mockAppConfig,
		logger:    mockLog,
		sem:       make(chan struct{}, 5), // Initialize semaphore for tests
	}
	// Initialize clients for testing with timeout
	// Short timeout to fail fast on invalid URLs in parallel tests
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := p.initializeClients(ctx); err != nil {
		return nil, err
	}
	return p, nil
}
