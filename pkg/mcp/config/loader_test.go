package config

import (

	"github.com/denkhaus/gollum/pkg/logger"
	"os"
	"path/filepath"
	"testing"
	"time"

	appconfig "github.com/denkhaus/gollum/pkg/config"
	"go.uber.org/mock/gomock"

)

func TestConfigLoader_Load_ValidConfig(t *testing.T) {
	// Create temp directory with mcp.json
	tmpDir := t.TempDir()
	configContent := `{
        "mcpServers": {
            "test-server": {
                "command": "echo",
                "args": ["hello"],
                "enabled": true
            }
        }
    }`
	configPath := filepath.Join(tmpDir, "mcp.json")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewConfigLoaderForTest(tmpDir, "")
	result, err := loader.Load()

	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(result) != 1 {
		t.Errorf("Load() returned %d servers, want 1", len(result))
	}

	server := result["test-server"]
	if server.Command != "echo" {
		t.Errorf("Command = %s, want echo", server.Command)
	}
}

// mockConfigService is a test double for appconfig.ConfigService
type mockConfigService struct{}

func (m *mockConfigService) GetLogLevel() string {
	return "info"
}

func (m *mockConfigService) IsDevMode() bool {
	return false
}

func (m *mockConfigService) GetAnthropicConfig() *appconfig.AnthropicConfig {
	return &appconfig.AnthropicConfig{}
}

func (m *mockConfigService) GetGeminiConfig() *appconfig.GeminiConfig {
	return &appconfig.GeminiConfig{}
}

func (m *mockConfigService) GetOpenAIConfig() *appconfig.OpenAIConfig {
	return &appconfig.OpenAIConfig{}
}

func (m *mockConfigService) GetAgentLimits() *appconfig.AgentLimitsConfig {
	return &appconfig.AgentLimitsConfig{}
}

func (m *mockConfigService) GetFilesConfig() *appconfig.FilesConfig {
	return &appconfig.FilesConfig{}
}

func (m *mockConfigService) GetLoggingConfig() *appconfig.LoggingConfig {
	return &appconfig.LoggingConfig{}
}

func (m *mockConfigService) GetBashConfig() *appconfig.BashConfig {
	return &appconfig.BashConfig{}
}

func (m *mockConfigService) GetHooksConfig() *appconfig.HooksConfig {
	return &appconfig.HooksConfig{}
}

func (m *mockConfigService) GetPromptStoreConfig() *appconfig.PromptStoreConfig {
	return &appconfig.PromptStoreConfig{}
}

func (m *mockConfigService) GetPromptOptimizerConfig() *appconfig.PromptOptimizerConfig {
	return &appconfig.PromptOptimizerConfig{}
}

func (m *mockConfigService) GetLangfuseConfig() *appconfig.LangfuseConfig {
	return &appconfig.LangfuseConfig{}
}

func (m *mockConfigService) GetEventsConfig() *appconfig.EventsConfig {
	return &appconfig.EventsConfig{}
}

func (m *mockConfigService) GetMCPConfig() *appconfig.MCPConfig {
	// Return default MCP config with 30-second timeout for tests
	return &appconfig.MCPConfig{
		CommandTimeoutSeconds: 30,
	}
}

// NewConfigLoaderForTest creates a ConfigLoader for testing with custom paths
// projectDir and globalDir are directories (will append mcp.json)
func NewConfigLoaderForTest(projectDir, globalDir string) ConfigLoader {
	projectPath := ""
	globalPath := ""

	if projectDir != "" {
		projectPath = filepath.Join(projectDir, "mcp.json")
	}
	if globalDir != "" {
		globalPath = filepath.Join(globalDir, "mcp.json")
	}

	ctrl := gomock.NewController(&testing.T{})
	mockLog := logger.NewMockLoggerService(ctrl)

	loader := &configLoaderImpl{
		projectPath: projectPath,
		globalPath:  globalPath,
		appConfig:   &mockConfigService{},
		logger:      mockLog,
	}
	return loader
}

