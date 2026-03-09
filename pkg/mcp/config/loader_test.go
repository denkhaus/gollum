package config

import (
	"os"
	"path/filepath"
	"testing"
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

	return &configLoaderImpl{
		projectPath: projectPath,
		globalPath:  globalPath,
	}
}

func TestConfigLoader_Load_MissingFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Use paths that definitely don't exist
	loader := &configLoaderImpl{
		projectPath: filepath.Join(tmpDir, "nonexistent-project-mcp.json"),
		globalPath:  filepath.Join(tmpDir, "nonexistent-global-mcp.json"),
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
	os.WriteFile(globalPath, []byte(globalContent), 0644)

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
	os.WriteFile(projectPath, []byte(projectContent), 0644)

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
	os.WriteFile(configPath, []byte(invalidContent), 0644)

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
	os.WriteFile(configPath, []byte(configContent), 0644)

	loader := NewConfigLoaderForTest(tmpDir, "")
	result, err := loader.Load()

	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Only enabled servers should be included
	if len(result) != 1 {
		t.Errorf("Load() returned %d servers, want 1 (only enabled)", len(result))
	}

	if _, exists := result["enabled-server"]; !exists {
		t.Error("enabled-server should be in result")
	}

	if _, exists := result["disabled-server"]; exists {
		t.Error("disabled-server should NOT be in result")
	}

	if _, exists := result["default-server"]; exists {
		t.Error("default-server should NOT be in result (enabled=false by default)")
	}
}
