package config

import (
	"os"
	"strings"
	"testing"
)

func TestInterpolateValue_EnvVars(t *testing.T) {
	// Set up test env vars
	os.Setenv("TEST_VAR", "test-value")
	os.Setenv("ANOTHER_VAR", "another-value")
	defer os.Unsetenv("TEST_VAR")
	defer os.Unsetenv("ANOTHER_VAR")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple var",
			input:    "$TEST_VAR",
			expected: "test-value",
		},
		{
			name:     "braced var",
			input:    "${TEST_VAR}",
			expected: "test-value",
		},
		{
			name:     "multiple vars",
			input:    "$TEST_VAR-${ANOTHER_VAR}",
			expected: "test-value-another-value",
		},
		{
			name:     "var with prefix",
			input:    "prefix-$TEST_VAR",
			expected: "prefix-test-value",
		},
		{
			name:     "var with suffix",
			input:    "$TEST_VAR-suffix",
			expected: "test-value-suffix",
		},
		{
			name:     "undefined var",
			input:    "$UNDEFINED_VAR",
			expected: "",
		},
		{
			name:     "no vars",
			input:    "plain-text",
			expected: "plain-text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := interpolateValue(tt.input)
			if result != tt.expected {
				t.Errorf("interpolateValue(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestInterpolateValue_ShellCommands(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string // Use contains for commands that may vary
	}{
		{
			name:     "echo command",
			input:    "$(echo hello)",
			contains: "hello",
		},
		{
			name:     "printf command",
			input:    "prefix-$(printf 'test')-suffix",
			contains: "prefix-test-suffix",
		},
		{
			name:     "command with pipe",
			input:    "$(echo 'test' | tr 't' 'T')",
			contains: "TesT",
		},
		{
			name:     "multiple commands",
			input:    "$(echo a)-$(echo b)",
			contains: "a-b",
		},
		{
			name:     "no commands",
			input:    "plain-text",
			contains: "plain-text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := interpolateValue(tt.input)
			if result != tt.contains && !strings.Contains(result, tt.contains) {
				t.Errorf("interpolateValue(%q) = %q, want to contain %q", tt.input, result, tt.contains)
			}
		})
	}
}

func TestInterpolateValue_Combined(t *testing.T) {
	os.Setenv("PREFIX", "pre")
	defer os.Unsetenv("PREFIX")

	result := interpolateValue("$PREFIX-$(echo hello)")
	expected := "pre-hello"
	if result != expected {
		t.Errorf("interpolateValue(%q) = %q, want %q", "$PREFIX-$(echo hello)", result, expected)
	}
}

func TestInterpolateEnvMap(t *testing.T) {
	os.Setenv("KEY1", "value1")
	os.Setenv("KEY2", "value2")
	defer os.Unsetenv("KEY1")
	defer os.Unsetenv("KEY2")

	input := map[string]string{
		"key1":     "$KEY1",
		"key2":     "prefix-$KEY2",
		"static":   "static-value",
		"command":  "$(echo test)",
		"combined": "$KEY1-$(echo world)",
	}

	result := interpolateEnvMap(input)

	expected := map[string]string{
		"key1":     "value1",
		"key2":     "prefix-value2",
		"static":   "static-value",
		"command":  "test",
		"combined": "value1-world",
	}

	for k, expectedVal := range expected {
		if result[k] != expectedVal {
			t.Errorf("interpolateEnvMap()[%q] = %q, want %q", k, result[k], expectedVal)
		}
	}
}

func TestInterpolateConfig(t *testing.T) {
	os.Setenv("API_KEY", "secret-key")
	defer os.Unsetenv("API_KEY")

	cfg := MCPServerConfig{
		Command: "test-command",
		Args:    []string{"arg1", "arg2"},
		Env: map[string]string{
			"KEY":       "$API_KEY",
			"STATIC":    "value",
			"CMD_OUTPUT": "$(echo output)",
		},
		Headers: map[string]string{
			"Auth": "$API_KEY",
		},
		Enabled: true,
	}

	result := interpolateConfig(cfg)

	if result.Env["KEY"] != "secret-key" {
		t.Errorf("Env[KEY] = %q, want 'secret-key'", result.Env["KEY"])
	}
	if result.Env["STATIC"] != "value" {
		t.Errorf("Env[STATIC] = %q, want 'value'", result.Env["STATIC"])
	}
	if result.Env["CMD_OUTPUT"] != "output" {
		t.Errorf("Env[CMD_OUTPUT] = %q, want 'output'", result.Env["CMD_OUTPUT"])
	}
	if result.Headers["Auth"] != "secret-key" {
		t.Errorf("Headers[Auth] = %q, want 'secret-key'", result.Headers["Auth"])
	}
}

func TestConfigLoader_Load_WithInterpolation(t *testing.T) {
	os.Setenv("TEST_TOKEN", "interpolated-token")
	defer os.Unsetenv("TEST_TOKEN")

	tmpDir := t.TempDir()
	configContent := `{
        "mcpServers": {
            "test-server": {
                "command": "echo",
                "args": ["hello"],
                "enabled": true,
                "env": {
                    "TOKEN": "$TEST_TOKEN",
                    "STATIC": "static-value",
                    "CMD": "$(echo cmd-output)"
                }
            }
        }
    }`
	configPath := tmpDir + "/mcp.json"
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewConfigLoaderForTest(tmpDir, "")
	result, err := loader.Load()

	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Load() returned %d servers, want 1", len(result))
	}

	server := result["test-server"]
	if server.Env["TOKEN"] != "interpolated-token" {
		t.Errorf("Env[TOKEN] = %q, want 'interpolated-token'", server.Env["TOKEN"])
	}
	if server.Env["STATIC"] != "static-value" {
		t.Errorf("Env[STATIC] = %q, want 'static-value'", server.Env["STATIC"])
	}
	if server.Env["CMD"] != "cmd-output" {
		t.Errorf("Env[CMD] = %q, want 'cmd-output'", server.Env["CMD"])
	}
}

func TestConfigLoader_Load_WithHeadersInterpolation(t *testing.T) {
	os.Setenv("AUTH_HEADER", "Bearer token123")
	defer os.Unsetenv("AUTH_HEADER")

	tmpDir := t.TempDir()
	configContent := `{
        "mcpServers": {
            "http-server": {
                "type": "http",
                "url": "http://localhost:8080/mcp",
                "enabled": true,
                "headers": {
                    "Authorization": "$AUTH_HEADER",
                    "X-Custom": "$(echo custom-value)"
                }
            }
        }
    }`
	configPath := tmpDir + "/mcp.json"
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewConfigLoaderForTest(tmpDir, "")
	result, err := loader.Load()

	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	server := result["http-server"]
	if server.Headers["Authorization"] != "Bearer token123" {
		t.Errorf("Headers[Authorization] = %q, want 'Bearer token123'", server.Headers["Authorization"])
	}
	if server.Headers["X-Custom"] != "custom-value" {
		t.Errorf("Headers[X-Custom] = %q, want 'custom-value'", server.Headers["X-Custom"])
	}
}

func TestInterpolateValue_Timeout(t *testing.T) {
	// Test that commands that timeout return the original string
	// instead of blocking indefinitely or crashing
	input := "$(sleep 10)"
	result := interpolateValue(input)

	// Should return the original string (unexpanded) due to timeout
	if result != input {
		t.Errorf("interpolateValue(%q) = %q, want %q (unchanged due to timeout)", input, result, input)
	}
}