// TestGetCommandTimeout validates the MCP config timeout helper
func TestGetCommandTimeout(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected time.Duration
	}{
		{"default", 30, 30 * time.Second}, // New default
		{"minimum", 0, 1 * time.Second},   // Below min gets clamped
		{"one second", 1, 1 * time.Second},
		{"ten seconds", 10, 10 * time.Second},
		{"gopass-friendly", 45, 45 * time.Second}, // Enough time for gopass
		{"maximum", 400, 300 * time.Second},       // Above max (300) gets clamped
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &appconfig.MCPConfig{CommandTimeoutSeconds: tt.input}
			result := cfg.GetCommandTimeout()
			if result != tt.expected {
				t.Errorf("GetCommandTimeout() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfigLoader_Load_MissingFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Use paths that definitely don't exist
	ctrl := gomock.NewController(t)
	mockLog := logger.NewMockLoggerService(ctrl)
	loader := &configLoaderImpl{
		projectPath: filepath.Join(tmpDir, "nonexistent-project-mcp.json"),
		globalPath:  filepath.Join(tmpDir, "nonexistent-global-mcp.json"),
		appConfig:   &mockConfigService{},
		logger:      mockLog,
	}

	result, err := loader.Load()

	if err != nil {
		t.Fatalf("Load() should not error on missing files, got: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Load() returned %d servers, want 0 (empty config)", len(result))
	}
}

func TestConfigLoader_Load_ProjectOverridesGlobal(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()

	// Create global config
	globalContent := `{
        "mcpServers": {
            "shared-server": {
                "command": "global-cmd",
                "args": ["global-arg"],
                "enabled": true
            },
            "global-only": {
                "command": "global-only-cmd",
                "enabled": true
            }
        }
    }`
	globalPath := filepath.Join(globalDir, "mcp.json")
	_ = os.WriteFile(globalPath, []byte(globalContent), 0644)

	// Create project config that overrides shared-server
	projectContent := `{
        "mcpServers": {
            "shared-server": {
                "command": "project-cmd",
                "args": ["project-arg"],
                "enabled": true
            },
            "project-only": {
                "command": "project-only-cmd",
                "enabled": true
            }
        }
    }`
	projectPath := filepath.Join(projectDir, "mcp.json")
	_ = os.WriteFile(projectPath, []byte(projectContent), 0644)

	loader := NewConfigLoaderForTest(projectDir, globalDir)
	result, err := loader.Load()

	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Should have 3 servers: shared-server (overridden), global-only, project-only
	if len(result) != 3 {
		t.Errorf("Load() returned %d servers, want 3", len(result))
	}

	// shared-server should have project values (project overrides global)
	shared := result["shared-server"]
	if shared.Command != "project-cmd" {
		t.Errorf("shared-server.Command = %s, want project-cmd", shared.Command)
	}

	// global-only should exist
	if result["global-only"].Command != "global-only-cmd" {
		t.Errorf("global-only.Command = %s, want global-only-cmd", result["global-only"].Command)
	}

	// project-only should exist
	if result["project-only"].Command != "project-only-cmd" {
		t.Errorf("project-only.Command = %s, want project-only-cmd", result["project-only"].Command)
	}
}

func TestConfigLoader_Load_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	invalidContent := `{"invalid json content`

	configPath := filepath.Join(tmpDir, "mcp.json")
	_ = os.WriteFile(configPath, []byte(invalidContent), 0644)

	loader := NewConfigLoaderForTest(tmpDir, "")
	_, err := loader.Load()

	if err == nil {
		t.Fatal("Load() should return error for invalid JSON")
	}
}

func TestConfigLoader_Load_EnabledFiltering(t *testing.T) {
	tmpDir := t.TempDir()
	configContent := `{
        "mcpServers": {
            "enabled-server": {
                "command": "enabled",
                "enabled": true
            },
            "disabled-server": {
                "command": "disabled",
                "enabled": false
            },
            "default-server": {
                "command": "default"
            }
        }
    }`
	configPath := filepath.Join(tmpDir, "mcp.json")
	_ = os.WriteFile(configPath, []byte(configContent), 0644)

	loader := NewConfigLoaderForTest(tmpDir, "")
	result, err := loader.Load()

	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Only enabled servers should be included
	if len(result) != 2 {
		t.Errorf("Load() returned %d servers, want 2 (enabled-server and default-server)", len(result))
	}

	if _, exists := result["enabled-server"]; !exists {
		t.Error("enabled-server should be in result")
	}

	if _, exists := result["disabled-server"]; exists {
		t.Error("disabled-server should NOT be in result")
	}

	if _, exists := result["default-server"]; !exists {
		t.Error("default-server should be in result (enabled=true by default)")
	}
}
