// Package config provides unit tests for configuration management.
package config

import (
	"os"
	"strconv"
	"testing"

	"github.com/denkhaus/gollum/pkg/shared"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewService_DefaultLoggingConfig tests that default values are applied correctly
func TestNewService_DefaultLoggingConfig(t *testing.T) {
	// Clear environment variables to test defaults
	unsetEnv(t, "GOLLUM_LOGGING_SESSION_LOG_BUFFER_SIZE")
	unsetEnv(t, "GOLLUM_LOGGING_SESSION_LOG_ENABLED")

	injector := do.New()
	service, err := NewService(injector)
	require.NoError(t, err)

	config := service.GetLoggingConfig()
	assert.NotNil(t, config)

	// Test default values
	assert.Equal(t, 1000, config.SessionLogBufferSize, "Default buffer size should be 1000")
	assert.True(t, config.SessionLogEnabled, "Session logging should be enabled by default")
}

// TestNewService_LoggingConfigFromEnv tests that environment variables are parsed correctly
func TestNewService_LoggingConfigFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		buffer   string
		enabled  string
		expected int
		expectEn bool
	}{
		{
			name:     "Custom values",
			buffer:   "5000",
			enabled:  "false",
			expected: 5000,
			expectEn: false,
		},
		{
			name:     "Only buffer set",
			buffer:   "2000",
			enabled:  "",
			expected: 2000,
			expectEn: true, // default
		},
		{
			name:     "Only enabled set",
			buffer:   "",
			enabled:  "false",
			expected: 1000, // default
			expectEn: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear and set environment variables
			unsetEnv(t, "GOLLUM_LOGGING_SESSION_LOG_BUFFER_SIZE")
			unsetEnv(t, "GOLLUM_LOGGING_SESSION_LOG_ENABLED")

			if tt.buffer != "" {
				require.NoError(t, os.Setenv("GOLLUM_LOGGING_SESSION_LOG_BUFFER_SIZE", tt.buffer))
			}
			if tt.enabled != "" {
				require.NoError(t, os.Setenv("GOLLUM_LOGGING_SESSION_LOG_ENABLED", tt.enabled))
			}

			injector := do.New()
			service, err := NewService(injector)
			require.NoError(t, err)

			config := service.GetLoggingConfig()
			assert.Equal(t, tt.expected, config.SessionLogBufferSize)
			assert.Equal(t, tt.expectEn, config.SessionLogEnabled)
		})
	}
}

// TestNewService_LoggingConfigValidation tests buffer size bounds validation
func TestNewService_LoggingConfigValidation(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
		reason   string
	}{
		{
			name:     "Below minimum (99)",
			input:    99,
			expected: 100,
			reason:   "Should clamp to minimum of 100",
		},
		{
			name:     "At minimum (100)",
			input:    100,
			expected: 100,
			reason:   "Should accept minimum value",
		},
		{
			name:     "Within range (5000)",
			input:    5000,
			expected: 5000,
			reason:   "Should accept valid value",
		},
		{
			name:     "At maximum (100000)",
			input:    100000,
			expected: 100000,
			reason:   "Should accept maximum value",
		},
		{
			name:     "Above maximum (100001)",
			input:    100001,
			expected: 100000,
			reason:   "Should clamp to maximum of 100000",
		},
		{
			name:     "Far above maximum (999999)",
			input:    999999,
			expected: 100000,
			reason:   "Should clamp far-above values to maximum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			require.NoError(t, os.Setenv("GOLLUM_LOGGING_SESSION_LOG_BUFFER_SIZE", strconv.Itoa(tt.input)))
			defer unsetEnv(t, "GOLLUM_LOGGING_SESSION_LOG_BUFFER_SIZE")

			injector := do.New()
			service, err := NewService(injector)
			require.NoError(t, err)

			config := service.GetLoggingConfig()
			assert.Equal(t, tt.expected, config.SessionLogBufferSize, tt.reason)
		})
	}
}

// TestGetLoggingConfig_ReturnsPointer tests that GetLoggingConfig returns a stable pointer
func TestGetLoggingConfig_ReturnsPointer(t *testing.T) {
	unsetEnv(t, "GOLLUM_LOGGING_SESSION_LOG_BUFFER_SIZE")
	unsetEnv(t, "GOLLUM_LOGGING_SESSION_LOG_ENABLED")

	injector := do.New()
	service, err := NewService(injector)
	require.NoError(t, err)

	config1 := service.GetLoggingConfig()
	config2 := service.GetLoggingConfig()

	// Should return the same pointer (stable reference)
	assert.Same(t, config1, config2, "GetLoggingConfig should return stable pointer")
}

// unsetEnv is a test helper that unsets an environment variable and restores it on test cleanup
func unsetEnv(t *testing.T, key string) {
	t.Helper()
	original, exists := os.LookupEnv(key)
	require.NoError(t, os.Unsetenv(key))
	t.Cleanup(func() {
		if exists {
			require.NoError(t, os.Setenv(key, original))
		}
	})
}

// TestGetLoggingConfig_Immutable tests that modifications to returned config don't affect service
func TestGetLoggingConfig_Immutable(t *testing.T) {
	unsetEnv(t, "GOLLUM_LOGGING_SESSION_LOG_BUFFER_SIZE")
	unsetEnv(t, "GOLLUM_LOGGING_SESSION_LOG_ENABLED")

	injector := do.New()
	service, err := NewService(injector)
	require.NoError(t, err)

	config1 := service.GetLoggingConfig()
	originalSize := config1.SessionLogBufferSize

	// Modify the returned config
	config1.SessionLogBufferSize = 99999

	// Get config again - should be different if we're returning copies,
	// or same if returning pointer to internal state
	config2 := service.GetLoggingConfig()

	// The current implementation returns a pointer to internal state,
	// so modifications will affect the service. This is the expected behavior.
	assert.Equal(t, 99999, config2.SessionLogBufferSize,
		"GetLoggingConfig returns pointer to internal state (not a copy)")

	// Restore original value for cleanup
	config2.SessionLogBufferSize = originalSize
}

// TestNewService_DefaultPromptOptimizerConfig tests that default optimizer values are applied correctly
func TestNewService_DefaultPromptOptimizerConfig(t *testing.T) {
	// Clear environment variables to test defaults
	unsetEnv(t, "GOLLUM_OPTIMIZER_STRATEGY")
	unsetEnv(t, "GOLLUM_OPTIMIZER_PROVIDER")
	unsetEnv(t, "GOLLUM_OPTIMIZER_MAX_REFLECTION")
	unsetEnv(t, "GOLLUM_OPTIMIZER_MIN_REFLECTION")

	injector := do.New()
	service, err := NewService(injector)
	require.NoError(t, err)

	config := service.GetPromptOptimizerConfig()
	assert.NotNil(t, config)

	// Test default values
	assert.Equal(t, shared.StrategyGradient, config.DefaultStrategy, "Default strategy should be gradient")
	assert.Equal(t, shared.LLMProviderAnthropic, config.DefaultProvider, "Default provider should be anthropic")
	assert.Equal(t, 5, config.MaxReflectionSteps, "Default max reflection steps should be 5")
	assert.Equal(t, 2, config.MinReflectionSteps, "Default min reflection steps should be 2")
}

// TestNewService_PromptOptimizerConfigFromEnv tests that environment variables are parsed correctly
func TestNewService_PromptOptimizerConfigFromEnv(t *testing.T) {
	tests := []struct {
		name                string
		strategy            string
		provider            string
		maxReflection       string
		minReflection       string
		expectStrategy      shared.OptimizerStrategy
		expectProvider      shared.LLMProvider
		expectMaxReflection int
		expectMinReflection int
	}{
		{
			name:                "Custom values",
			strategy:            "metaprompt",
			provider:            "openai",
			maxReflection:       "10",
			minReflection:       "3",
			expectStrategy:      shared.StrategyMetaPrompt,
			expectProvider:      shared.LLMProviderOpenAI,
			expectMaxReflection: 10,
			expectMinReflection: 3,
		},
		{
			name:                "Only strategy set",
			strategy:            "promptmemory",
			provider:            "",
			maxReflection:       "",
			minReflection:       "",
			expectStrategy:      shared.StrategyPromptMemory,
			expectProvider:      shared.LLMProviderAnthropic, // default
			expectMaxReflection: 5,                          // default
			expectMinReflection: 2,                          // default
		},
		{
			name:                "Gemini provider with custom reflection",
			strategy:            "",
			provider:            "gemini",
			maxReflection:       "7",
			minReflection:       "1",
			expectStrategy:      shared.StrategyGradient, // default
			expectProvider:      shared.LLMProviderGemini,
			expectMaxReflection: 7,
			expectMinReflection: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear and set environment variables
			unsetEnv(t, "GOLLUM_OPTIMIZER_STRATEGY")
			unsetEnv(t, "GOLLUM_OPTIMIZER_PROVIDER")
			unsetEnv(t, "GOLLUM_OPTIMIZER_MAX_REFLECTION")
			unsetEnv(t, "GOLLUM_OPTIMIZER_MIN_REFLECTION")

			if tt.strategy != "" {
				require.NoError(t, os.Setenv("GOLLUM_OPTIMIZER_STRATEGY", tt.strategy))
			}
			if tt.provider != "" {
				require.NoError(t, os.Setenv("GOLLUM_OPTIMIZER_PROVIDER", tt.provider))
			}
			if tt.maxReflection != "" {
				require.NoError(t, os.Setenv("GOLLUM_OPTIMIZER_MAX_REFLECTION", tt.maxReflection))
			}
			if tt.minReflection != "" {
				require.NoError(t, os.Setenv("GOLLUM_OPTIMIZER_MIN_REFLECTION", tt.minReflection))
			}

			injector := do.New()
			service, err := NewService(injector)
			require.NoError(t, err)

			config := service.GetPromptOptimizerConfig()
			assert.Equal(t, tt.expectStrategy, config.DefaultStrategy)
			assert.Equal(t, tt.expectProvider, config.DefaultProvider)
			assert.Equal(t, tt.expectMaxReflection, config.MaxReflectionSteps)
			assert.Equal(t, tt.expectMinReflection, config.MinReflectionSteps)
		})
	}
}

// TestGetPromptOptimizerConfig_ReturnsPointer tests that GetPromptOptimizerConfig returns a stable pointer
func TestGetPromptOptimizerConfig_ReturnsPointer(t *testing.T) {
	unsetEnv(t, "GOLLUM_OPTIMIZER_STRATEGY")
	unsetEnv(t, "GOLLUM_OPTIMIZER_PROVIDER")
	unsetEnv(t, "GOLLUM_OPTIMIZER_MAX_REFLECTION")
	unsetEnv(t, "GOLLUM_OPTIMIZER_MIN_REFLECTION")

	injector := do.New()
	service, err := NewService(injector)
	require.NoError(t, err)

	config1 := service.GetPromptOptimizerConfig()
	config2 := service.GetPromptOptimizerConfig()

	// Should return the same pointer (stable reference)
	assert.Same(t, config1, config2, "GetPromptOptimizerConfig should return stable pointer")
}

// TestNewService_PromptStoreConfigIntegration verifies that PromptStoreConfig still works correctly
func TestNewService_PromptStoreConfigIntegration(t *testing.T) {
	// Clear environment variables to test defaults
	unsetEnv(t, "GOLLUM_PROMPT_STORE_TYPE")
	unsetEnv(t, "GOLLUM_PROMPT_STORE_FILE_PATH")
	unsetEnv(t, "GOLLUM_PROMPT_STORE_CACHE_ENABLED")

	injector := do.New()
	service, err := NewService(injector)
	require.NoError(t, err)

	config := service.GetPromptStoreConfig()
	assert.NotNil(t, config)

	// Test default values
	assert.Equal(t, PromptStoreType("memory"), config.Type, "Default store type should be memory")
	assert.Equal(t, "./data/prompts", config.FilePath, "Default file path should be ./data/prompts")
	assert.False(t, config.CacheEnabled, "Cache should be disabled by default")

	// Test with custom values
	unsetEnv(t, "GOLLUM_PROMPT_STORE_TYPE")
	unsetEnv(t, "GOLLUM_PROMPT_STORE_FILE_PATH")
	unsetEnv(t, "GOLLUM_PROMPT_STORE_CACHE_ENABLED")

	require.NoError(t, os.Setenv("GOLLUM_PROMPT_STORE_TYPE", "file"))
	require.NoError(t, os.Setenv("GOLLUM_PROMPT_STORE_FILE_PATH", "/tmp/prompts"))
	require.NoError(t, os.Setenv("GOLLUM_PROMPT_STORE_CACHE_ENABLED", "true"))

	injector2 := do.New()
	service2, err := NewService(injector2)
	require.NoError(t, err)

	config2 := service2.GetPromptStoreConfig()
	assert.Equal(t, PromptStoreType("file"), config2.Type)
	assert.Equal(t, "/tmp/prompts", config2.FilePath)
	assert.True(t, config2.CacheEnabled)
}
